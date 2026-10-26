package andromoney

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/traditionalchinese"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const andromoneyTestData = "\"Windows Excel\",\"理財幫手AndroMoney\",\"20261006\"\n" +
	"\"Id\",\"幣別\",\"金額\",\"分類\",\"子分類\",\"日期\",\"付款(轉出)\",\"收款(轉入)\",\"備註\",\"Periodic\",\"專案\",\"商家(公司)\",\"uid\",\"時間\"\n" +
	"\"10959\",\"GBP\",\"30\",\"BB用品\",\"Holiday Camp\",\"20260901\",\"AE Cashback\",\"\",\"\",\"\",\"BabyExpenses\",\"Bright Horizons \",\"7aa0bf5b5ae148998746647db2eab8d9\",\"1622\"\n" +
	"\"10951\",\"GBP\",\"57.5\",\"居家生活\",\"家電用品\",\"20260901\",\"AE Cashback\",\"\",\"Yale Motion Sensor\",\"\",\"家庭開支\",\"Amazon\",\"983ccf98dd05413f8924d6fe53077c1e\",\"116\"\n" +
	"\"11037\",\"GBP\",\"3570.17\",\"一般收入\",\"家庭收入\",\"20260924\",\"\",\"HSBCUK - Common ACC\",\"\",\"6|20250724|20260924|null|2|1\",\"\",\"\",\"04B14F3D8C5245FDA19DE85F3799801B_20260924\",\"\"\n" +
	"\"11034\",\"GBP\",\"150.0\",\"轉帳\",\"儲蓄計畫\",\"20260926\",\"Barclays UK\",\"MS Stock\",\"\",\"\",\"\",\"\",\"05d7d79b6ee7425aafc1ad1dbda9ac0f\",\"52\"\n"

func TestAndroMoneyCsvFileImporterParseImportedData(t *testing.T) {
	importer := AndroMoneyTransactionDataCsvFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "GBP",
	}

	big5Data, err := traditionalchinese.Big5.NewEncoder().Bytes([]byte(andromoneyTestData))
	assert.Nil(t, err)

	for _, data := range [][]byte{[]byte(andromoneyTestData), big5Data} {
		allNewTransactions, allNewAccounts, allNewSubExpenseCategories, allNewSubIncomeCategories, allNewSubTransferCategories, allNewTags, err := importer.ParseImportedData(context, user, data, time.UTC, converter.DefaultImporterOptions.WithProjectAsTag(), nil, nil, nil, nil, nil)

		assert.Nil(t, err)

		assert.Equal(t, 4, len(allNewTransactions))
		assert.Equal(t, 4, len(allNewAccounts))
		assert.Equal(t, 2, len(allNewSubExpenseCategories))
		assert.Equal(t, 1, len(allNewSubIncomeCategories))
		assert.Equal(t, 1, len(allNewSubTransferCategories))
		assert.Equal(t, 2, len(allNewTags))

		assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, allNewTransactions[0].Type)
		assert.Equal(t, int64(1788225360), utils.GetUnixTimeFromTransactionTime(allNewTransactions[0].TransactionTime))
		assert.Equal(t, int64(5750), allNewTransactions[0].Amount)
		assert.Equal(t, "家電用品", allNewTransactions[0].OriginalCategoryName)
		assert.Equal(t, "Yale Motion Sensor", allNewTransactions[0].Comment)
		assert.Equal(t, []string{"家庭開支"}, allNewTransactions[0].OriginalTagNames)

		assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, allNewTransactions[1].Type)
		assert.Equal(t, int64(1788279720), utils.GetUnixTimeFromTransactionTime(allNewTransactions[1].TransactionTime))
		assert.Equal(t, int64(3000), allNewTransactions[1].Amount)
		assert.Equal(t, "AE Cashback", allNewTransactions[1].OriginalSourceAccountName)
		assert.Equal(t, "GBP", allNewTransactions[1].OriginalSourceAccountCurrency)
		assert.Equal(t, "Holiday Camp", allNewTransactions[1].OriginalCategoryName)
		assert.Equal(t, "Bright Horizons", allNewTransactions[1].Merchant)
		assert.Equal(t, []string{"BabyExpenses"}, allNewTransactions[1].OriginalTagNames)

		assert.Equal(t, models.TRANSACTION_DB_TYPE_INCOME, allNewTransactions[2].Type)
		assert.Equal(t, int64(1790208000), utils.GetUnixTimeFromTransactionTime(allNewTransactions[2].TransactionTime))
		assert.Equal(t, int64(357017), allNewTransactions[2].Amount)
		assert.Equal(t, "HSBCUK - Common ACC", allNewTransactions[2].OriginalSourceAccountName)
		assert.Equal(t, "家庭收入", allNewTransactions[2].OriginalCategoryName)
		assert.Equal(t, 0, len(allNewTransactions[2].OriginalTagNames))

		assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, allNewTransactions[3].Type)
		assert.Equal(t, int64(1790383920), utils.GetUnixTimeFromTransactionTime(allNewTransactions[3].TransactionTime))
		assert.Equal(t, int64(15000), allNewTransactions[3].Amount)
		assert.Equal(t, "Barclays UK", allNewTransactions[3].OriginalSourceAccountName)
		assert.Equal(t, "MS Stock", allNewTransactions[3].OriginalDestinationAccountName)
		assert.Equal(t, int64(15000), allNewTransactions[3].RelatedAccountAmount)
		assert.Equal(t, "儲蓄計畫", allNewTransactions[3].OriginalCategoryName)
	}
}

func TestAndroMoneyCsvFileImporterParseImportedData_WithoutProjectAsTag(t *testing.T) {
	importer := AndroMoneyTransactionDataCsvFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "GBP",
	}

	allNewTransactions, _, _, _, _, allNewTags, err := importer.ParseImportedData(context, user, []byte(andromoneyTestData), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)

	assert.Nil(t, err)
	assert.Equal(t, 4, len(allNewTransactions))
	assert.Equal(t, 0, len(allNewTags))
}

func TestAndroMoneyCsvFileImporterParseImportedData_InvalidFile(t *testing.T) {
	importer := AndroMoneyTransactionDataCsvFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "GBP",
	}

	_, _, _, _, _, _, err := importer.ParseImportedData(context, user, []byte("Time,Type,Amount\n2026-09-01 00:00:00,Expense,1\n"), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.EqualError(t, err, errs.ErrInvalidCSVFile.Message)

	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, []byte("\"Windows Excel\",\"AndroMoney\",\"20261006\"\n\"Id\",\"Currency\",\"Amount\",\"Category\",\"Sub Category\",\"Date\",\"From\",\"To\",\"Note\",\"Periodic\",\"Project\",\"Payee\",\"uid\",\"Time\"\n"), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.EqualError(t, err, errs.ErrNotFoundTransactionDataInFile.Message)

	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, []byte("\"Id\",\"Currency\",\"Amount\",\"Category\",\"Sub Category\",\"Date\",\"From\",\"To\",\"Note\",\"Periodic\",\"Project\",\"Payee\",\"uid\",\"Time\"\n\"1\",\"GBP\",\"1\",\"A\",\"B\",\"2026-9-1\",\"Cash\",\"\",\"\",\"\",\"\",\"\",\"x\",\"\"\n"), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.EqualError(t, err, errs.ErrTransactionTimeInvalid.Message)
}

func TestAndroMoneyCsvFileImporterParseImportedData_TransferToNewAccountKeepsCurrency(t *testing.T) {
	importer := AndroMoneyTransactionDataCsvFileImporter
	context := core.NewNullContext()

	// the user's default currency differs from the currency of the file
	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "USD",
	}

	data := "\"Id\",\"Currency\",\"Amount\",\"Category\",\"Sub Category\",\"Date\",\"From\",\"To\",\"Note\",\"Periodic\",\"Project\",\"Payee\",\"uid\",\"Time\"\n" +
		"\"1\",\"GBP\",\"100\",\"Transfer\",\"Savings\",\"20260901\",\"Account A\",\"Account B\",\"\",\"\",\"\",\"\",\"a\",\"900\"\n" +
		"\"2\",\"GBP\",\"5\",\"Food\",\"Lunch\",\"20260902\",\"Account B\",\"\",\"\",\"\",\"\",\"\",\"b\",\"1200\"\n" +
		"\"3\",\"HKD\",\"80\",\"Food\",\"Lunch\",\"20260903\",\"Account C\",\"\",\"\",\"\",\"\",\"\",\"c\",\"1200\"\n"

	allNewTransactions, allNewAccounts, _, _, _, _, err := importer.ParseImportedData(context, user, []byte(data), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(allNewTransactions))
	assert.Equal(t, 3, len(allNewAccounts))

	for _, account := range allNewAccounts {
		if account.Name == "Account C" {
			assert.Equal(t, "HKD", account.Currency)
		} else {
			assert.Equal(t, "GBP", account.Currency, account.Name)
		}
	}

	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, allNewTransactions[0].Type)
	assert.Equal(t, "GBP", allNewTransactions[0].OriginalDestinationAccountCurrency)
	assert.Equal(t, int64(10000), allNewTransactions[0].RelatedAccountAmount)

	// a transfer into an account used with another currency has an unknown received amount, so it is refused
	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, []byte(data+
		"\"4\",\"GBP\",\"10\",\"Transfer\",\"Savings\",\"20260904\",\"Account A\",\"Account C\",\"\",\"\",\"\",\"\",\"d\",\"1200\"\n"), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.EqualError(t, err, errs.ErrAccountCurrencyInvalid.Message)
}
