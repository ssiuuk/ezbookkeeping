//go:build !windows

package storage

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestPrivateLocalFileSystemObjectStorageUnderUmask022(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	config := &settings.Config{
		StorageType:         settings.LocalFileSystemObjectStorageType,
		LocalFileSystemPath: dir,
	}

	objectStorage, err := newDatabaseBackupObjectStorage(config)
	assert.Nil(t, err)

	c := core.NewNullContext()
	assert.Nil(t, objectStorage.Save(c, "ezbookkeeping-20261018-030000.db", newByteSliceObject([]byte("backup"))))

	dirInfo, err := os.Stat(filepath.Join(dir, databaseBackupPathPrefix))
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(dir, databaseBackupPathPrefix, "ezbookkeeping-20261018-030000.db"))
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	assert.Nil(t, objectStorage.Delete(c, "ezbookkeeping-20261018-030000.db"))
	_, err = os.Stat(filepath.Join(dir, databaseBackupPathPrefix, "ezbookkeeping-20261018-030000.db"))
	assert.True(t, os.IsNotExist(err))
}
