package services

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// legacyTransactionCategoryBudget is the category budget table before budgets had a currency
type legacyTransactionCategoryBudget struct {
	Uid             int64 `xorm:"PK"`
	CategoryId      int64 `xorm:"PK"`
	Amount          int64 `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
}

func (b *legacyTransactionCategoryBudget) TableName() string {
	return "transaction_category_budget"
}

// initializes the global data store with a temporary sqlite database containing the tables used by category budgets,
// the budget table is created in its legacy structure with legacy budgets and then updated like "database update" does
func initializeCategoryBudgetTestDataStore(t *testing.T, legacyBudgets ...*legacyTransactionCategoryBudget) {
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
			DatabaseType:          settings.Sqlite3DbType,
			DatabasePath:          filepath.Join(t.TempDir(), "ezbookkeeping.db"),
			MaxIdleConnection:     1,
			MaxOpenConnection:     1,
			ConnectionMaxLifeTime: 60,
		},
	})
	assert.Nil(t, err)

	// the sqlite file must be released before the temporary directory is removed on Windows
	testDataStore := datastore.Container.UserDataStore
	t.Cleanup(func() {
		_ = testDataStore.Close()
	})

	err = datastore.Container.UserStore.SyncStructs(new(models.User), new(models.SharedBook))
	assert.Nil(t, err)

	err = datastore.Container.UserDataStore.SyncStructs(new(legacyTransactionCategoryBudget))
	assert.Nil(t, err)

	for _, budget := range legacyBudgets {
		_, err = datastore.Container.UserDataStore.Choose(budget.Uid).NewSession(core.NewNullContext()).Insert(budget)
		assert.Nil(t, err)
	}

	err = datastore.Container.UserDataStore.SyncStructs(new(models.TransactionCategoryBudget))
	assert.Nil(t, err)

	users := []*models.User{
		{Uid: 1, Username: "gbp", Email: "gbp@example.com", Nickname: "GBP", DefaultCurrency: "GBP"},
		{Uid: 2, Username: "usd", Email: "usd@example.com", Nickname: "USD", DefaultCurrency: "USD"},
	}

	for _, user := range users {
		_, err = datastore.Container.UserStore.Choose(user.Uid).NewSession(core.NewNullContext()).Insert(user)
		assert.Nil(t, err)
	}

	_, err = datastore.Container.UserStore.Choose(0).NewSession(core.NewNullContext()).Insert(&models.SharedBook{BookId: 100, OwnerUid: 1, Name: "Family"})
	assert.Nil(t, err)
}

func getCategoryBudgetMap(t *testing.T, uid int64) map[int64]*models.TransactionCategoryBudget {
	budgets, err := TransactionCategoryBudgets.GetAllBudgetsByUid(core.NewNullContext(), uid)
	assert.Nil(t, err)

	result := make(map[int64]*models.TransactionCategoryBudget, len(budgets))

	for _, budget := range budgets {
		result[budget.CategoryId] = budget
	}

	return result
}

func TestFillBudgetsCurrency(t *testing.T) {
	budgets := []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 100, Currency: "EUR"},
		{CategoryId: 2, Amount: 200},
	}

	assert.True(t, TransactionCategoryBudgets.HasBudgetWithoutCurrency(budgets))

	TransactionCategoryBudgets.FillBudgetsCurrency(budgets, "GBP")
	assert.Equal(t, "EUR", budgets[0].Currency)
	assert.Equal(t, "GBP", budgets[1].Currency)
	assert.False(t, TransactionCategoryBudgets.HasBudgetWithoutCurrency(budgets))
	assert.Equal(t, "GBP", budgets[1].ToTransactionCategoryBudgetInfoResponse().Currency)
}

func TestSaveBudgets_PersistsCurrency(t *testing.T) {
	initializeCategoryBudgetTestDataStore(t)
	c := core.NewNullContext()

	// a GBP member saves a 100.00 budget in the shared book, a USD member then saves another category with an explicit currency
	err := TransactionCategoryBudgets.SaveBudgets(c, 100, []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 10000},
	}, "GBP", "GBP")
	assert.Nil(t, err)

	err = TransactionCategoryBudgets.SaveBudgets(c, 100, []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 10000, Currency: "GBP"},
		{CategoryId: 2, Amount: 5000, Currency: "USD"},
	}, "USD", "GBP")
	assert.Nil(t, err)

	budgets := getCategoryBudgetMap(t, 100)
	assert.Equal(t, 2, len(budgets))
	assert.Equal(t, int64(10000), budgets[1].Amount)
	assert.Equal(t, "GBP", budgets[1].Currency)
	assert.Equal(t, int64(5000), budgets[2].Amount)
	assert.Equal(t, "USD", budgets[2].Currency)
}

func TestSaveBudgets_WithoutCurrency(t *testing.T) {
	initializeCategoryBudgetTestDataStore(t)
	c := core.NewNullContext()

	err := TransactionCategoryBudgets.SaveBudgets(c, 100, []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 10000, Currency: "GBP"},
		{CategoryId: 2, Amount: 5000, Currency: "GBP"},
	}, "GBP", "GBP")
	assert.Nil(t, err)

	// a client which does not send currency: an unchanged budget keeps its currency, a changed or new one is in the saver's currency
	err = TransactionCategoryBudgets.SaveBudgets(c, 100, []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 10000},
		{CategoryId: 2, Amount: 6000},
		{CategoryId: 3, Amount: 7000},
	}, "USD", "GBP")
	assert.Nil(t, err)

	budgets := getCategoryBudgetMap(t, 100)
	assert.Equal(t, "GBP", budgets[1].Currency)
	assert.Equal(t, "USD", budgets[2].Currency)
	assert.Equal(t, "USD", budgets[3].Currency)
}

func TestGetBookDefaultCurrency(t *testing.T) {
	initializeCategoryBudgetTestDataStore(t)
	c := core.NewNullContext()

	// personal books
	currency, err := TransactionCategoryBudgets.GetBookDefaultCurrency(c, 1)
	assert.Nil(t, err)
	assert.Equal(t, "GBP", currency)

	currency, err = TransactionCategoryBudgets.GetBookDefaultCurrency(c, 2)
	assert.Nil(t, err)
	assert.Equal(t, "USD", currency)

	// shared book owned by the GBP user
	currency, err = TransactionCategoryBudgets.GetBookDefaultCurrency(c, 100)
	assert.Nil(t, err)
	assert.Equal(t, "GBP", currency)
}

func TestSaveBudgets_LegacyBudgetWithoutCurrency(t *testing.T) {
	// budgets saved before budgets had a currency, then the table is migrated
	initializeCategoryBudgetTestDataStore(t,
		&legacyTransactionCategoryBudget{Uid: 100, CategoryId: 1, Amount: 10000, CreatedUnixTime: 1},
		&legacyTransactionCategoryBudget{Uid: 100, CategoryId: 2, Amount: 5000, CreatedUnixTime: 2},
	)
	c := core.NewNullContext()

	budgets := getCategoryBudgetMap(t, 100)
	assert.Equal(t, "", budgets[1].Currency)

	// on read, a legacy budget is in the default currency of the book owner, not of the reader
	legacyCurrency, err := TransactionCategoryBudgets.GetBookDefaultCurrency(c, 100)
	assert.Nil(t, err)

	budgetList := []*models.TransactionCategoryBudget{budgets[1], budgets[2]}
	TransactionCategoryBudgets.FillBudgetsCurrency(budgetList, legacyCurrency)
	assert.Equal(t, "GBP", budgetList[0].Currency)
	assert.Equal(t, "GBP", budgetList[1].Currency)

	// the USD member saves the unchanged budgets with an old client, the currency is back-filled with the book owner's currency
	err = TransactionCategoryBudgets.SaveBudgets(c, 100, []*models.TransactionCategoryBudget{
		{CategoryId: 1, Amount: 10000},
		{CategoryId: 2, Amount: 5000},
	}, "USD", legacyCurrency)
	assert.Nil(t, err)

	budgets = getCategoryBudgetMap(t, 100)
	assert.Equal(t, "GBP", budgets[1].Currency)
	assert.Equal(t, int64(1), budgets[1].CreatedUnixTime)
	assert.Equal(t, "GBP", budgets[2].Currency)
	assert.Equal(t, int64(2), budgets[2].CreatedUnixTime)
}
