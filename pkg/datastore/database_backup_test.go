package datastore

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestGetDatabaseBackupFileName(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
	assert.Equal(t, "ezbookkeeping-20260102-030405.db", GetDatabaseBackupFileName(ts))
	assert.True(t, IsDatabaseBackupFileName(GetDatabaseBackupFileName(ts)))
}

func TestIsDatabaseBackupFileName(t *testing.T) {
	assert.True(t, IsDatabaseBackupFileName("ezbookkeeping-20261018-235959.db"))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping.db"))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping-20261018-235959.db.tmp"))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping-2026101-235959.db"))
	assert.False(t, IsDatabaseBackupFileName("other-20261018-235959.db"))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping-20261018-235959.db.bak"))
	assert.False(t, IsDatabaseBackupFileName("xezbookkeeping-20261018-235959.db"))
}

func TestQuoteSqliteStringLiteral(t *testing.T) {
	assert.Equal(t, "'/tmp/a.db'", quoteSqliteStringLiteral("/tmp/a.db"))
	assert.Equal(t, "'/tmp/it''s/a.db'", quoteSqliteStringLiteral("/tmp/it's/a.db"))
	assert.Equal(t, "''''''", quoteSqliteStringLiteral("''"))
}

func TestPruneDatabaseBackupFiles(t *testing.T) {
	dir := t.TempDir()
	backupNames := []string{
		"ezbookkeeping-20260101-030000.db",
		"ezbookkeeping-20260102-030000.db",
		"ezbookkeeping-20260103-030000.db",
		"ezbookkeeping-20260104-030000.db",
		"ezbookkeeping-20260105-030000.db",
	}
	otherNames := []string{
		"ezbookkeeping.db",
		"ezbookkeeping-20250101-030000.db.tmp",
		"notes.txt",
	}

	// create in non-chronological order to make sure sorting does not depend on creation order
	for _, name := range []string{backupNames[3], backupNames[0], backupNames[4], backupNames[2], backupNames[1]} {
		assert.Nil(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644))
	}

	for _, name := range otherNames {
		assert.Nil(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644))
	}

	// a directory matching the pattern must be ignored
	assert.Nil(t, os.Mkdir(filepath.Join(dir, "ezbookkeeping-20240101-030000.db"), 0755))

	deleted, err := pruneDatabaseBackupFiles(dir, 0)
	assert.Nil(t, err)
	assert.Empty(t, deleted)

	deleted, err = pruneDatabaseBackupFiles(dir, 10)
	assert.Nil(t, err)
	assert.Empty(t, deleted)

	deleted, err = pruneDatabaseBackupFiles(dir, 3)
	assert.Nil(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, backupNames[0]),
		filepath.Join(dir, backupNames[1]),
	}, deleted)

	entries, err := os.ReadDir(dir)
	assert.Nil(t, err)

	remaining := make([]string, 0, len(entries))

	for _, entry := range entries {
		remaining = append(remaining, entry.Name())
	}

	expected := append([]string{"ezbookkeeping-20240101-030000.db"}, backupNames[2:]...)
	expected = append(expected, otherNames...)
	sort.Strings(expected)
	sort.Strings(remaining)
	assert.Equal(t, expected, remaining)

	deleted, err = pruneDatabaseBackupFiles(dir, 1)
	assert.Nil(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, backupNames[2]),
		filepath.Join(dir, backupNames[3]),
	}, deleted)
}

func TestBackupSqliteDatabase(t *testing.T) {
	dir := t.TempDir()
	// a directory name containing a single quote verifies the SQL escaping
	backupDir := filepath.Join(dir, "it's backups")

	db, err := initializeDatabase(&settings.DatabaseConfig{
		DatabaseType:      settings.Sqlite3DbType,
		DatabasePath:      filepath.Join(dir, "ezbookkeeping.db"),
		MaxIdleConnection: 2,
	})
	assert.Nil(t, err)
	defer db.engineGroup.Close()

	c := core.NewNullContext()
	_, err = db.NewSession(c).Exec("CREATE TABLE test_item (id INTEGER PRIMARY KEY, name TEXT NOT NULL)")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("INSERT INTO test_item (id, name) VALUES (1, 'first'), (2, 'second')")
	assert.Nil(t, err)

	now := time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local)
	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, now)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(backupDir, "ezbookkeeping-20261018-030000.db"), backupFilePath)

	_, err = os.Stat(backupFilePath + databaseBackupTempFileSuffix)
	assert.True(t, os.IsNotExist(err))

	// a second backup with the same timestamp must not overwrite the existing file
	_, err = backupSqliteDatabase(c, db, backupDir, now)
	assert.NotNil(t, err)

	backupDb, err := sql.Open("sqlite3", "file:"+backupFilePath+"?mode=ro")
	assert.Nil(t, err)
	defer backupDb.Close()

	var integrity string
	assert.Nil(t, backupDb.QueryRow("PRAGMA integrity_check").Scan(&integrity))
	assert.Equal(t, "ok", integrity)

	var count int
	assert.Nil(t, backupDb.QueryRow("SELECT COUNT(*) FROM test_item").Scan(&count))
	assert.Equal(t, 2, count)

	var name string
	assert.Nil(t, backupDb.QueryRow("SELECT name FROM test_item WHERE id = 2").Scan(&name))
	assert.Equal(t, "second", name)
}

func TestBackupSqliteDatabaseRejectsNonSqlite(t *testing.T) {
	_, err := backupSqliteDatabase(core.NewNullContext(), &Database{databaseType: settings.MySqlDbType}, t.TempDir(), time.Now())
	assert.Equal(t, ErrDatabaseBackupNotSupported, err)

	_, err = BackupDatabase(core.NewNullContext(), &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.PostgresDbType},
	}, t.TempDir())
	assert.Equal(t, ErrDatabaseBackupNotSupported, err)
}
