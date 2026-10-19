package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// Database represents the database command
var Database = &cli.Command{
	Name:  "database",
	Usage: "ezBookkeeping database maintenance",
	Commands: []*cli.Command{
		{
			Name:   "update",
			Usage:  "Update database structure",
			Action: bindAction(updateDatabaseStructure),
		},
		{
			Name:   "backup",
			Usage:  "Back up the sqlite database once",
			Action: bindAction(backupDatabase),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "path",
					Aliases:  []string{"p"},
					Required: false,
					Usage:    "Backup directory path (default is the database_backup_path in the [cron] section of the config file)",
				},
			},
			Commands: []*cli.Command{
				{
					Name:   "list",
					Usage:  "List the sqlite database backup files",
					Action: bindAction(listDatabaseBackups),
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "path",
							Aliases:  []string{"p"},
							Required: false,
							Usage:    "Backup directory path (default is the database_backup_path in the [cron] section of the config file)",
						},
					},
				},
			},
		},
		{
			Name:   "restore",
			Usage:  "Restore the sqlite database from a backup file, the server must be stopped first",
			Action: bindAction(restoreDatabase),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "file",
					Aliases:  []string{"f"},
					Required: true,
					Usage:    "Backup file path to restore from",
				},
				&cli.StringFlag{
					Name:     "path",
					Aliases:  []string{"p"},
					Required: false,
					Usage:    "Directory path to save the backup of the current database before restoring (default is the database_backup_path in the [cron] section of the config file)",
				},
				&cli.BoolFlag{
					Name:     "yes",
					Aliases:  []string{"y"},
					Required: false,
					Usage:    "Restore without asking for confirmation",
				},
			},
		},
	},
}

var errDatabaseRestoreCancelled = errors.New("database restore is cancelled")

func updateDatabaseStructure(c *core.CliContext) error {
	_, err := initializeSystem(c)

	if err != nil {
		return err
	}

	log.CliInfof(c, "[database.updateDatabaseStructure] starting maintaining")

	err = updateAllDatabaseTablesStructure(c)

	if err != nil {
		log.CliErrorf(c, "[database.updateDatabaseStructure] update database table structure failed, because %s", err.Error())
		return err
	}

	log.CliInfof(c, "[database.updateDatabaseStructure] all tables maintained successfully")
	return nil
}

func backupDatabase(c *core.CliContext) error {
	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	backupDir, err := getDatabaseBackupDir(c, config.DatabaseBackupPath)

	if err != nil {
		log.CliErrorf(c, "[database.backupDatabase] cannot resolve backup path, because %s", err.Error())
		return err
	}

	backupFilePath, err := datastore.BackupDatabase(c, config, backupDir)

	if err != nil {
		log.CliErrorf(c, "[database.backupDatabase] back up database failed, because %s", err.Error())
		return err
	}

	log.CliInfof(c, "[database.backupDatabase] database has been backed up to \"%s\"", backupFilePath)
	return nil
}

func listDatabaseBackups(c *core.CliContext) error {
	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	backupDir, err := getDatabaseBackupDir(c, config.DatabaseBackupPath)

	if err != nil {
		log.CliErrorf(c, "[database.listDatabaseBackups] cannot resolve backup path, because %s", err.Error())
		return err
	}

	backupFiles, err := datastore.ListDatabaseBackupFiles(backupDir)

	if err != nil {
		log.CliErrorf(c, "[database.listDatabaseBackups] cannot list backup files in \"%s\", because %s", backupDir, err.Error())
		return err
	}

	if len(backupFiles) < 1 {
		log.CliInfof(c, "[database.listDatabaseBackups] there are no backup files in \"%s\"", backupDir)
		return nil
	}

	for i := 0; i < len(backupFiles); i++ {
		if i > 0 {
			fmt.Printf("---\n")
		}

		fmt.Printf("[File] %s\n", backupFiles[i].FilePath)
		fmt.Printf("[Time] %s\n", backupFiles[i].BackupTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("[Size] %d bytes\n", backupFiles[i].Size)

		if backupFiles[i].IsBeforeRestore {
			fmt.Printf("[Type] Before restore\n")
		} else {
			fmt.Printf("[Type] Backup\n")
		}
	}

	return nil
}

func restoreDatabase(c *core.CliContext) error {
	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	backupDir, err := getDatabaseBackupDir(c, config.DatabaseBackupPath)

	if err != nil {
		log.CliErrorf(c, "[database.restoreDatabase] cannot resolve backup path, because %s", err.Error())
		return err
	}

	restoreFromFilePath, err := filepath.Abs(c.String("file"))

	if err != nil {
		log.CliErrorf(c, "[database.restoreDatabase] cannot resolve backup file path, because %s", err.Error())
		return err
	}

	if config.DatabaseConfig == nil || config.DatabaseConfig.DatabaseType != settings.Sqlite3DbType {
		log.CliErrorf(c, "[database.restoreDatabase] %s, please use the tools of your database server to restore", datastore.ErrDatabaseRestoreNotSupported.Error())
		return datastore.ErrDatabaseRestoreNotSupported
	}

	// check the file before asking, so an invalid file is refused without any confirmation
	if err = datastore.VerifySqliteDatabaseFile(restoreFromFilePath); err != nil {
		log.CliErrorf(c, "[database.restoreDatabase] the backup file is refused and the database is untouched, because %s", err.Error())
		return err
	}

	if !c.Bool("yes") {
		fmt.Printf("The database \"%s\" will be replaced with \"%s\".\n", config.DatabaseConfig.DatabasePath, restoreFromFilePath)
		fmt.Printf("The current database will be backed up into \"%s\" first.\n", backupDir)
		fmt.Printf("Make sure the ezBookkeeping server is stopped, otherwise it will keep using the old database.\n")
		fmt.Printf("Type \"yes\" to continue: ")

		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		if answer = strings.ToLower(strings.TrimSpace(answer)); answer != "yes" && answer != "y" {
			log.CliErrorf(c, "[database.restoreDatabase] %s", errDatabaseRestoreCancelled.Error())
			return errDatabaseRestoreCancelled
		}
	}

	result, err := datastore.RestoreDatabase(c, config, restoreFromFilePath, backupDir)

	if result != nil && result.BeforeRestoreBackupFilePath != "" {
		log.CliInfof(c, "[database.restoreDatabase] the database before restoring has been backed up to \"%s\"", result.BeforeRestoreBackupFilePath)
	}

	if err != nil {
		log.CliErrorf(c, "[database.restoreDatabase] restore database failed, because %s", err.Error())
		return err
	}

	for i := 0; i < len(result.RemovedSideFilePaths); i++ {
		log.CliInfof(c, "[database.restoreDatabase] sqlite side file \"%s\" of the old database has been removed", result.RemovedSideFilePaths[i])
	}

	log.CliInfof(c, "[database.restoreDatabase] database \"%s\" has been restored from \"%s\"", result.DatabaseFilePath, result.RestoredFromFilePath)
	log.CliInfof(c, "[database.restoreDatabase] to undo it, run \"database restore --file %s\"", result.BeforeRestoreBackupFilePath)
	return nil
}

func getDatabaseBackupDir(c *core.CliContext, defaultBackupDir string) (string, error) {
	if c.String("path") != "" {
		return filepath.Abs(c.String("path"))
	}

	return defaultBackupDir, nil
}

func updateAllDatabaseTablesStructure(c *core.CliContext) error {
	var err error

	err = datastore.Container.UserStore.SyncStructs(new(models.User))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] user table maintained successfully")

	err = datastore.Container.UserStore.SyncStructs(new(models.TwoFactor))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] two-factor table maintained successfully")

	err = datastore.Container.UserStore.SyncStructs(new(models.TwoFactorRecoveryCode))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] two-factor recovery code table maintained successfully")

	err = datastore.Container.TokenStore.SyncStructs(new(models.TokenRecord))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] token record table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.Account))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] account table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.Transaction))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionCategory))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction category table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTagGroup))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction tag group table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTag))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction tag table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTagIndex))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction tag index table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTemplate))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction template table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionPictureInfo))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction picture table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.UserCustomIcon))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] user custom icon table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.UserCustomExchangeRate))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] user custom exchange rate table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.UserApplicationCloudSetting))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] user application cloud settings table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.UserExternalAuth))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] user external auth table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.InsightsExplorer))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] insights explorer table maintained successfully")

	err = datastore.Container.UserStore.SyncStructs(new(models.BookMember))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] book member table maintained successfully")

	err = datastore.Container.UserStore.SyncStructs(new(models.SharedBook))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] shared book table maintained successfully")

	err = datastore.Container.UserStore.SyncStructs(new(models.SharedBookAccount))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] shared book account table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionCategoryBudget))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] transaction category budget table maintained successfully")

	err = datastore.Container.UserDataStore.SyncStructs(new(models.Project))

	if err != nil {
		return err
	}

	log.BootInfof(c, "[database.updateAllDatabaseTablesStructure] project table maintained successfully")

	return nil
}
