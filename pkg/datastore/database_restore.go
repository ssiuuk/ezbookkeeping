package datastore

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

const (
	databaseBeforeRestoreBackupMarker = "-before-restore"
)

// sqlite creates these files next to the database file, they belong to the old database and must not be applied to the restored one
var sqliteDatabaseSideFileSuffixes = []string{"-journal", "-wal", "-shm"}

// ErrDatabaseRestoreNotSupported is returned when the current database type does not support built-in restore
var ErrDatabaseRestoreNotSupported = errors.New("database restore is only supported for sqlite3 database")

// ErrDatabaseRestoreFileIsCurrentDatabase is returned when the file to restore from is the current database file
var ErrDatabaseRestoreFileIsCurrentDatabase = errors.New("the file to restore from is the current database file")

// DatabaseRestoreResult represents what has been done by restoring the database
type DatabaseRestoreResult struct {
	DatabaseFilePath            string
	RestoredFromFilePath        string
	BeforeRestoreBackupFilePath string
	RemovedSideFilePaths        []string
}

// GetDatabaseBeforeRestoreBackupFileName returns the file name of the backup taken before restoring at the specified time
func GetDatabaseBeforeRestoreBackupFileName(t time.Time) string {
	return databaseBackupFileNamePrefix + t.Format(databaseBackupFileNameTimeLayout) + databaseBeforeRestoreBackupMarker + databaseBackupFileNameSuffix
}

// RestoreDatabase replaces the current sqlite database file with the specified backup file,
// the current database is backed up into the backup directory first so that the restore can be undone,
// the data store is closed afterwards and must not be used anymore
func RestoreDatabase(c core.Context, config *settings.Config, restoreFromFilePath string, backupDir string) (*DatabaseRestoreResult, error) {
	if config.DatabaseConfig == nil || config.DatabaseConfig.DatabaseType != settings.Sqlite3DbType {
		return nil, ErrDatabaseRestoreNotSupported
	}

	if Container.UserStore == nil || Container.UserStore.Count() < 1 {
		return nil, ErrDatabaseIsNotInitialized
	}

	result, err := restoreSqliteDatabase(c, Container.UserStore.Get(0), config.DatabaseConfig.DatabasePath, restoreFromFilePath, backupDir, time.Now())

	if err != nil {
		return result, err
	}

	log.Infof(c, "[database_restore.RestoreDatabase] database \"%s\" has been restored from \"%s\"", result.DatabaseFilePath, result.RestoredFromFilePath)

	return result, nil
}

// VerifySqliteDatabaseFile returns an error if the specified file is not an intact ezBookkeeping sqlite database
func VerifySqliteDatabaseFile(filePath string) error {
	fileInfo, err := os.Stat(filePath)

	if err != nil {
		return err
	}

	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("\"%s\" is not a regular file", filePath)
	}

	// sqlite treats an empty file as a valid empty database
	if fileInfo.Size() < 1 {
		return fmt.Errorf("\"%s\" is empty", filePath)
	}

	db, err := sql.Open("sqlite3", "file:"+filePath+"?mode=ro")

	if err != nil {
		return err
	}

	defer db.Close()

	var integrity string

	if err = db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return fmt.Errorf("\"%s\" is not a valid sqlite database, because %s", filePath, err.Error())
	}

	if integrity != "ok" {
		return fmt.Errorf("\"%s\" failed the sqlite integrity check: %s", filePath, integrity)
	}

	var userTableCount int

	if err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'user'").Scan(&userTableCount); err != nil {
		return fmt.Errorf("\"%s\" cannot be read, because %s", filePath, err.Error())
	}

	if userTableCount < 1 {
		return fmt.Errorf("\"%s\" is not an ezBookkeeping database, because it has no user table", filePath)
	}

	return nil
}

func restoreSqliteDatabase(c core.Context, db *Database, databasePath string, restoreFromFilePath string, backupDir string, now time.Time) (*DatabaseRestoreResult, error) {
	if db.databaseType != settings.Sqlite3DbType {
		return nil, ErrDatabaseRestoreNotSupported
	}

	databasePath, err := filepath.Abs(databasePath)

	if err != nil {
		return nil, err
	}

	restoreFromFilePath, err = filepath.Abs(restoreFromFilePath)

	if err != nil {
		return nil, err
	}

	if isSameFile(databasePath, restoreFromFilePath) {
		return nil, ErrDatabaseRestoreFileIsCurrentDatabase
	}

	// verify before touching anything, an invalid file leaves the current database untouched
	if err = VerifySqliteDatabaseFile(restoreFromFilePath); err != nil {
		return nil, err
	}

	result := &DatabaseRestoreResult{
		DatabaseFilePath:     databasePath,
		RestoredFromFilePath: restoreFromFilePath,
	}

	// keep the current database so that the restore can be undone by restoring this file
	result.BeforeRestoreBackupFilePath, err = backupSqliteDatabaseToFile(c, db, backupDir, GetDatabaseBeforeRestoreBackupFileName(now))

	if err != nil {
		return nil, fmt.Errorf("failed to back up the current database before restoring, because %s", err.Error())
	}

	// all connections must be closed before replacing the file, otherwise they would keep using the old database (and windows cannot replace an open file)
	if err = db.engineGroup.Close(); err != nil {
		return result, err
	}

	// the journal or wal files of the old database would be applied to the restored database, so remove them before replacing
	for i := 0; i < len(sqliteDatabaseSideFileSuffixes); i++ {
		sideFilePath := databasePath + sqliteDatabaseSideFileSuffixes[i]

		if err = os.Remove(sideFilePath); err == nil {
			result.RemovedSideFilePaths = append(result.RemovedSideFilePaths, sideFilePath)
		} else if !os.IsNotExist(err) {
			return result, err
		}
	}

	// copy into a temporary file in the database directory and rename it, so the database file is never partially written
	if err = copyFileAtomically(restoreFromFilePath, databasePath, databaseBackupFileMode); err != nil {
		return result, err
	}

	return result, nil
}

func isSameFile(path1 string, path2 string) bool {
	if path1 == path2 {
		return true
	}

	info1, err1 := os.Stat(path1)
	info2, err2 := os.Stat(path2)

	return err1 == nil && err2 == nil && os.SameFile(info1, info2)
}
