package datastore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

const (
	databaseBackupFileNamePrefix     = "ezbookkeeping-"
	databaseBackupFileNameSuffix     = ".db"
	databaseBackupFileNameTimeLayout = "20060102-150405"
	databaseBackupTempFileSuffix     = ".tmp"
)

var databaseBackupFileNamePattern = regexp.MustCompile(`^ezbookkeeping-\d{8}-\d{6}\.db$`)

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

	return backupFilePath, nil
}

func backupSqliteDatabase(c core.Context, db *Database, backupDir string, now time.Time) (string, error) {
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

	if err = os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	backupFilePath := filepath.Join(backupDir, GetDatabaseBackupFileName(now))
	tempFilePath := backupFilePath + databaseBackupTempFileSuffix

	if _, err = os.Stat(backupFilePath); err == nil {
		return "", fmt.Errorf("database backup file \"%s\" already exists", backupFilePath)
	}

	// VACUUM INTO requires that the target file does not exist or is empty
	if err = os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
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

func quoteSqliteStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
