package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	_default "github.com/mayswind/ezbookkeeping/pkg/converters/default"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// exports a family book whose accounts share names, then imports the file back into the same book
func exportAndImportSharedBookTransactions(t *testing.T, bookId int64, accounts []*models.Account, ownerNicknames map[int64]string, transactions []*models.Transaction) models.ImportedTransactionSlice {
	accounts = renameAccountsOfBookToUniqueNames(services.Accounts, accounts, bookId, ownerNicknames)
	csvConverter := _default.DefaultTransactionDataCSVFileConverter
	c := core.NewNullContext()

	exportAccountMap := make(map[int64]*models.Account, len(accounts))

	for i := 0; i < len(accounts); i++ {
		exportAccountMap[accounts[i].AccountId] = accounts[i]
	}

	categoryMap := map[int64]*models.TransactionCategory{
		1: {CategoryId: 1, Type: models.CATEGORY_TYPE_EXPENSE, Name: "Food"},
		2: {CategoryId: 2, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1, Name: "Groceries"},
	}

	exportedContent, err := csvConverter.ToExportedContent(c, bookId, transactions, exportAccountMap, categoryMap, map[int64]*models.TransactionTag{}, map[int64][]int64{})
	assert.Nil(t, err)

	user := &models.User{Uid: bookId, DefaultCurrency: "CNY"}
	importAccountMap := services.Accounts.GetVisibleAccountNameMapOfBookByList(accounts, bookId)
	expenseCategoryMap := map[string]map[string]*models.TransactionCategory{
		"Groceries": {"Food": categoryMap[2]},
	}

	importedTransactions, _, _, _, _, _, err := csvConverter.ParseImportedData(c, user, exportedContent, time.UTC, converter.DefaultImporterOptions, importAccountMap, expenseCategoryMap, nil, nil, nil)
	assert.Nil(t, err)

	return importedTransactions
}

func assertImportedAccounts(t *testing.T, importedTransactions models.ImportedTransactionSlice, expectedAccountIds map[string]int64) {
	assert.Equal(t, len(expectedAccountIds), len(importedTransactions))

	for i := 0; i < len(importedTransactions); i++ {
		expectedAccountId, exists := expectedAccountIds[importedTransactions[i].Comment]
		assert.True(t, exists)
		assert.Equal(t, expectedAccountId, importedTransactions[i].AccountId, importedTransactions[i].Comment)
	}
}

func TestSharedBookExportImport_BookAndSharedAccountWithSameName(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: bookId, AccountId: 1001, Name: "Cash", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2001, Name: "Cash", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2002, Name: "Bob Card", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	transactions := []*models.Transaction{
		{TransactionId: 1, TransactionTime: 1725165296000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 1001, Amount: -1000, Comment: "book cash"},
		{TransactionId: 2, TransactionTime: 1725165297000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 2001, Amount: -2000, Comment: "bob cash"},
		{TransactionId: 3, TransactionTime: 1725165298000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 2002, Amount: -3000, Comment: "bob card"},
	}

	importedTransactions := exportAndImportSharedBookTransactions(t, bookId, accounts, map[int64]string{1: "Bob"}, transactions)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"book cash": 1001, "bob cash": 2001, "bob card": 2002})
}

func TestSharedBookExportImport_TwoSharedAccountsWithSameName(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 2, AccountId: 3001, Name: "Visa", Currency: "USD", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: bookId, AccountId: 1001, Name: "Family Cash", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	transactions := []*models.Transaction{
		{TransactionId: 1, TransactionTime: 1725165296000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 2001, Amount: -1000, Comment: "bob visa"},
		{TransactionId: 2, TransactionTime: 1725165297000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 3001, Amount: -2000, Comment: "alice visa"},
		{TransactionId: 3, TransactionTime: 1725165298000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 1001, Amount: -3000, Comment: "family cash"},
	}

	// the two "Visa" accounts use different currencies, which used to make a valid exported file fail to parse
	importedTransactions := exportAndImportSharedBookTransactions(t, bookId, accounts, map[int64]string{1: "Bob", 2: "Alice"}, transactions)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"bob visa": 2001, "alice visa": 3001, "family cash": 1001})
}

func TestSharedBookExportImport_SameMemberSharesTwoAccountsWithSameName(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2002, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	transactions := []*models.Transaction{
		{TransactionId: 1, TransactionTime: 1725165296000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 2001, Amount: -1000, Comment: "first"},
		{TransactionId: 2, TransactionTime: 1725165297000, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2, AccountId: 2002, Amount: -2000, Comment: "second"},
	}

	importedTransactions := exportAndImportSharedBookTransactions(t, bookId, accounts, map[int64]string{1: "Bob"}, transactions)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"first": 2001, "second": 2002})
}

func TestSharedBookImport_AmbiguousPlainNameIsNotMatched(t *testing.T) {
	bookId := int64(9001)
	// without an owner nickname the shared accounts keep their names, so a file naming "Visa" must not pick either of them
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 2, AccountId: 3001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}

	accountMap := services.Accounts.GetVisibleAccountNameMapOfBookByList(accounts, bookId)
	assert.NotContains(t, accountMap, "Visa")
}
