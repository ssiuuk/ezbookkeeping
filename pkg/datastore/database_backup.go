package datastore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/storage"
)

const (
	databaseBackupFileNamePrefix     = "ezbookkeeping-"
	databaseBackupFileNameSuffix     = ".db"
	databaseBackupFileNameTimeLayout = "20060102-150405"
	databaseBackupTempFileSuffix     = ".tmp"
	databaseBackupDirMode            = 0700
	databaseBackupFileMode           = 0600
)

var databaseBackupFileNamePattern = regexp.MustCompile(`^ezbookkeeping-\d{8}-\d{6}\.db$`)
var databaseAnyBackupFileNamePattern = regexp.MustCompile(`^ezbookkeeping-(\d{8}-\d{6})(` + regexp.QuoteMeta(databaseBeforeRestoreBackupMarker) + `)?\.db$`)

// DatabaseBackupFileInfo represents a database backup file in the backup directory
type DatabaseBackupFileInfo struct {
	FileName        string
	FilePath        string
	Size            int64
	BackupTime      time.Time
	IsBeforeRestore bool
}

// ErrDatabaseBackupNotSupported is returned when the current database type does not support built-in backup
var ErrDatabaseBackupNotSupported = errors.New("database backup is only supported for sqlite3 database")

// ErrDatabaseIsNotInitialized is returned when the data store has not been initialized
var ErrDatabaseIsNotInitialized = errors.New("database is not initialized")

// ErrDatabaseBackupPathIsEmpty is returned when the database backup directory is empty
var ErrDatabaseBackupPathIsEmpty = errors.New("database backup path is empty")

// GetDatabaseBackupFileName returns the backup file name for the specified time
func GetDatabaseBackupFileName(t time.Time) string {
	return databaseBackupFileNamePrefix + t.Format(databaseBackupFileNameTimeLayout) + databaseBackupFileNameSuffix
}

// IsDatabaseBackupFileName returns whether the specified file name matches the backup file naming pattern
func IsDatabaseBackupFileName(fileName string) bool {
	return databaseBackupFileNamePattern.MatchString(fileName)
}

// BackupDatabase takes a consistent snapshot of the current sqlite database into the backup directory,
// then deletes old backup files exceeding the keep count (0 means keeping all), and returns the backup file path
func BackupDatabase(c core.Context, config *settings.Config, backupDir string) (string, error) {
	if config.DatabaseConfig == nil || config.DatabaseConfig.DatabaseType != settings.Sqlite3DbType {
		return "", ErrDatabaseBackupNotSupported
	}

	if Container.UserStore == nil || Container.UserStore.Count() < 1 {
		return "", ErrDatabaseIsNotInitialized
	}

	backupFilePath, err := backupSqliteDatabase(c, Container.UserStore.Get(0), backupDir, time.Now())

	if err != nil {
		return "", err
	}

	log.Infof(c, "[database_backup.BackupDatabase] database has been backed up to \"%s\"", backupFilePath)

	deletedFiles, err := pruneDatabaseBackupFiles(backupDir, config.DatabaseBackupKeepCount)

	for i := 0; i < len(deletedFiles); i++ {
		log.Infof(c, "[database_backup.BackupDatabase] old database backup file \"%s\" has been deleted", deletedFiles[i])
	}

	if err != nil {
		log.Warnf(c, "[database_backup.BackupDatabase] failed to delete old database backup files, because %s", err.Error())
	}

	// a failed off-site copy only logs errors, the local backup file is kept and the backup still succeeds
	copyDatabaseBackupToOffsite(c, config, backupDir, backupFilePath, deletedFiles)

	return backupFilePath, nil
}

// ListDatabaseBackupFiles returns all backup files (including the backups taken before restoring) in the backup directory, oldest first
func ListDatabaseBackupFiles(backupDir string) ([]*DatabaseBackupFileInfo, error) {
	if backupDir == "" {
		return nil, ErrDatabaseBackupPathIsEmpty
	}

	entries, err := os.ReadDir(backupDir)

	if err != nil {
		return nil, err
	}

	backupFiles := make([]*DatabaseBackupFileInfo, 0, len(entries))

	for i := 0; i < len(entries); i++ {
		if !entries[i].Type().IsRegular() {
			continue
		}

		matches := databaseAnyBackupFileNamePattern.FindStringSubmatch(entries[i].Name())

		if matches == nil {
			continue
		}

		backupTime, err := time.ParseInLocation(databaseBackupFileNameTimeLayout, matches[1], time.Local)

		if err != nil {
			continue
		}

		fileInfo, err := entries[i].Info()

		if err != nil {
			return nil, err
		}

		backupFiles = append(backupFiles, &DatabaseBackupFileInfo{
			FileName:        entries[i].Name(),
			FilePath:        filepath.Join(backupDir, entries[i].Name()),
			Size:            fileInfo.Size(),
			BackupTime:      backupTime,
			IsBeforeRestore: matches[2] != "",
		})
	}

	sort.Slice(backupFiles, func(i, j int) bool {
		if !backupFiles[i].BackupTime.Equal(backupFiles[j].BackupTime) {
			return backupFiles[i].BackupTime.Before(backupFiles[j].BackupTime)
		}

		return backupFiles[i].FileName < backupFiles[j].FileName
	})

	return backupFiles, nil
}

func backupSqliteDatabase(c core.Context, db *Database, backupDir string, now time.Time) (string, error) {
	return backupSqliteDatabaseToFile(c, db, backupDir, GetDatabaseBackupFileName(now))
}

func backupSqliteDatabaseToFile(c core.Context, db *Database, backupDir string, backupFileName string) (string, error) {
	if db.databaseType != settings.Sqlite3DbType {
		return "", ErrDatabaseBackupNotSupported
	}

	if backupDir == "" {
		return "", ErrDatabaseBackupPathIsEmpty
	}

	backupDir, err := filepath.Abs(backupDir)

	if err != nil {
		return "", err
	}

	// backups hold every user's data, so a new backup directory is only accessible to the server user
	if err = os.MkdirAll(backupDir, databaseBackupDirMode); err != nil {
		return "", err
	}

	backupFilePath := filepath.Join(backupDir, backupFileName)
	tempFilePath := backupFilePath + databaseBackupTempFileSuffix

	if _, err = os.Stat(backupFilePath); err == nil {
		return "", fmt.Errorf("database backup file \"%s\" already exists", backupFilePath)
	}

	// VACUUM INTO requires that the target file does not exist or is empty
	if err = os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// create the empty target with private permissions before writing, VACUUM INTO keeps the mode of an existing empty file
	tempFile, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, databaseBackupFileMode)

	if err != nil {
		return "", err
	}

	if err = tempFile.Close(); err != nil {
		_ = os.Remove(tempFilePath)
		return "", err
	}

	if _, err = db.NewSession(c).Exec("VACUUM INTO " + quoteSqliteStringLiteral(tempFilePath)); err != nil {
		_ = os.Remove(tempFilePath)
		return "", err
	}

	if err = os.Rename(tempFilePath, backupFilePath); err != nil {
		_ = os.Remove(tempFilePath)
		return "", err
	}

	return backupFilePath, nil
}

// pruneDatabaseBackupFiles deletes the oldest backup files so that only keepCount files are kept (0 means keeping all),
// only files matching the backup file naming pattern are considered, and returns the deleted file paths
func pruneDatabaseBackupFiles(backupDir string, keepCount uint32) ([]string, error) {
	if keepCount == 0 {
		return nil, nil
	}

	entries, err := os.ReadDir(backupDir)

	if err != nil {
		return nil, err
	}

	backupFileNames := make([]string, 0, len(entries))

	for i := 0; i < len(entries); i++ {
		if entries[i].Type().IsRegular() && IsDatabaseBackupFileName(entries[i].Name()) {
			backupFileNames = append(backupFileNames, entries[i].Name())
		}
	}

	if len(backupFileNames) <= int(keepCount) {
		return nil, nil
	}

	// file names contain a fixed width timestamp, so lexical order is chronological order
	sort.Strings(backupFileNames)

	deleteCount := len(backupFileNames) - int(keepCount)
	deletedFiles := make([]string, 0, deleteCount)
	var lastErr error

	for i := 0; i < deleteCount; i++ {
		filePath := filepath.Join(backupDir, backupFileNames[i])

		if err := os.Remove(filePath); err != nil {
			lastErr = err
			continue
		}

		deletedFiles = append(deletedFiles, filePath)
	}

	return deletedFiles, lastErr
}

// copyDatabaseBackupToOffsite copies the new backup file to the backup copy directory and the object storage if enabled,
// and deletes the old copies there, errors are only logged
func copyDatabaseBackupToOffsite(c core.Context, config *settings.Config, backupDir string, backupFilePath string, deletedLocalFiles []string) {
	if config.DatabaseBackupCopyPath != "" {
		if isSameDirectory(backupDir, config.DatabaseBackupCopyPath) {
			log.Warnf(c, "[database_backup.copyDatabaseBackupToOffsite] database backup copy path \"%s\" is the same as the backup path, skip copying", config.DatabaseBackupCopyPath)
		} else {
			copyFilePath, deletedCopyFiles, err := copyDatabaseBackupFileToDirectory(backupFilePath, config.DatabaseBackupCopyPath, config.DatabaseBackupKeepCount)

			if copyFilePath != "" {
				log.Infof(c, "[database_backup.copyDatabaseBackupToOffsite] database backup has been copied to \"%s\"", copyFilePath)
			}

			for i := 0; i < len(deletedCopyFiles); i++ {
				log.Infof(c, "[database_backup.copyDatabaseBackupToOffsite] old database backup copy \"%s\" has been deleted", deletedCopyFiles[i])
			}

			if err != nil {
				log.Errorf(c, "[database_backup.copyDatabaseBackupToOffsite] failed to copy database backup to \"%s\", because %s", config.DatabaseBackupCopyPath, err.Error())
			}
		}
	}

	if config.DatabaseBackupCopyToStorage {
		if !storage.Container.IsDatabaseBackupStorageEnabled() {
			log.Errorf(c, "[database_backup.copyDatabaseBackupToOffsite] failed to upload database backup to object storage, because the object storage is not initialized")
			return
		}

		backupFileName := filepath.Base(backupFilePath)

		if err := uploadDatabaseBackupFileToStorage(c, backupFilePath, backupFileName); err != nil {
			log.Errorf(c, "[database_backup.copyDatabaseBackupToOffsite] failed to upload database backup \"%s\" to object storage, because %s", backupFileName, err.Error())
			return
		}

		log.Infof(c, "[database_backup.copyDatabaseBackupToOffsite] database backup \"%s\" has been uploaded to object storage", backupFileName)

		// the object storage cannot list files, so only the copies of the local backup files just deleted are deleted there
		for i := 0; i < len(deletedLocalFiles); i++ {
			deletedFileName := filepath.Base(deletedLocalFiles[i])

			if err := storage.Container.DeleteDatabaseBackup(c, deletedFileName); err != nil {
				log.Warnf(c, "[database_backup.copyDatabaseBackupToOffsite] failed to delete old database backup \"%s\" from object storage, because %s", deletedFileName, err.Error())
			} else {
				log.Infof(c, "[database_backup.copyDatabaseBackupToOffsite] old database backup \"%s\" has been deleted from object storage", deletedFileName)
			}
		}
	}
}

func uploadDatabaseBackupFileToStorage(c core.Context, backupFilePath string, backupFileName string) error {
	backupFile, err := os.Open(backupFilePath)

	if err != nil {
		return err
	}

	defer backupFile.Close()

	return storage.Container.SaveDatabaseBackup(c, backupFileName, backupFile)
}

// copyDatabaseBackupFileToDirectory copies the backup file into the target directory with private permissions,
// then deletes old backup files there exceeding the keep count, and returns the copied file path and the deleted file paths
func copyDatabaseBackupFileToDirectory(backupFilePath string, targetDir string, keepCount uint32) (string, []string, error) {
	targetDir, err := filepath.Abs(targetDir)

	if err != nil {
		return "", nil, err
	}

	if err = os.MkdirAll(targetDir, databaseBackupDirMode); err != nil {
		return "", nil, err
	}

	targetFilePath := filepath.Join(targetDir, filepath.Base(backupFilePath))

	if err = copyFileAtomically(backupFilePath, targetFilePath, databaseBackupFileMode); err != nil {
		return "", nil, err
	}

	deletedFiles, err := pruneDatabaseBackupFiles(targetDir, keepCount)

	return targetFilePath, deletedFiles, err
}

// copyFileAtomically copies the source file to a temporary file next to the target file, flushes it to disk,
// then renames it to the target file path, so the target file is either the old one or a complete copy
func copyFileAtomically(sourceFilePath string, targetFilePath string, fileMode os.FileMode) error {
	sourceFile, err := os.Open(sourceFilePath)

	if err != nil {
		return err
	}

	defer sourceFile.Close()

	tempFilePath := targetFilePath + databaseBackupTempFileSuffix

	if err = os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	tempFile, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)

	if err != nil {
		return err
	}

	if _, err = io.Copy(tempFile, sourceFile); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
		return err
	}

	if err = tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
		return err
	}

	if err = tempFile.Close(); err != nil {
		_ = os.Remove(tempFilePath)
		return err
	}

	if err = os.Rename(tempFilePath, targetFilePath); err != nil {
		_ = os.Remove(tempFilePath)
		return err
	}

	syncDirectory(filepath.Dir(targetFilePath))

	return nil
}

// syncDirectory flushes the directory entry changes to disk, it is best effort since not every platform supports it
func syncDirectory(dir string) {
	d, err := os.Open(dir)

	if err != nil {
		return
	}

	_ = d.Sync()
	_ = d.Close()
}

func isSameDirectory(dir1 string, dir2 string) bool {
	absDir1, err1 := filepath.Abs(dir1)
	absDir2, err2 := filepath.Abs(dir2)

	if err1 != nil || err2 != nil {
		return false
	}

	return isSameFile(absDir1, absDir2)
}

func quoteSqliteStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
