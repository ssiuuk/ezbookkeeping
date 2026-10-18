//go:build !windows

package datastore

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestBackupSqliteDatabaseIsPrivateUnderUmask022(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	databasePath := filepath.Join(dir, "ezbookkeeping.db")
	backupDir := filepath.Join(dir, "backups")

	db, err := initializeDatabase(&settings.DatabaseConfig{
		DatabaseType:      settings.Sqlite3DbType,
		DatabasePath:      databasePath,
		MaxIdleConnection: 2,
	})
	assert.Nil(t, err)
	defer db.engineGroup.Close()

	c := core.NewNullContext()
	_, err = db.NewSession(c).Exec("CREATE TABLE test_item (id INTEGER PRIMARY KEY)")
	assert.Nil(t, err)
	assert.Nil(t, os.Chmod(databasePath, 0600))

	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	dirInfo, err := os.Stat(backupDir)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(backupFilePath)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
	assert.True(t, fileInfo.Size() > 0)
}

func TestBackupSqliteDatabaseTempFileIsCreatedPrivate(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()

	db, err := initializeDatabase(&settings.DatabaseConfig{
		DatabaseType:      settings.Sqlite3DbType,
		DatabasePath:      filepath.Join(dir, "ezbookkeeping.db"),
		MaxIdleConnection: 2,
	})
	assert.Nil(t, err)
	defer db.engineGroup.Close()

	// a backup into an existing empty private file keeps its mode, which is what the temporary file relies on
	target := filepath.Join(dir, "target.db")
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, databaseBackupFileMode)
	assert.Nil(t, err)
	assert.Nil(t, f.Close())

	c := core.NewNullContext()
	_, err = db.NewSession(c).Exec("CREATE TABLE test_item (id INTEGER PRIMARY KEY)")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("VACUUM INTO " + quoteSqliteStringLiteral(target))
	assert.Nil(t, err)

	fileInfo, err := os.Stat(target)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
	assert.True(t, fileInfo.Size() > 0)
}
