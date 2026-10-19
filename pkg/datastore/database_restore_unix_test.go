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
)

func TestRestoreSqliteDatabaseIsPrivateUnderUmask022(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	db, databasePath := newTestSqliteDatabase(t, dir)
	c := core.NewNullContext()

	backupFilePath, err := backupSqliteDatabase(c, db, backupDir, time.Date(2026, 10, 18, 3, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	result, err := restoreSqliteDatabase(c, db, databasePath, backupFilePath, backupDir, time.Date(2026, 10, 19, 10, 0, 0, 0, time.Local))
	assert.Nil(t, err)

	fileInfo, err := os.Stat(databasePath)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	fileInfo, err = os.Stat(result.BeforeRestoreBackupFilePath)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}

func TestCopyDatabaseBackupFileToDirectoryIsPrivateUnderUmask022(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	backupFilePath := filepath.Join(dir, "ezbookkeeping-20261018-030000.db")
	copyDir := filepath.Join(dir, "copies")

	// the source file is readable by others, the copy must still be private
	assert.Nil(t, os.WriteFile(backupFilePath, []byte("x"), 0644))
	assert.Nil(t, os.Chmod(backupFilePath, 0644))

	copyFilePath, _, err := copyDatabaseBackupFileToDirectory(backupFilePath, copyDir, 30)
	assert.Nil(t, err)

	dirInfo, err := os.Stat(copyDir)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(copyFilePath)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}
