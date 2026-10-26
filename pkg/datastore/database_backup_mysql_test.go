package datastore

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestQuoteMysqlStringLiteral(t *testing.T) {
	assert.Equal(t, "'abc'", quoteMysqlStringLiteral("abc"))
	assert.Equal(t, "'it\\'s'", quoteMysqlStringLiteral("it's"))
	assert.Equal(t, "'a\\\\b\\nc\\rd\\0e\\Zf\\\"g'", quoteMysqlStringLiteral("a\\b\nc\rd\x00e\x1af\"g"))
	assert.Equal(t, "'家庭開支'", quoteMysqlStringLiteral("家庭開支"))
}

func TestQuoteMysqlIdentifier(t *testing.T) {
	assert.Equal(t, "`user`", quoteMysqlIdentifier("user"))
	assert.Equal(t, "`a``b`", quoteMysqlIdentifier("a`b"))
}

func TestFormatMysqlValue(t *testing.T) {
	assert.Equal(t, "NULL", formatMysqlValue(nil, "BIGINT"))
	assert.Equal(t, "-123", formatMysqlValue([]byte("-123"), "BIGINT"))
	assert.Equal(t, "1.5e-7", formatMysqlValue([]byte("1.5e-7"), "DOUBLE"))
	assert.Equal(t, "'1; DROP TABLE x'", formatMysqlValue([]byte("1; DROP TABLE x"), "BIGINT"))
	assert.Equal(t, "0x00ff", formatMysqlValue([]byte{0x00, 0xff}, "BLOB"))
	assert.Equal(t, "''", formatMysqlValue([]byte{}, "VARBINARY"))
	assert.Equal(t, "''", formatMysqlValue([]byte{}, "VARCHAR"))
}

func TestIsDatabaseBackupFileName_MysqlBackup(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
	assert.Equal(t, "ezbookkeeping-20260102-030405.sql.gz", getMysqlDatabaseBackupFileName(ts))
	assert.True(t, IsDatabaseBackupFileName(getMysqlDatabaseBackupFileName(ts)))
	assert.False(t, IsDatabaseBackupFileName(getMysqlDatabaseBeforeRestoreBackupFileName(ts)))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping-20260102-030405.sql"))
	assert.False(t, IsDatabaseBackupFileName("ezbookkeeping-20260102-030405.sql.gz.tmp"))
}

func TestListDatabaseBackupFiles_MysqlBackup(t *testing.T) {
	dir := t.TempDir()
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20260102-030000.sql.gz"), []byte("x"), 0600))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "ezbookkeeping-20260103-030000-before-restore.sql.gz"), []byte("x"), 0600))

	backupFiles, err := ListDatabaseBackupFiles(dir)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(backupFiles))
	assert.False(t, backupFiles[0].IsBeforeRestore)
	assert.True(t, backupFiles[1].IsBeforeRestore)
}

func writeGzipTestFile(t *testing.T, filePath string, content string) {
	file, err := os.Create(filePath)
	assert.Nil(t, err)

	writer := gzip.NewWriter(file)
	_, err = writer.Write([]byte(content))
	assert.Nil(t, err)
	assert.Nil(t, writer.Close())
	assert.Nil(t, file.Close())
}

func TestVerifyMysqlDatabaseBackupFile(t *testing.T) {
	dir := t.TempDir()
	validContent := mysqlDatabaseBackupHeaderLine + "\nSET NAMES utf8mb4;\nCREATE TABLE `user` (`uid` bigint(20) NOT NULL);\n" + mysqlDatabaseBackupFooterLine + "\n"

	validFilePath := filepath.Join(dir, "valid.sql.gz")
	writeGzipTestFile(t, validFilePath, validContent)
	assert.Nil(t, VerifyMysqlDatabaseBackupFile(validFilePath))
	assert.Nil(t, VerifyDatabaseBackupFile(settings.MySqlDbType, validFilePath))

	statements := make([]string, 0)
	assert.Nil(t, readMysqlDatabaseBackupStatements(validFilePath, func(statement string) error {
		statements = append(statements, statement)
		return nil
	}))
	assert.Equal(t, []string{"SET NAMES utf8mb4;", "CREATE TABLE `user` (`uid` bigint(20) NOT NULL);"}, statements)

	truncatedFilePath := filepath.Join(dir, "truncated.sql.gz")
	writeGzipTestFile(t, truncatedFilePath, validContent[0:len(validContent)-len(mysqlDatabaseBackupFooterLine)-1])
	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(truncatedFilePath))

	noUserTableFilePath := filepath.Join(dir, "no-user.sql.gz")
	writeGzipTestFile(t, noUserTableFilePath, mysqlDatabaseBackupHeaderLine+"\nCREATE TABLE `other` (`id` int);\n"+mysqlDatabaseBackupFooterLine+"\n")
	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(noUserTableFilePath))

	otherFilePath := filepath.Join(dir, "other.sql.gz")
	writeGzipTestFile(t, otherFilePath, "-- MySQL dump\n"+mysqlDatabaseBackupFooterLine+"\n")
	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(otherFilePath))

	plainFilePath := filepath.Join(dir, "plain.sql.gz")
	assert.Nil(t, os.WriteFile(plainFilePath, []byte(validContent), 0600))
	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(plainFilePath))

	afterFooterFilePath := filepath.Join(dir, "after-footer.sql.gz")
	writeGzipTestFile(t, afterFooterFilePath, validContent+"DROP TABLE `user`;\n")
	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(afterFooterFilePath))

	assert.NotNil(t, VerifyMysqlDatabaseBackupFile(filepath.Join(dir, "missing.sql.gz")))
}

// the mysql tests need a mysql or mariadb database which can be dropped and created freely, e.g.
// EZBOOKKEEPING_TEST_MYSQL_HOST=127.0.0.1:3306 EZBOOKKEEPING_TEST_MYSQL_NAME=ezb_test EZBOOKKEEPING_TEST_MYSQL_USER=ezb EZBOOKKEEPING_TEST_MYSQL_PASSWORD=ezbpass
func openMysqlTestDatabase(t *testing.T) *Database {
	host := os.Getenv("EZBOOKKEEPING_TEST_MYSQL_HOST")

	if host == "" {
		t.Skip("EZBOOKKEEPING_TEST_MYSQL_HOST is not set")
	}

	db, err := initializeDatabase(&settings.DatabaseConfig{
		DatabaseType:      settings.MySqlDbType,
		DatabaseHost:      host,
		DatabaseName:      os.Getenv("EZBOOKKEEPING_TEST_MYSQL_NAME"),
		DatabaseUser:      os.Getenv("EZBOOKKEEPING_TEST_MYSQL_USER"),
		DatabasePassword:  os.Getenv("EZBOOKKEEPING_TEST_MYSQL_PASSWORD"),
		MaxIdleConnection: 2,
	})
	assert.Nil(t, err)

	return db
}

func initializeMysqlTestDatabase(t *testing.T) *Database {
	db := openMysqlTestDatabase(t)
	c := core.NewNullContext()
	var err error

	for _, statement := range []string{
		"DROP TABLE IF EXISTS `user`",
		"DROP TABLE IF EXISTS `test_item`",
		"CREATE TABLE `user` (`uid` BIGINT NOT NULL, `name` VARCHAR(64) NOT NULL, PRIMARY KEY (`uid`)) DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE `test_item` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, `name` MEDIUMTEXT NULL, `amount` DECIMAL(20,4) NULL, `rate` DOUBLE NULL, `flag` TINYINT(1) NOT NULL DEFAULT 0, `data` BLOB NULL, `created` DATETIME NULL, PRIMARY KEY (`id`)) DEFAULT CHARSET=utf8mb4",
	} {
		_, err = db.NewSession(c).Exec(statement)
		assert.Nil(t, err)
	}

	return db
}

func TestBackupAndRestoreMysqlDatabase(t *testing.T) {
	db := initializeMysqlTestDatabase(t)
	defer db.engineGroup.Close()

	c := core.NewNullContext()
	trickyName := "it's a \"test\"\\\nnew line\r\x1a 家庭開支 😀"

	_, err := db.NewSession(c).Exec("INSERT INTO `user` (`uid`, `name`) VALUES (1, 'sam')")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("INSERT INTO `test_item` (`id`, `name`, `amount`, `rate`, `flag`, `data`, `created`) VALUES (1, ?, -1234.5600, 1.5e-7, 1, ?, '2026-10-26 01:02:03'), (18446744073709551615, NULL, NULL, NULL, 0, NULL, NULL), (5, '', 0, 0, 0, '', NULL)", trickyName, []byte{0x00, 0x01, 0xfe, 0xff})
	assert.Nil(t, err)

	// enough rows to need several insert statements
	longText := strings.Repeat("x", 100*1024)

	for i := 0; i < 25; i++ {
		_, err = db.NewSession(c).Exec("INSERT INTO `test_item` (`id`, `name`) VALUES (?, ?)", 100+i, longText)
		assert.Nil(t, err)
	}

	backupDir := filepath.Join(t.TempDir(), "backups")
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.Local)
	backupFilePath, err := backupMysqlDatabaseToFile(c, db, backupDir, getMysqlDatabaseBackupFileName(now))
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(backupDir, "ezbookkeeping-20261026-030000.sql.gz"), backupFilePath)
	assert.Nil(t, VerifyMysqlDatabaseBackupFile(backupFilePath))

	fileInfo, err := os.Stat(backupFilePath)
	assert.Nil(t, err)
	assert.Equal(t, os.FileMode(databaseBackupFileMode), fileInfo.Mode().Perm())

	insertStatementCount := 0
	assert.Nil(t, readMysqlDatabaseBackupStatements(backupFilePath, func(statement string) error {
		if strings.HasPrefix(statement, "INSERT INTO `test_item`") {
			insertStatementCount++
			assert.LessOrEqual(t, len(statement), mysqlDatabaseBackupMaxInsertStatementSize+len(longText)+100)
		}
		return nil
	}))
	assert.Greater(t, insertStatementCount, 1)

	// a second backup with the same timestamp must not overwrite the existing file
	_, err = backupMysqlDatabaseToFile(c, db, backupDir, getMysqlDatabaseBackupFileName(now))
	assert.NotNil(t, err)

	_, err = db.NewSession(c).Exec("UPDATE `user` SET `name` = 'changed'")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("DELETE FROM `test_item` WHERE `id` = 1")
	assert.Nil(t, err)
	_, err = db.NewSession(c).Exec("INSERT INTO `test_item` (`id`, `name`) VALUES (6, 'added after backup')")
	assert.Nil(t, err)

	result, err := restoreMysqlDatabase(c, db, "ezb_test", backupFilePath, backupDir, now.Add(time.Hour))
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(backupDir, "ezbookkeeping-20261026-040000-before-restore.sql.gz"), result.BeforeRestoreBackupFilePath)
	assert.Nil(t, VerifyMysqlDatabaseBackupFile(result.BeforeRestoreBackupFilePath))

	// restoring closes the data store, so the restored data is read with a new one
	restoredDb := openMysqlTestDatabase(t)
	defer restoredDb.engineGroup.Close()

	sqlDB := restoredDb.engineGroup.Master().DB().DB

	var userName string
	assert.Nil(t, sqlDB.QueryRow("SELECT `name` FROM `user` WHERE `uid` = 1").Scan(&userName))
	assert.Equal(t, "sam", userName)

	var count int
	assert.Nil(t, sqlDB.QueryRow("SELECT COUNT(*) FROM `test_item`").Scan(&count))
	assert.Equal(t, 28, count)

	var name string
	var amount string
	var rate float64
	var data []byte
	var created string
	assert.Nil(t, sqlDB.QueryRow("SELECT `name`, `amount`, `rate`, `data`, DATE_FORMAT(`created`, '%Y-%m-%d %H:%i:%s') FROM `test_item` WHERE `id` = 1").Scan(&name, &amount, &rate, &data, &created))
	assert.Equal(t, trickyName, name)
	assert.Equal(t, "-1234.5600", amount)
	assert.Equal(t, 1.5e-7, rate)
	assert.Equal(t, []byte{0x00, 0x01, 0xfe, 0xff}, data)
	assert.Equal(t, "2026-10-26 01:02:03", created)

	var nullName *string
	assert.Nil(t, sqlDB.QueryRow("SELECT `name` FROM `test_item` WHERE `id` = 18446744073709551615").Scan(&nullName))
	assert.Nil(t, nullName)

	assert.Nil(t, sqlDB.QueryRow("SELECT `name` FROM `test_item` WHERE `id` = 105").Scan(&name))
	assert.Equal(t, longText, name)

	assert.Nil(t, sqlDB.QueryRow("SELECT COUNT(*) FROM `test_item` WHERE `id` = 6").Scan(&count))
	assert.Equal(t, 0, count)

	// restoring the backup taken before restoring undoes the restore
	undoResult, err := restoreMysqlDatabase(c, restoredDb, "ezb_test", result.BeforeRestoreBackupFilePath, backupDir, now.Add(2*time.Hour))
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(backupDir, "ezbookkeeping-20261026-050000-before-restore.sql.gz"), undoResult.BeforeRestoreBackupFilePath)

	undoneDb := openMysqlTestDatabase(t)
	defer undoneDb.engineGroup.Close()

	sqlDB = undoneDb.engineGroup.Master().DB().DB

	assert.Nil(t, sqlDB.QueryRow("SELECT `name` FROM `user` WHERE `uid` = 1").Scan(&userName))
	assert.Equal(t, "changed", userName)

	assert.Nil(t, sqlDB.QueryRow("SELECT COUNT(*) FROM `test_item`").Scan(&count))
	assert.Equal(t, 28, count)

	assert.Nil(t, sqlDB.QueryRow("SELECT COUNT(*) FROM `test_item` WHERE `id` = 1").Scan(&count))
	assert.Equal(t, 0, count)

	assert.Nil(t, sqlDB.QueryRow("SELECT `name` FROM `test_item` WHERE `id` = 6").Scan(&name))
	assert.Equal(t, "added after backup", name)
}

func TestRestoreMysqlDatabaseRefusesInvalidFile(t *testing.T) {
	db := initializeMysqlTestDatabase(t)
	defer db.engineGroup.Close()

	c := core.NewNullContext()
	_, err := db.NewSession(c).Exec("INSERT INTO `user` (`uid`, `name`) VALUES (1, 'sam')")
	assert.Nil(t, err)

	dir := t.TempDir()
	invalidFilePath := filepath.Join(dir, "invalid.sql.gz")
	writeGzipTestFile(t, invalidFilePath, mysqlDatabaseBackupHeaderLine+"\nDROP TABLE IF EXISTS `user`;\n")

	_, err = restoreMysqlDatabase(c, db, "ezb_test", invalidFilePath, filepath.Join(dir, "backups"), time.Now())
	assert.NotNil(t, err)

	// nothing is done, not even the backup before restoring
	_, err = os.Stat(filepath.Join(dir, "backups"))
	assert.True(t, os.IsNotExist(err))

	var userName string
	assert.Nil(t, db.engineGroup.Master().DB().DB.QueryRow("SELECT `name` FROM `user` WHERE `uid` = 1").Scan(&userName))
	assert.Equal(t, "sam", userName)
}

func TestBackupMysqlDatabaseRejectsSqlite(t *testing.T) {
	_, err := backupMysqlDatabaseToFile(core.NewNullContext(), &Database{databaseType: settings.Sqlite3DbType}, t.TempDir(), "x.sql.gz")
	assert.Equal(t, ErrDatabaseBackupNotSupported, err)

	_, err = restoreMysqlDatabase(core.NewNullContext(), &Database{databaseType: settings.Sqlite3DbType}, "", "x.sql.gz", t.TempDir(), time.Now())
	assert.Equal(t, ErrDatabaseRestoreNotSupported, err)
}
