package services

import (
	"errors"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/mail"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

const (
	budgetAlertTestUid               = int64(1)
	budgetAlertTestMemberUid         = int64(2)
	budgetAlertTestOptedOutMemberUid = int64(3)
	budgetAlertTestSharedBookId      = int64(100)
	budgetAlertTestFoodCategoryId    = int64(11)
	budgetAlertTestGroceryCategoryId = int64(12)
	budgetAlertTestOtherCategoryId   = int64(13)
	budgetAlertTestUsdAccountId      = int64(21)
	budgetAlertTestEurAccountId      = int64(22)
	budgetAlertTestJpyAccountId      = int64(23)
	budgetAlertTestBudgetAmount      = int64(10000)
)

// budgetAlertTestNow is in the middle of a month, so that transactions a few hours earlier are in the same budget period
var budgetAlertTestNow = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

// fakeBudgetAlertMailer records the sent emails instead of sending them
type fakeBudgetAlertMailer struct {
	mutex    sync.Mutex
	messages []*mail.MailMessage
}

func (m *fakeBudgetAlertMailer) SendMail(message *mail.MailMessage) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.messages = append(m.messages, message)
	return nil
}

func (m *fakeBudgetAlertMailer) getRecipients() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	recipients := make([]string, len(m.messages))

	for i, message := range m.messages {
		recipients[i] = message.To
	}

	return recipients
}

type budgetAlertTestEnv struct {
	t              *testing.T
	config         *settings.Config
	mailer         *fakeBudgetAlertMailer
	nextUnixTime   int64
	exchangeRates  []*models.LatestExchangeRate
	exchangeRateOk bool
}

// initializes the global data store with a temporary sqlite database, a personal book of user 1 with a food budget of 100.00 USD,
// and a shared book owned by user 1 with two members, then replaces the mailer and exchange rates of the budget alert service
func initializeBudgetAlertTestEnv(t *testing.T) *budgetAlertTestEnv {
	oldUserStore := datastore.Container.UserStore
	oldTokenStore := datastore.Container.TokenStore
	oldUserDataStore := datastore.Container.UserDataStore
	oldConfig := settings.Container.GetCurrentConfig()
	oldMailer := BudgetAlerts.mailer
	oldExchangeRatesGetter := BudgetAlerts.latestExchangeRatesGetter
	oldRunInBackground := BudgetAlerts.runInBackground

	t.Cleanup(func() {
		datastore.Container.UserStore = oldUserStore
		datastore.Container.TokenStore = oldTokenStore
		datastore.Container.UserDataStore = oldUserDataStore
		settings.SetCurrentConfig(oldConfig)
		BudgetAlerts.mailer = oldMailer
		BudgetAlerts.latestExchangeRatesGetter = oldExchangeRatesGetter
		BudgetAlerts.runInBackground = oldRunInBackground
	})

	// the email templates are read from the working directory, which is the repository root when the server runs
	t.Chdir(filepath.Join("..", ".."))

	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(t.TempDir(), "ezbookkeeping.db"),
			MaxIdleConnection: 2,
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
		EnableBudgetAlert: true,
		EnableSMTP:        true,
		RootUrl:           "https://ezbookkeeping.example.com/",
	}

	settings.SetCurrentConfig(config)
	assert.Nil(t, datastore.InitializeDataStore(config))

	// the sqlite file must be released before the temporary directory is removed on Windows
	testDataStore := datastore.Container.UserDataStore
	t.Cleanup(func() {
		_ = testDataStore.Close()
	})

	assert.Nil(t, uuid.InitializeUuidGenerator(config))
	assert.Nil(t, datastore.Container.UserStore.SyncStructs(new(models.User), new(models.SharedBook), new(models.BookMember), new(models.SharedBookAccount)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction), new(models.TransactionCategory), new(models.TransactionCategoryBudget), new(models.BudgetAlert)))

	env := &budgetAlertTestEnv{
		t:              t,
		config:         config,
		mailer:         &fakeBudgetAlertMailer{},
		nextUnixTime:   budgetAlertTestNow.Unix() - 3600,
		exchangeRateOk: true,
		exchangeRates: []*models.LatestExchangeRate{
			{Currency: "EUR", Rate: "1"},
			{Currency: "USD", Rate: "1.25"},
		},
	}

	BudgetAlerts.mailer = env.mailer
	BudgetAlerts.runInBackground = func(task func()) {
		task()
	}
	BudgetAlerts.SetLatestExchangeRatesGetter(func(c core.Context, uid int64) (*models.LatestExchangeRateResponse, error) {
		if !env.exchangeRateOk {
			return nil, errors.New("exchange rates are not available")
		}

		return &models.LatestExchangeRateResponse{BaseCurrency: "EUR", ExchangeRates: env.exchangeRates}, nil
	})

	c := core.NewNullContext()
	users := []*models.User{
		{Uid: budgetAlertTestUid, Username: "owner", Email: "owner@example.com", Nickname: "Owner", DefaultCurrency: "USD"},
		{Uid: budgetAlertTestMemberUid, Username: "member", Email: "member@example.com", Nickname: "Member", DefaultCurrency: "USD", Language: "zh-Hans"},
		{Uid: budgetAlertTestOptedOutMemberUid, Username: "optedout", Email: "optedout@example.com", Nickname: "Opted Out", DefaultCurrency: "USD", NoBudgetAlertEmail: true},
	}

	for _, user := range users {
		_, err := datastore.Container.UserStore.Choose(user.Uid).NewSession(c).Insert(user)
		assert.Nil(t, err)
	}

	_, err := datastore.Container.UserStore.Choose(0).NewSession(c).Insert(&models.SharedBook{BookId: budgetAlertTestSharedBookId, OwnerUid: budgetAlertTestUid, Name: "Family"})
	assert.Nil(t, err)
	_, err = datastore.Container.UserStore.Choose(0).NewSession(c).Insert(
		&models.BookMember{BookUid: budgetAlertTestSharedBookId, MemberUid: budgetAlertTestMemberUid},
		&models.BookMember{BookUid: budgetAlertTestSharedBookId, MemberUid: budgetAlertTestOptedOutMemberUid})
	assert.Nil(t, err)

	for i, bookUid := range []int64{budgetAlertTestUid, budgetAlertTestSharedBookId} {
		offset := int64(i * 1000)
		sess := datastore.Container.UserDataStore.Choose(bookUid).NewSession(c)

		_, err = sess.Insert(
			&models.TransactionCategory{CategoryId: budgetAlertTestFoodCategoryId + offset, Uid: bookUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: models.LevelOneTransactionCategoryParentId, Name: "Food"},
			&models.TransactionCategory{CategoryId: budgetAlertTestGroceryCategoryId + offset, Uid: bookUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: budgetAlertTestFoodCategoryId + offset, Name: "Groceries"},
			&models.TransactionCategory{CategoryId: budgetAlertTestOtherCategoryId + offset, Uid: bookUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: models.LevelOneTransactionCategoryParentId, Name: "Other"},
			&models.Account{AccountId: budgetAlertTestUsdAccountId + offset, Uid: bookUid, Name: "Cash", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "USD"},
			&models.Account{AccountId: budgetAlertTestEurAccountId + offset, Uid: bookUid, Name: "Euro Card", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "EUR"},
			&models.Account{AccountId: budgetAlertTestJpyAccountId + offset, Uid: bookUid, Name: "Yen Card", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "JPY"},
			&models.TransactionCategoryBudget{Uid: bookUid, CategoryId: budgetAlertTestFoodCategoryId + offset, Amount: budgetAlertTestBudgetAmount, Currency: "USD"},
		)
		sess.Close()
		assert.Nil(t, err)
	}

	return env
}

// addExpense adds an expense of the groceries category (a secondary category of food) of the personal book
func (e *budgetAlertTestEnv) addExpense(accountId int64, amount int64) {
	e.addBookExpenseAt(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, accountId, amount, e.nextUnixTime)
	e.nextUnixTime++
}

func (e *budgetAlertTestEnv) addBookExpenseAt(uid int64, categoryId int64, accountId int64, amount int64, unixTime int64) {
	sess := datastore.Container.UserDataStore.Choose(uid).NewSession(core.NewNullContext())
	defer sess.Close()

	_, err := sess.Insert(&models.Transaction{
		TransactionId:   uuid.Container.GenerateUuid(uuid.UUID_TYPE_TRANSACTION),
		Uid:             uid,
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:      categoryId,
		AccountId:       accountId,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(unixTime),
		Amount:          amount,
	})
	assert.Nil(e.t, err)
}

func (e *budgetAlertTestEnv) check(uid int64, categoryId int64, now time.Time) []*models.BudgetAlert {
	alerts, err := BudgetAlerts.CheckBudgetAlerts(core.NewNullContext(), uid, []int64{categoryId}, now, time.UTC)
	assert.Nil(e.t, err)
	return alerts
}

func getBudgetAlertThresholds(alerts []*models.BudgetAlert) []int32 {
	thresholds := make([]int32, len(alerts))

	for i, alert := range alerts {
		thresholds[i] = alert.Threshold
	}

	return thresholds
}

func TestCheckBudgetAlerts_AlertsEachThresholdOncePerPeriod(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	// no alert below 80%
	env.addExpense(budgetAlertTestUsdAccountId, 7999)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
	assert.Empty(t, env.mailer.getRecipients())

	// one alert when the spending reaches 80%
	env.addExpense(budgetAlertTestUsdAccountId, 1)
	alerts := env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(alerts))
	assert.Equal(t, budgetAlertTestFoodCategoryId, alerts[0].CategoryId)
	assert.Equal(t, "2026-10", alerts[0].Period)
	assert.Equal(t, int64(8000), alerts[0].SpentAmount)
	assert.Equal(t, budgetAlertTestBudgetAmount, alerts[0].BudgetAmount)
	assert.Equal(t, "USD", alerts[0].Currency)
	assert.False(t, alerts[0].Dismissed)
	assert.Equal(t, []string{"owner@example.com"}, env.mailer.getRecipients())
	assert.Contains(t, env.mailer.messages[0].Body, "80% of its budget")
	assert.Contains(t, env.mailer.messages[0].Body, "80.00 USD of 100.00 USD")

	// not repeated while the spending stays between 80% and 100%
	env.addExpense(budgetAlertTestUsdAccountId, 1000)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
	assert.Len(t, env.mailer.getRecipients(), 1)

	// one alert when the spending reaches 100%
	env.addExpense(budgetAlertTestUsdAccountId, 1000)
	alerts = env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)
	assert.Equal(t, []int32{100}, getBudgetAlertThresholds(alerts))
	assert.Equal(t, int64(10000), alerts[0].SpentAmount)
	assert.Len(t, env.mailer.getRecipients(), 2)
	assert.Contains(t, env.mailer.messages[1].Body, "has exceeded its budget")

	// not repeated after the budget is exceeded
	env.addExpense(budgetAlertTestUsdAccountId, 5000)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
	assert.Len(t, env.mailer.getRecipients(), 2)

	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, false)
	assert.Nil(t, err)
	assert.Equal(t, []int32{100, 80}, getBudgetAlertThresholds(recentAlerts))
}

func TestCheckBudgetAlerts_NewPeriodRearmsThresholds(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addExpense(budgetAlertTestUsdAccountId, 8500)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)))

	// the spending of last month is not counted in the next month
	nextMonth := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, nextMonth))

	env.addBookExpenseAt(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestUsdAccountId, 9000, nextMonth.Unix()-3600)
	alerts := env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, nextMonth)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(alerts))
	assert.Equal(t, "2026-11", alerts[0].Period)
	assert.Equal(t, int64(9000), alerts[0].SpentAmount)
	assert.Len(t, env.mailer.getRecipients(), 2)
}

func TestCheckBudgetAlerts_ReachingBothThresholdsAtOnceShowsOnlyTheHighest(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addExpense(budgetAlertTestUsdAccountId, 12000)
	alerts := env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)
	assert.Equal(t, []int32{80, 100}, getBudgetAlertThresholds(alerts))
	assert.True(t, alerts[0].Dismissed)
	assert.False(t, alerts[1].Dismissed)

	// one email for the highest threshold
	assert.Len(t, env.mailer.getRecipients(), 1)
	assert.Contains(t, env.mailer.messages[0].Body, "has exceeded its budget")
	assert.NotContains(t, env.mailer.messages[0].Body, "80% of its budget")

	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, false)
	assert.Nil(t, err)
	assert.Equal(t, []int32{100}, getBudgetAlertThresholds(recentAlerts))

	recentAlerts, err = BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, []int32{100, 80}, getBudgetAlertThresholds(recentAlerts))
}

func TestCheckBudgetAlerts_IgnoresCategoriesWithoutBudgetAndOtherTypes(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addBookExpenseAt(budgetAlertTestUid, budgetAlertTestOtherCategoryId, budgetAlertTestUsdAccountId, 50000, env.nextUnixTime)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestOtherCategoryId, budgetAlertTestNow))

	// the spending of other categories is not counted in the food budget
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
}

func TestCheckBudgetAlerts_ConvertsSpendingIntoBudgetCurrency(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	// 40.00 EUR is 50.00 USD, 30.00 USD more makes 80%
	env.addExpense(budgetAlertTestEurAccountId, 4000)
	env.addExpense(budgetAlertTestUsdAccountId, 2999)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))

	env.addExpense(budgetAlertTestUsdAccountId, 1)
	alerts := env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(alerts))
	assert.Equal(t, int64(8000), alerts[0].SpentAmount)
}

func TestCheckBudgetAlerts_SkipsBudgetWhenSpendingCannotBeConverted(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	// there is no exchange rate of JPY
	env.addExpense(budgetAlertTestUsdAccountId, 9000)
	env.addExpense(budgetAlertTestJpyAccountId, 100)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))

	// the exchange rates cannot be loaded
	env.exchangeRateOk = false
	env.addExpense(budgetAlertTestEurAccountId, 100)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
	assert.Empty(t, env.mailer.getRecipients())
}

func TestCheckBudgetAlerts_EmailsBookMembersWhoEnableBudgetAlertEmails(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addBookExpenseAt(budgetAlertTestSharedBookId, budgetAlertTestGroceryCategoryId+1000, budgetAlertTestUsdAccountId+1000, 8000, env.nextUnixTime)
	alerts := env.check(budgetAlertTestSharedBookId, budgetAlertTestGroceryCategoryId+1000, budgetAlertTestNow)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(alerts))

	assert.Equal(t, []string{"owner@example.com", "member@example.com"}, env.mailer.getRecipients())
	assert.Contains(t, env.mailer.messages[0].Body, "Family - Food")
	assert.True(t, strings.HasPrefix(env.mailer.messages[1].Subject, "预算提醒"))

	// the personal book of the owner is not affected
	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, true)
	assert.Nil(t, err)
	assert.Empty(t, recentAlerts)
}

func TestCheckBudgetAlerts_NoEmailWhenSmtpIsDisabled(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)
	env.config.EnableSMTP = false

	env.addExpense(budgetAlertTestUsdAccountId, 8000)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)))
	assert.Empty(t, env.mailer.getRecipients())
}

func TestCheckBudgetAlerts_DisabledByServerSetting(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)
	env.config.EnableBudgetAlert = false

	env.addExpense(budgetAlertTestUsdAccountId, 12000)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
	BudgetAlerts.CheckBudgetAlertsInBackground(core.NewNullContext(), budgetAlertTestUid, []int64{budgetAlertTestGroceryCategoryId}, time.UTC)

	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, true)
	assert.Nil(t, err)
	assert.Empty(t, recentAlerts)
	assert.Empty(t, env.mailer.getRecipients())
}

func TestCheckBudgetAlertsInBackground_RecordsAlertOfCurrentPeriod(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addBookExpenseAt(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestUsdAccountId, 8000, time.Now().Unix()-1)
	BudgetAlerts.CheckBudgetAlertsInBackground(core.NewNullContext(), budgetAlertTestUid, []int64{budgetAlertTestGroceryCategoryId}, time.UTC)

	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, false)
	assert.Nil(t, err)
	assert.Equal(t, []int32{80}, getBudgetAlertThresholds(recentAlerts))
}

func TestDismissBudgetAlerts(t *testing.T) {
	env := initializeBudgetAlertTestEnv(t)

	env.addExpense(budgetAlertTestUsdAccountId, 8000)
	alerts := env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow)
	assert.Len(t, alerts, 1)

	// alerts of another book cannot be dismissed
	assert.Nil(t, BudgetAlerts.DismissBudgetAlerts(core.NewNullContext(), budgetAlertTestSharedBookId, []int64{alerts[0].AlertId}))
	recentAlerts, err := BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, false)
	assert.Nil(t, err)
	assert.Len(t, recentAlerts, 1)

	assert.Nil(t, BudgetAlerts.DismissBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, []int64{alerts[0].AlertId}))
	recentAlerts, err = BudgetAlerts.GetRecentBudgetAlerts(core.NewNullContext(), budgetAlertTestUid, 10, false)
	assert.Nil(t, err)
	assert.Empty(t, recentAlerts)

	// a dismissed threshold still does not alert again in the same period
	env.addExpense(budgetAlertTestUsdAccountId, 100)
	assert.Empty(t, env.check(budgetAlertTestUid, budgetAlertTestGroceryCategoryId, budgetAlertTestNow))
}

func TestGetNewlyReachedBudgetAlertThresholds(t *testing.T) {
	assert.Empty(t, getNewlyReachedBudgetAlertThresholds(7999, 10000, nil))
	assert.Equal(t, []int32{80}, getNewlyReachedBudgetAlertThresholds(8000, 10000, nil))
	assert.Equal(t, []int32{80, 100}, getNewlyReachedBudgetAlertThresholds(10000, 10000, nil))
	assert.Equal(t, []int32{100}, getNewlyReachedBudgetAlertThresholds(10001, 10000, map[int32]bool{80: true}))
	assert.Empty(t, getNewlyReachedBudgetAlertThresholds(20000, 10000, map[int32]bool{80: true, 100: true}))
	assert.Empty(t, getNewlyReachedBudgetAlertThresholds(20000, 0, nil))
}

func TestGetBudgetAlertExchangedAmount(t *testing.T) {
	rates := map[string]*big.Rat{"EUR": big.NewRat(1, 1), "USD": big.NewRat(5, 4)}
	assert.Equal(t, int64(5000), getBudgetAlertExchangedAmount(big.NewInt(4000), "EUR", "USD", rates).Int64())
	assert.Equal(t, int64(3999), getBudgetAlertExchangedAmount(big.NewInt(4999), "USD", "EUR", rates).Int64())
	assert.Nil(t, getBudgetAlertExchangedAmount(big.NewInt(100), "JPY", "USD", rates))
}

func TestGetBudgetAlertPeriod(t *testing.T) {
	timezone := time.FixedZone("UTC+8", 8*3600)
	period, startUnixTime, endUnixTime := getBudgetAlertPeriod(time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC), timezone)
	assert.Equal(t, "2026-11", period)
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, timezone).Unix(), startUnixTime)
	assert.Equal(t, time.Date(2026, 12, 1, 0, 0, 0, 0, timezone).Unix()-1, endUnixTime)
}
