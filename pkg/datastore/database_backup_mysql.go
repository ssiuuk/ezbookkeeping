package datastore

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

const (
	mysqlDatabaseBackupHeaderLine = "-- ezBookkeeping MySQL database backup"
	mysqlDatabaseBackupFooterLine = "-- ezBookkeeping MySQL database backup end"

	// each insert statement is kept well below the default max_allowed_packet of mysql and mariadb
	mysqlDatabaseBackupMaxInsertStatementSize = 1024 * 1024
)

// these statements are written at the beginning of the backup file, so that restoring is not affected by the session settings of the server
var mysqlDatabaseBackupSessionStatements = []string{
	"SET NAMES utf8mb4;",
	"SET FOREIGN_KEY_CHECKS=0;",
	"SET UNIQUE_CHECKS=0;",
	"SET SQL_MODE='NO_AUTO_VALUE_ON_ZERO';",
}

var mysqlNumericColumnTypes = map[string]bool{
	"TINYINT":   true,
	"SMALLINT":  true,
	"MEDIUMINT": true,
	"INT":       true,
	"INTEGER":   true,
	"BIGINT":    true,
	"DECIMAL":   true,
	"FLOAT":     true,
	"DOUBLE":    true,
	"YEAR":      true,
}

var mysqlBinaryColumnTypes = map[string]bool{
	"BINARY":     true,
	"VARBINARY":  true,
	"TINYBLOB":   true,
	"BLOB":       true,
	"MEDIUMBLOB": true,
	"LONGBLOB":   true,
	"BIT":        true,
	"GEOMETRY":   true,
}

// ErrMysqlDatabaseBackupFileInvalid is returned when the file to restore from is not a complete ezBookkeeping mysql backup
var ErrMysqlDatabaseBackupFileInvalid = errors.New("the file is not a complete ezBookkeeping mysql database backup")

// backupMysqlDatabaseToFile writes all tables of the current mysql database as sql statements into a gzip file in the backup directory,
// the data of all tables is read in one transaction, so the backup is consistent without locking the tables
func backupMysqlDatabaseToFile(c core.Context, db *Database, backupDir string, backupFileName string) (string, error) {
	if db.databaseType != settings.MySqlDbType {
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

	if err = os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	tempFile, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, databaseBackupFileMode)

	if err != nil {
		return "", err
	}

	err = writeMysqlDatabaseBackup(c, db.engineGroup.Master().DB().DB, tempFile)

	if err == nil {
		err = tempFile.Sync()
	}

	if closeErr := tempFile.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		_ = os.Remove(tempFilePath)
		return "", err
	}

	if err = os.Rename(tempFilePath, backupFilePath); err != nil {
		_ = os.Remove(tempFilePath)
		return "", err
	}

	syncDirectory(backupDir)

	return backupFilePath, nil
}

func writeMysqlDatabaseBackup(c context.Context, sqlDB *sql.DB, writer io.Writer) error {
	conn, err := sqlDB.Conn(c)

	if err != nil {
		return err
	}

	defer conn.Close()

	// without SESSION, the isolation level only applies to the next transaction, so the pooled connection keeps its settings
	if _, err = conn.ExecContext(c, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
		return err
	}

	if _, err = conn.ExecContext(c, "START TRANSACTION WITH CONSISTENT SNAPSHOT, READ ONLY"); err != nil {
		return err
	}

	// the transaction only reads, so it is always rolled back
	defer conn.ExecContext(context.Background(), "ROLLBACK") // nolint: errcheck

	tableNames, err := getMysqlTableNames(c, conn)

	if err != nil {
		return err
	}

	gzipWriter := gzip.NewWriter(writer)
	bufferedWriter := bufio.NewWriter(gzipWriter)

	lines := make([]string, 0, 2+len(mysqlDatabaseBackupSessionStatements))
	lines = append(lines, mysqlDatabaseBackupHeaderLine, "-- Created at "+time.Now().Format(time.RFC3339))
	lines = append(lines, mysqlDatabaseBackupSessionStatements...)

	for i := 0; i < len(lines); i++ {
		if _, err = bufferedWriter.WriteString(lines[i] + "\n"); err != nil {
			return err
		}
	}

	for i := 0; i < len(tableNames); i++ {
		if err = writeMysqlTableBackup(c, conn, tableNames[i], bufferedWriter); err != nil {
			return fmt.Errorf("cannot back up table \"%s\", because %s", tableNames[i], err.Error())
		}
	}

	// restoring refuses a file without this line, so a truncated backup is never restored
	if _, err = bufferedWriter.WriteString(mysqlDatabaseBackupFooterLine + "\n"); err != nil {
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		return err
	}

	return gzipWriter.Close()
}

func getMysqlTableNames(c context.Context, conn *sql.Conn) ([]string, error) {
	rows, err := conn.QueryContext(c, "SHOW FULL TABLES WHERE Table_type = 'BASE TABLE'")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tableNames := make([]string, 0)

	for rows.Next() {
		var tableName, tableType string

		if err = rows.Scan(&tableName, &tableType); err != nil {
			return nil, err
		}

		tableNames = append(tableNames, tableName)
	}

	return tableNames, rows.Err()
}

func writeMysqlTableBackup(c context.Context, conn *sql.Conn, tableName string, writer *bufio.Writer) error {
	var createTableName, createTableStatement string

	if err := conn.QueryRowContext(c, "SHOW CREATE TABLE "+quoteMysqlIdentifier(tableName)).Scan(&createTableName, &createTableStatement); err != nil {
		return err
	}

	// every statement is on one line, so restoring can execute the file line by line
	createTableStatement = strings.ReplaceAll(strings.ReplaceAll(createTableStatement, "\r", " "), "\n", " ")

	if _, err := writer.WriteString("DROP TABLE IF EXISTS " + quoteMysqlIdentifier(tableName) + ";\n" + createTableStatement + ";\n"); err != nil {
		return err
	}

	rows, err := conn.QueryContext(c, "SELECT * FROM "+quoteMysqlIdentifier(tableName))

	if err != nil {
		return err
	}

	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()

	if err != nil {
		return err
	}

	columnNames := make([]string, len(columnTypes))
	columnKinds := make([]string, len(columnTypes))

	for i := 0; i < len(columnTypes); i++ {
		columnNames[i] = quoteMysqlIdentifier(columnTypes[i].Name())
		columnKinds[i] = strings.TrimPrefix(strings.ToUpper(columnTypes[i].DatabaseTypeName()), "UNSIGNED ")
	}

	insertPrefix := "INSERT INTO " + quoteMysqlIdentifier(tableName) + " (" + strings.Join(columnNames, ",") + ") VALUES "
	values := make([]sql.RawBytes, len(columnTypes))
	scanArgs := make([]any, len(columnTypes))

	for i := 0; i < len(values); i++ {
		scanArgs[i] = &values[i]
	}

	var statement strings.Builder

	for rows.Next() {
		if err = rows.Scan(scanArgs...); err != nil {
			return err
		}

		var rowValues strings.Builder
		rowValues.WriteString("(")

		for i := 0; i < len(values); i++ {
			if i > 0 {
				rowValues.WriteString(",")
			}

			rowValues.WriteString(formatMysqlValue(values[i], columnKinds[i]))
		}

		rowValues.WriteString(")")

		if statement.Len() > 0 && statement.Len()+rowValues.Len() > mysqlDatabaseBackupMaxInsertStatementSize {
			if _, err = writer.WriteString(statement.String() + ";\n"); err != nil {
				return err
			}

			statement.Reset()
		}

		if statement.Len() < 1 {
			statement.WriteString(insertPrefix)
		} else {
			statement.WriteString(",")
		}

		statement.WriteString(rowValues.String())
	}

	if err = rows.Err(); err != nil {
		return err
	}

	if statement.Len() > 0 {
		if _, err = writer.WriteString(statement.String() + ";\n"); err != nil {
			return err
		}
	}

	return nil
}

func formatMysqlValue(value sql.RawBytes, columnKind string) string {
	if value == nil {
		return "NULL"
	}

	if mysqlBinaryColumnTypes[columnKind] {
		if len(value) < 1 {
			return "''"
		}

		return "0x" + hex.EncodeToString(value)
	}

	if mysqlNumericColumnTypes[columnKind] && isMysqlNumericLiteral(value) {
		return string(value)
	}

	return quoteMysqlStringLiteral(string(value))
}

func isMysqlNumericLiteral(value []byte) bool {
	if len(value) < 1 {
		return false
	}

	for i := 0; i < len(value); i++ {
		ch := value[i]

		if (ch < '0' || ch > '9') && ch != '-' && ch != '+' && ch != '.' && ch != 'e' && ch != 'E' {
			return false
		}
	}

	return true
}

func quoteMysqlIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// quoteMysqlStringLiteral escapes the string the same way as mysqldump, the backup file sets a sql mode without NO_BACKSLASH_ESCAPES
func quoteMysqlStringLiteral(s string) string {
	var builder strings.Builder
	builder.Grow(len(s) + 2)
	builder.WriteByte('\'')

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0:
			builder.WriteString("\\0")
		case '\n':
			builder.WriteString("\\n")
		case '\r':
			builder.WriteString("\\r")
		case '\\':
			builder.WriteString("\\\\")
		case '\'':
			builder.WriteString("\\'")
		case '"':
			builder.WriteString("\\\"")
		case 0x1a:
			builder.WriteString("\\Z")
		default:
			builder.WriteByte(s[i])
		}
	}

	builder.WriteByte('\'')

	return builder.String()
}

// VerifyMysqlDatabaseBackupFile returns an error if the specified file is not a complete ezBookkeeping mysql backup
func VerifyMysqlDatabaseBackupFile(filePath string) error {
	return readMysqlDatabaseBackupStatements(filePath, nil)
}

// readMysqlDatabaseBackupStatements checks the whole backup file and then calls the handler with every statement,
// so that a damaged or truncated file is refused before any statement is executed
func readMysqlDatabaseBackupStatements(filePath string, handler func(statement string) error) error {
	fileInfo, err := os.Stat(filePath)

	if err != nil {
		return err
	}

	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("\"%s\" is not a regular file", filePath)
	}

	hasUserTable := false
	completed := false

	err = forEachMysqlDatabaseBackupLine(filePath, func(lineIndex int, line string) error {
		if lineIndex == 0 && line != mysqlDatabaseBackupHeaderLine {
			return ErrMysqlDatabaseBackupFileInvalid
		}

		if completed {
			return ErrMysqlDatabaseBackupFileInvalid
		}

		if line == mysqlDatabaseBackupFooterLine {
			completed = true
		} else if strings.HasPrefix(line, "CREATE TABLE `user` (") {
			hasUserTable = true
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrMysqlDatabaseBackupFileInvalid) {
			return fmt.Errorf("\"%s\" is not a complete ezBookkeeping mysql database backup", filePath)
		}

		return fmt.Errorf("\"%s\" cannot be read, because %s", filePath, err.Error())
	}

	if !completed {
		return fmt.Errorf("\"%s\" is not a complete ezBookkeeping mysql database backup, it may be truncated", filePath)
	}

	if !hasUserTable {
		return fmt.Errorf("\"%s\" is not an ezBookkeeping database backup, because it has no user table", filePath)
	}

	if handler == nil {
		return nil
	}

	return forEachMysqlDatabaseBackupLine(filePath, func(lineIndex int, line string) error {
		if line == "" || strings.HasPrefix(line, "--") {
			return nil
		}

		return handler(line)
	})
}

func forEachMysqlDatabaseBackupLine(filePath string, handler func(lineIndex int, line string) error) error {
	file, err := os.Open(filePath)

	if err != nil {
		return err
	}

	defer file.Close()

	gzipReader, err := gzip.NewReader(file)

	if err != nil {
		return ErrMysqlDatabaseBackupFileInvalid
	}

	defer gzipReader.Close()

	reader := bufio.NewReader(gzipReader)

	for lineIndex := 0; ; lineIndex++ {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			if line != "" {
				// the last line of a complete backup always ends with a line break
				return ErrMysqlDatabaseBackupFileInvalid
			}

			return nil
		}

		if err != nil {
			return err
		}

		if err = handler(lineIndex, strings.TrimSuffix(line, "\n")); err != nil {
			return err
		}
	}
}

// restoreMysqlDatabaseFromFile executes all statements of the backup file on one connection,
// each table in the backup is dropped and recreated, the tables which are not in the backup are not touched
func restoreMysqlDatabaseFromFile(c context.Context, sqlDB *sql.DB, filePath string) error {
	conn, err := sqlDB.Conn(c)

	if err != nil {
		return err
	}

	defer conn.Close()

	return readMysqlDatabaseBackupStatements(filePath, func(statement string) error {
		if _, err := conn.ExecContext(c, statement); err != nil {
			if len(statement) > 100 {
				statement = statement[0:100] + "..."
			}

			return fmt.Errorf("cannot execute \"%s\", because %s", statement, err.Error())
		}

		return nil
	})
}
