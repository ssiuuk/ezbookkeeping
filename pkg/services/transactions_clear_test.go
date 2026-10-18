package services

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

const (
	clearTestUid            = int64(1001)
	clearTestBookId         = int64(9001)
	clearTestAccountId      = int64(101)
	clearTestOtherAccountId = int64(102)
)

// initClearTestDataStore initializes the global data store with a temporary sqlite database for the duration of the test
func initClearTestDataStore(t *testing.T) {
	oldUserStore := datastore.Container.UserStore
	oldTokenStore := datastore.Container.TokenStore
	oldUserDataStore := datastore.Container.UserDataStore

	t.Cleanup(func() {
		datastore.Container.UserStore = oldUserStore
		datastore.Container.TokenStore = oldTokenStore
		datastore.Container.UserDataStore = oldUserDataStore
	})

	err := datastore.InitializeDataStore(&settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(t.TempDir(), "ezbookkeeping.db"),
			MaxIdleConnection: 2,
		},
	})
	assert.Nil(t, err)

	err = datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction), new(models.TransactionTagIndex), new(models.TransactionPictureInfo), new(models.SharedBookAccount))
	assert.Nil(t, err)
}

// insertClearTestData creates the scenario of the report: an account with a personal income of 100.00 which is shared
// to a family book and used by a family book expense of 30.00, so its balance is 70.00
func insertClearTestData(t *testing.T, shareAccount bool, addFamilyTransaction bool) {
	sess := datastore.Container.UserDataStore.Choose(clearTestUid).NewSession(core.NewNullContext())
	defer sess.Close()

	balance := int64(10000)

	if addFamilyTransaction {
		balance -= 3000
	}

	_, err := sess.Insert(&models.Account{
		AccountId: clearTestAccountId,
		Uid:       clearTestUid,
		Name:      "Shared Cash",
		Type:      models.ACCOUNT_TYPE_SINGLE_ACCOUNT,
		Currency:  "USD",
		Balance:   balance,
	}, &models.Account{
		AccountId: clearTestOtherAccountId,
		Uid:       clearTestUid,
		Name:      "Private Cash",
		Type:      models.ACCOUNT_TYPE_SINGLE_ACCOUNT,
		Currency:  "USD",
		Balance:   500,
	})
	assert.Nil(t, err)

	_, err = sess.Insert(&models.Transaction{
		TransactionId:   1,
		Uid:             clearTestUid,
		Type:            models.TRANSACTION_DB_TYPE_INCOME,
		AccountId:       clearTestAccountId,
		TransactionTime: 1000,
		Amount:          10000,
	}, &models.Transaction{
		TransactionId:   2,
		Uid:             clearTestUid,
		Type:            models.TRANSACTION_DB_TYPE_INCOME,
		AccountId:       clearTestOtherAccountId,
		TransactionTime: 2000,
		Amount:          500,
	})
	assert.Nil(t, err)

	if shareAccount {
		_, err = sess.Insert(&models.SharedBookAccount{
			BookId:    clearTestBookId,
			AccountId: clearTestAccountId,
			OwnerUid:  clearTestUid,
		})
		assert.Nil(t, err)
	}

	if addFamilyTransaction {
		_, err = sess.Insert(&models.Transaction{
			TransactionId:   3,
			Uid:             clearTestBookId,
			Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
			AccountId:       clearTestAccountId,
			TransactionTime: 3000,
			Amount:          3000,
			CreatedByUid:    clearTestUid,
		})
		assert.Nil(t, err)
	}
}

func getClearTestAccount(t *testing.T, accountId int64) *models.Account {
	account := &models.Account{}
	has, err := datastore.Container.UserDataStore.Choose(clearTestUid).NewSession(core.NewNullContext()).ID(accountId).Get(account)
	assert.Nil(t, err)
	assert.True(t, has)
	return account
}

func getClearTestTransaction(t *testing.T, transactionId int64) *models.Transaction {
	transaction := &models.Transaction{}
	has, err := datastore.Container.UserDataStore.Choose(clearTestUid).NewSession(core.NewNullContext()).ID(transactionId).Get(transaction)
	assert.Nil(t, err)
	assert.True(t, has)
	return transaction
}

func assertClearTestDataUntouched(t *testing.T, sharedAccountBalance int64, familyTransactionExists bool) {
	sharedAccount := getClearTestAccount(t, clearTestAccountId)
	assert.False(t, sharedAccount.Deleted)
	assert.Equal(t, sharedAccountBalance, sharedAccount.Balance)

	otherAccount := getClearTestAccount(t, clearTestOtherAccountId)
	assert.False(t, otherAccount.Deleted)
	assert.Equal(t, int64(500), otherAccount.Balance)

	assert.False(t, getClearTestTransaction(t, 1).Deleted)
	assert.False(t, getClearTestTransaction(t, 2).Deleted)

	if familyTransactionExists {
		assert.False(t, getClearTestTransaction(t, 3).Deleted)
	}
}

func TestDeleteAllTransactions_RejectedWhenAccountIsSharedAndUsedByFamilyBook(t *testing.T) {
	for _, deleteAccount := range []bool{false, true} {
		initClearTestDataStore(t)
		insertClearTestData(t, true, true)
		c := core.NewNullContext()

		assert.Equal(t, errs.ErrAccountsUsedBySharedBook, Transactions.CheckAccountsNotUsedByOtherBooks(c, clearTestUid))

		err := Transactions.DeleteAllTransactions(c, clearTestUid, deleteAccount)
		assert.Equal(t, errs.ErrAccountsUsedBySharedBook, err)
		assertClearTestDataUntouched(t, 7000, true)
	}
}

func TestDeleteAllTransactions_RejectedWhenAccountIsOnlyShared(t *testing.T) {
	for _, deleteAccount := range []bool{false, true} {
		initClearTestDataStore(t)
		insertClearTestData(t, true, false)

		err := Transactions.DeleteAllTransactions(core.NewNullContext(), clearTestUid, deleteAccount)
		assert.Equal(t, errs.ErrAccountsUsedBySharedBook, err)
		assertClearTestDataUntouched(t, 10000, false)
	}
}

func TestDeleteAllTransactions_RejectedWhenAccountIsUsedByFamilyBookTransactionWithoutSharing(t *testing.T) {
	for _, deleteAccount := range []bool{false, true} {
		initClearTestDataStore(t)
		insertClearTestData(t, false, true)

		err := Transactions.DeleteAllTransactions(core.NewNullContext(), clearTestUid, deleteAccount)
		assert.Equal(t, errs.ErrAccountsUsedBySharedBook, err)
		assertClearTestDataUntouched(t, 7000, true)
	}
}

func TestDeleteAllTransactions_RejectedWhenAccountIsDestinationOfFamilyBookTransfer(t *testing.T) {
	initClearTestDataStore(t)
	insertClearTestData(t, false, false)

	sess := datastore.Container.UserDataStore.Choose(clearTestUid).NewSession(core.NewNullContext())
	_, err := sess.Insert(&models.Transaction{
		TransactionId:    4,
		Uid:              clearTestBookId,
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        5001,
		RelatedAccountId: clearTestOtherAccountId,
		TransactionTime:  4000,
	})
	sess.Close()
	assert.Nil(t, err)

	err = Transactions.DeleteAllTransactions(core.NewNullContext(), clearTestUid, true)
	assert.Equal(t, errs.ErrAccountsUsedBySharedBook, err)
	assertClearTestDataUntouched(t, 10000, false)
}

func TestDeleteAllTransactions_ClearsTransactionsWhenNoAccountIsUsedByOtherBooks(t *testing.T) {
	initClearTestDataStore(t)
	insertClearTestData(t, false, false)
	c := core.NewNullContext()

	assert.Nil(t, Transactions.CheckAccountsNotUsedByOtherBooks(c, clearTestUid))

	err := Transactions.DeleteAllTransactions(c, clearTestUid, false)
	assert.Nil(t, err)

	for _, accountId := range []int64{clearTestAccountId, clearTestOtherAccountId} {
		account := getClearTestAccount(t, accountId)
		assert.False(t, account.Deleted)
		assert.Equal(t, int64(0), account.Balance)
	}

	assert.True(t, getClearTestTransaction(t, 1).Deleted)
	assert.True(t, getClearTestTransaction(t, 2).Deleted)
}

func TestDeleteAllTransactions_ClearsTransactionsAndAccountsWhenNoAccountIsUsedByOtherBooks(t *testing.T) {
	initClearTestDataStore(t)
	insertClearTestData(t, false, false)

	err := Transactions.DeleteAllTransactions(core.NewNullContext(), clearTestUid, true)
	assert.Nil(t, err)

	for _, accountId := range []int64{clearTestAccountId, clearTestOtherAccountId} {
		account := getClearTestAccount(t, accountId)
		assert.True(t, account.Deleted)
		assert.Equal(t, int64(0), account.Balance)
	}

	assert.True(t, getClearTestTransaction(t, 1).Deleted)
	assert.True(t, getClearTestTransaction(t, 2).Deleted)
}

func TestDeleteAllTransactions_IgnoresDeletedFamilyBookTransactions(t *testing.T) {
	initClearTestDataStore(t)
	insertClearTestData(t, false, true)

	sess := datastore.Container.UserDataStore.Choose(clearTestUid).NewSession(core.NewNullContext())
	_, err := sess.ID(int64(3)).Cols("deleted").Update(&models.Transaction{Deleted: true})
	sess.Close()
	assert.Nil(t, err)

	err = Transactions.DeleteAllTransactions(core.NewNullContext(), clearTestUid, true)
	assert.Nil(t, err)
	assert.True(t, getClearTestAccount(t, clearTestAccountId).Deleted)
}

func TestDeleteAllTransactionsOfAccount_KeepsFamilyBookTransactionsInBalance(t *testing.T) {
	initClearTestDataStore(t)
	insertClearTestData(t, true, true)

	err := Transactions.DeleteAllTransactionsOfAccount(core.NewNullContext(), clearTestUid, clearTestAccountId, 100)
	assert.Nil(t, err)

	// only the personal income is removed, the family book expense still counts in the balance
	sharedAccount := getClearTestAccount(t, clearTestAccountId)
	assert.False(t, sharedAccount.Deleted)
	assert.Equal(t, int64(-3000), sharedAccount.Balance)

	assert.True(t, getClearTestTransaction(t, 1).Deleted)
	assert.False(t, getClearTestTransaction(t, 2).Deleted)
	assert.False(t, getClearTestTransaction(t, 3).Deleted)
	assert.Equal(t, int64(500), getClearTestAccount(t, clearTestOtherAccountId).Balance)
}
