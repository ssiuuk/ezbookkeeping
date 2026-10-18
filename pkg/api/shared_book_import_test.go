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

var sharedBookTestCategoryMap = map[int64]*models.TransactionCategory{
	1: {CategoryId: 1, Type: models.CATEGORY_TYPE_EXPENSE, Name: "Food"},
	2: {CategoryId: 2, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1, Name: "Groceries"},
}

// exports the transactions with the accounts named the way the export handler names them, or with their original names to stand for a bank or legacy file
func exportSharedBookTransactions(t *testing.T, accounts []*models.Account, naming *bookAccountNaming, transactions []*models.Transaction, useUniqueNames bool) []byte {
	if useUniqueNames {
		accounts = renameAccountsToUniqueNames(services.Accounts, accounts, naming)
	}

	exportAccountMap := make(map[int64]*models.Account, len(accounts))

	for i := 0; i < len(accounts); i++ {
		exportAccountMap[accounts[i].AccountId] = accounts[i]
	}

	exportedContent, err := _default.DefaultTransactionDataCSVFileConverter.ToExportedContent(core.NewNullContext(), naming.bookId, transactions, exportAccountMap, sharedBookTestCategoryMap, map[int64]*models.TransactionTag{}, map[int64][]int64{})
	assert.Nil(t, err)

	return exportedContent
}

// parses the file the same way the import parse handler does
func parseSharedBookImportFile(t *testing.T, accounts []*models.Account, naming *bookAccountNaming, content []byte) models.ImportedTransactionSlice {
	user := &models.User{Uid: naming.bookId, DefaultCurrency: "CNY"}
	importAccountMap := getUniqueAccountNameMap(services.Accounts, accounts, naming)
	expenseCategoryMap := map[string]map[string]*models.TransactionCategory{
		"Groceries": {"Food": sharedBookTestCategoryMap[2]},
	}

	importedTransactions, _, _, _, _, _, err := _default.DefaultTransactionDataCSVFileConverter.ParseImportedData(core.NewNullContext(), user, content, time.UTC, converter.DefaultImporterOptions, importAccountMap, expenseCategoryMap, nil, nil, nil)
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

func newSharedBookTestTransaction(id int64, accountId int64, comment string) *models.Transaction {
	return &models.Transaction{
		TransactionId:   id,
		TransactionTime: 1725165296000 + id*1000,
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:      2,
		AccountId:       accountId,
		Amount:          -1000 * id,
		Comment:         comment,
	}
}

func getBookAndSharedCashAccounts(bookId int64) ([]*models.Account, *bookAccountNaming) {
	accounts := []*models.Account{
		{Uid: bookId, AccountId: 1001, Name: "Cash", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2001, Name: "Cash", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2002, Name: "Bob Card", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	naming := &bookAccountNaming{bookId: bookId, bookName: "Family", ownerNicknames: map[int64]string{1: "Bob"}}

	return accounts, naming
}

func TestSharedBookExportImport_BookAndSharedAccountWithSameName(t *testing.T) {
	accounts, naming := getBookAndSharedCashAccounts(9001)
	transactions := []*models.Transaction{
		newSharedBookTestTransaction(1, 1001, "book cash"),
		newSharedBookTestTransaction(2, 2001, "bob cash"),
		newSharedBookTestTransaction(3, 2002, "bob card"),
	}

	content := exportSharedBookTransactions(t, accounts, naming, transactions, true)
	assert.Contains(t, string(content), ",Cash (Family),")
	assert.Contains(t, string(content), ",Cash (Bob),")

	importedTransactions := parseSharedBookImportFile(t, accounts, naming, content)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"book cash": 1001, "bob cash": 2001, "bob card": 2002})
}

func TestSharedBookImport_PlainNameOfBookAndSharedAccountIsNotMatched(t *testing.T) {
	accounts, naming := getBookAndSharedCashAccounts(9001)
	transactions := []*models.Transaction{
		newSharedBookTestTransaction(1, 1001, "plain cash 1"),
		newSharedBookTestTransaction(2, 2001, "plain cash 2"),
		newSharedBookTestTransaction(3, 2002, "bob card"),
	}

	// a bank or legacy file only has the plain name "Cash", which must not silently pick either account
	content := exportSharedBookTransactions(t, accounts, naming, transactions, false)
	importedTransactions := parseSharedBookImportFile(t, accounts, naming, content)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"plain cash 1": 0, "plain cash 2": 0, "bob card": 2002})
}

func TestSharedBookExportImport_TwoSharedAccountsWithSameName(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 2, AccountId: 3001, Name: "Visa", Currency: "USD", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: bookId, AccountId: 1001, Name: "Family Cash", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	naming := &bookAccountNaming{bookId: bookId, bookName: "Family", ownerNicknames: map[int64]string{1: "Bob", 2: "Alice"}}
	transactions := []*models.Transaction{
		newSharedBookTestTransaction(1, 2001, "bob visa"),
		newSharedBookTestTransaction(2, 3001, "alice visa"),
		newSharedBookTestTransaction(3, 1001, "family cash"),
	}

	// the two "Visa" accounts use different currencies, which used to make a valid exported file fail to parse
	content := exportSharedBookTransactions(t, accounts, naming, transactions, true)
	importedTransactions := parseSharedBookImportFile(t, accounts, naming, content)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"bob visa": 2001, "alice visa": 3001, "family cash": 1001})
}

func TestSharedBookImport_PlainNameOfTwoSharedAccountsIsNotMatched(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 2, AccountId: 3001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	naming := &bookAccountNaming{bookId: bookId, bookName: "Family", ownerNicknames: map[int64]string{1: "Bob", 2: "Alice"}}
	transactions := []*models.Transaction{
		newSharedBookTestTransaction(1, 2001, "plain visa 1"),
		newSharedBookTestTransaction(2, 3001, "plain visa 2"),
	}

	content := exportSharedBookTransactions(t, accounts, naming, transactions, false)
	importedTransactions := parseSharedBookImportFile(t, accounts, naming, content)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"plain visa 1": 0, "plain visa 2": 0})
}

func TestSharedBookExportImport_SameMemberSharesTwoAccountsWithSameName(t *testing.T) {
	bookId := int64(9001)
	accounts := []*models.Account{
		{Uid: 1, AccountId: 2001, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Uid: 1, AccountId: 2002, Name: "Visa", Currency: "CNY", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}
	naming := &bookAccountNaming{bookId: bookId, bookName: "Family", ownerNicknames: map[int64]string{1: "Bob"}}
	transactions := []*models.Transaction{
		newSharedBookTestTransaction(1, 2001, "first"),
		newSharedBookTestTransaction(2, 2002, "second"),
	}

	content := exportSharedBookTransactions(t, accounts, naming, transactions, true)
	importedTransactions := parseSharedBookImportFile(t, accounts, naming, content)
	assertImportedAccounts(t, importedTransactions, map[string]int64{"first": 2001, "second": 2002})
}
