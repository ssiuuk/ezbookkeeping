package cmd

import (
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
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
		},
	},
}

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

	backupDir := config.DatabaseBackupPath

	if c.String("path") != "" {
		backupDir, err = filepath.Abs(c.String("path"))

		if err != nil {
			log.CliErrorf(c, "[database.backupDatabase] cannot resolve backup path, because %s", err.Error())
			return err
		}
	}

	backupFilePath, err := datastore.BackupDatabase(c, config, backupDir)

	if err != nil {
		log.CliErrorf(c, "[database.backupDatabase] back up database failed, because %s", err.Error())
		return err
	}

	log.CliInfof(c, "[database.backupDatabase] database has been backed up to \"%s\"", backupFilePath)
	return nil
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
