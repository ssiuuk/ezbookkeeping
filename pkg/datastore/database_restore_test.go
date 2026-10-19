package datastore

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// newTestSqliteDatabase creates a sqlite database with a user table and two rows in the specified directory,
// the database is closed before the temporary directory is removed (windows cannot remove open files)
func newTestSqliteDatabase(t *testing.T, dir string) (*Database, string) {
	databasePath := filepath.Join(dir, "ezbookkeeping.db")

	db, err := initializeDatabase(&settings.DatabaseConfig{
		DatabaseType:      settings.Sqlite3DbType,
		DatabasePath:      databasePath,
		MaxIdleConnection: 2,
	})
	assert.Nil(t, err)
	t.Cleanup(func() {
		_ = db.engineGroup.Close()
	})

	c := core.NewNullContext()
	_, err = db.NewSession(c).Exec("CREATE TABLE user (uid INTEGER PRIMARY KEY, username TEXT NOT NULL)")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("INSERT INTO user (uid, username) VALUES (1, 'first'), (2, 'second')")
	assert.Nil(t, err)

	return db, databasePath
}

func queryTestUserNames(t *testing.T, databasePath string) []string {
	db, err := sql.Open("sqlite3", "file:"+databasePath+"?mode=ro")
	assert.Nil(t, err)
	defer db.Close()

	rows, err := db.Query("SELECT username FROM user ORDER BY uid")
	assert.Nil(t, err)
	defer rows.Close()

	names := make([]string, 0)

	for rows.Next() {
		var name string
		assert.Nil(t, rows.Scan(&name))
		names = append(names, name)
	}

	assert.Nil(t, rows.Err())

	return names
}

func TestGetDatabaseBeforeRestoreBackupFileName(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
	assert.Equal(t, "ezbookkeeping-20260102-030405-before-restore.db", GetDatabaseBeforeRestoreBackupFileName(ts))

	// backups taken before restoring are never deleted by pruning
	assert.False(t, IsDatabaseBackupFileName(GetDatabaseBeforeRestoreBackupFileName(ts)))
}

func TestRestoreSqliteDatabase(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	db, databasePath := newTestSqliteDatabase(t, dir)
	c := core.NewNullContext()

	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	_, err = db.NewSession(c).Exec("INSERT INTO user (uid, username) VALUES (3, 'third')")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("UPDATE user SET username = 'changed' WHERE uid = 1")
	assert.Nil(t, err)

	// a stale side file of the old database must be gone (sqlite itself may already delete it when closing)
	walFilePath := databasePath + "-wal"
	assert.Nil(t, os.WriteFile(walFilePath, []byte("stale"), 0600))

	result, err := restoreSqliteDatabase(c, db, databasePath, backupFilePath, backupDir, time.Date(2026, 10, 19, 10, 0, 0, 0, time.Local))
	assert.Nil(t, err)
	assert.Equal(t, databasePath, result.DatabaseFilePath)
	assert.Equal(t, backupFilePath, result.RestoredFromFilePath)
	assert.Equal(t, filepath.Join(backupDir, "ezbookkeeping-20261019-100000-before-restore.db"), result.BeforeRestoreBackupFilePath)

	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(databasePath + databaseBackupTempFileSuffix)
	assert.True(t, os.IsNotExist(err))

	assert.Equal(t, []string{"first", "second"}, queryTestUserNames(t, databasePath))

	// the database before restoring is kept so the restore can be undone
	assert.Nil(t, VerifySqliteDatabaseFile(result.BeforeRestoreBackupFilePath))
	assert.Equal(t, []string{"changed", "second", "third"}, queryTestUserNames(t, result.BeforeRestoreBackupFilePath))

	// the backup file itself is not modified
	assert.Equal(t, []string{"first", "second"}, queryTestUserNames(t, backupFilePath))
}

func TestRestoreSqliteDatabaseRefusesInvalidFile(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	db, databasePath := newTestSqliteDatabase(t, dir)
	c := core.NewNullContext()

	garbageFilePath := filepath.Join(dir, "garbage.db")
	assert.Nil(t, os.WriteFile(garbageFilePath, []byte("this is not a sqlite database, just some text that is long enough to have a header"), 0600))

	emptyFilePath := filepath.Join(dir, "empty.db")
	assert.Nil(t, os.WriteFile(emptyFilePath, nil, 0600))

	// a valid sqlite database which is not an ezBookkeeping database
	otherFilePath := filepath.Join(dir, "other.db")
	otherDb, err := sql.Open("sqlite3", "file:"+otherFilePath+"?mode=rwc")
	assert.Nil(t, err)
	_, err = otherDb.Exec("CREATE TABLE other_item (id INTEGER PRIMARY KEY)")
	assert.Nil(t, err)
	assert.Nil(t, otherDb.Close())

	invalidFilePaths := []string{
		garbageFilePath,
		emptyFilePath,
		otherFilePath,
		filepath.Join(dir, "not-exists.db"),
		dir,
		databasePath,
	}

	for _, invalidFilePath := range invalidFilePaths {
		result, err := restoreSqliteDatabase(c, db, databasePath, invalidFilePath, backupDir, time.Now())
		assert.NotNil(t, err, invalidFilePath)
		assert.Nil(t, result, invalidFilePath)
	}

	_, err = restoreSqliteDatabase(c, db, databasePath, databasePath, backupDir, time.Now())
	assert.Equal(t, ErrDatabaseRestoreFileIsCurrentDatabase, err)

	// the current database is untouched and still usable, and no backup has been taken
	var count int
	_, err = db.NewSession(c).SQL("SELECT COUNT(*) FROM user").Get(&count)
	assert.Nil(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, []string{"first", "second"}, queryTestUserNames(t, databasePath))

	_, err = os.Stat(backupDir)
	assert.True(t, os.IsNotExist(err))
}

func TestRestoreSqliteDatabaseRejectsNonSqlite(t *testing.T) {
	_, err := restoreSqliteDatabase(core.NewNullContext(), &Database{databaseType: settings.MySqlDbType}, "a.db", "b.db", t.TempDir(), time.Now())
	assert.Equal(t, ErrDatabaseRestoreNotSupported, err)

	_, err = RestoreDatabase(core.NewNullContext(), &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.PostgresDbType},
	}, "b.db", t.TempDir())
	assert.Equal(t, ErrDatabaseRestoreNotSupported, err)
}

func TestListDatabaseBackupFiles(t *testing.T) {
	dir := t.TempDir()

	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20261019-100000-before-restore.db"), []byte("xyz"), 0600))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20261018-030000.db"), []byte("x"), 0600))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20261019-030000.db"), []byte("xy"), 0600))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20261020-030000.db.tmp"), []byte("x"), 0600))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0600))

	backupFiles, err := ListDatabaseBackupFiles(dir)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(backupFiles))

	assert.Equal(t, "ezbookkeeping-20261018-030000.db", backupFiles[0].FileName)
	assert.Equal(t, filepath.Join(dir, "ezbookkeeping-20261018-030000.db"), backupFiles[0].FilePath)
	assert.Equal(t, int64(1), backupFiles[0].Size)
	assert.Equal(t, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local), backupFiles[0].BackupTime)
	assert.False(t, backupFiles[0].IsBeforeRestore)

	assert.Equal(t, "ezbookkeeping-20261019-030000.db", backupFiles[1].FileName)
	assert.False(t, backupFiles[1].IsBeforeRestore)

	assert.Equal(t, "ezbookkeeping-20261019-100000-before-restore.db", backupFiles[2].FileName)
	assert.Equal(t, int64(3), backupFiles[2].Size)
	assert.True(t, backupFiles[2].IsBeforeRestore)

	_, err = ListDatabaseBackupFiles(filepath.Join(dir, "not-exists"))
	assert.NotNil(t, err)

	_, err = ListDatabaseBackupFiles("")
	assert.Equal(t, ErrDatabaseBackupPathIsEmpty, err)
}

func TestCopyDatabaseBackupFileToDirectory(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	copyDir := filepath.Join(dir, "offsite", "copies")
	assert.Nil(t, os.MkdirAll(backupDir, 0700))

	backupNames := []string{
		"ezbookkeeping-20261016-030000.db",
		"ezbookkeeping-20261017-030000.db",
		"ezbookkeeping-20261018-030000.db",
	}

	for i, name := range backupNames {
		assert.Nil(t, os.WriteFile(filepath.Join(backupDir, name), []byte(name), 0600))

		copyFilePath, deleted, err := copyDatabaseBackupFileToDirectory(filepath.Join(backupDir, name), copyDir, 2)
		assert.Nil(t, err)
		assert.Equal(t, filepath.Join(copyDir, name), copyFilePath)

		if i < 2 {
			assert.Empty(t, deleted)
		} else {
			assert.Equal(t, []string{filepath.Join(copyDir, backupNames[0])}, deleted)
		}

		content, err := os.ReadFile(copyFilePath)
		assert.Nil(t, err)
		assert.Equal(t, name, string(content))
	}

	entries, err := os.ReadDir(copyDir)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, backupNames[1], entries[0].Name())
	assert.Equal(t, backupNames[2], entries[1].Name())

	// copying the same file again replaces the copy
	assert.Nil(t, os.WriteFile(filepath.Join(backupDir, backupNames[2]), []byte("new content"), 0600))
	_, _, err = copyDatabaseBackupFileToDirectory(filepath.Join(backupDir, backupNames[2]), copyDir, 2)
	assert.Nil(t, err)
	content, err := os.ReadFile(filepath.Join(copyDir, backupNames[2]))
	assert.Nil(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestBackupDatabaseCopiesToOffsiteDirectory(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	copyDir := filepath.Join(dir, "copies")
	db, _ := newTestSqliteDatabase(t, dir)
	c := core.NewNullContext()

	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	copyDatabaseBackupToOffsite(c, &settings.Config{
		DatabaseBackupCopyPath:  copyDir,
		DatabaseBackupKeepCount: 30,
	}, backupDir, backupFilePath, nil)

	copyFilePath := filepath.Join(copyDir, filepath.Base(backupFilePath))
	assert.Nil(t, VerifySqliteDatabaseFile(copyFilePath))
	assert.Equal(t, []string{"first", "second"}, queryTestUserNames(t, copyFilePath))
}

func TestBackupDatabaseOffsiteCopyFailureKeepsLocalBackup(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	db, _ := newTestSqliteDatabase(t, dir)
	c := core.NewNullContext()

	// the copy path is an existing regular file, so the directory cannot be created
	blockingFilePath := filepath.Join(dir, "copies")
	assert.Nil(t, os.WriteFile(blockingFilePath, []byte("x"), 0600))

	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	// the object storage is not initialized in tests, so uploading also fails and must only be logged
	copyDatabaseBackupToOffsite(c, &settings.Config{
		DatabaseBackupCopyPath:      blockingFilePath,
		DatabaseBackupCopyToStorage: true,
		DatabaseBackupKeepCount:     30,
	}, backupDir, backupFilePath, nil)

	assert.Nil(t, VerifySqliteDatabaseFile(backupFilePath))

	// the copy path being the backup path is skipped
	copyDatabaseBackupToOffsite(c, &settings.Config{
		DatabaseBackupCopyPath:  backupDir,
		DatabaseBackupKeepCount: 1,
	}, backupDir, backupFilePath, nil)

	backupFiles, err := ListDatabaseBackupFiles(backupDir)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backupFiles))
}
