package services

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/locales"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/mail"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/templates"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

const budgetAlertBudgetUrlFormat = "%sdesktop#/budgets"

// LatestExchangeRatesGetter returns the latest exchange rates which the given book uses
type LatestExchangeRatesGetter func(c core.Context, uid int64) (*models.LatestExchangeRateResponse, error)

// BudgetAlertService represents budget alert service
type BudgetAlertService struct {
	ServiceUsingDB
	ServiceUsingConfig
	ServiceUsingUuid
	mailer                    mail.Mailer
	latestExchangeRatesGetter LatestExchangeRatesGetter
	runInBackground           func(task func())
}

// Initialize a budget alert service singleton instance
var (
	BudgetAlerts = &BudgetAlertService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingConfig: ServiceUsingConfig{
			container: settings.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
		mailer: mail.Container,
		runInBackground: func(task func()) {
			go task()
		},
	}
)

// budgetAlertSpendingItem represents the spending of one account in one primary expense category
type budgetAlertSpendingItem struct {
	primaryCategoryId int64
	currency          string
	amount            *big.Int
}

// SetLatestExchangeRatesGetter sets the function which returns the latest exchange rates, it is used to convert spending into the currency of budgets
// (the exchange rates data providers depend on services, so they cannot be used here directly)
func (s *BudgetAlertService) SetLatestExchangeRatesGetter(getter LatestExchangeRatesGetter) {
	s.latestExchangeRatesGetter = getter
}

// CheckBudgetAlertsInBackground checks the budget alerts of the given categories of the given book in background after transactions are saved,
// failures are only logged, so they never affect the saved transactions
func (s *BudgetAlertService) CheckBudgetAlertsInBackground(c core.Context, uid int64, categoryIds []int64, timezone *time.Location) {
	if !s.CurrentConfig().EnableBudgetAlert || uid <= 0 || len(categoryIds) < 1 {
		return
	}

	if timezone == nil {
		timezone = time.Local
	}

	backgroundContext := core.NewDetachedContext(c)

	s.runInBackground(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf(backgroundContext, "[budget_alerts.CheckBudgetAlertsInBackground] failed to check budget alerts for user \"uid:%d\", because %v", uid, r)
			}
		}()

		_, err := s.CheckBudgetAlerts(backgroundContext, uid, categoryIds, time.Now(), timezone)

		if err != nil {
			log.Warnf(backgroundContext, "[budget_alerts.CheckBudgetAlertsInBackground] failed to check budget alerts for user \"uid:%d\", because %s", uid, err.Error())
		}
	})
}

// CheckBudgetAlerts records the alert thresholds which the spending of the given categories (or their primary categories) of the given book has newly reached
// in the budget period (the calendar month in the given timezone) of the given time, sends alert emails of them, and returns the newly recorded alerts.
// The spending is calculated in the same way as the budget page, amounts in other currencies are converted into the currency of the budget by the latest exchange rates,
// and the budget of a category is not checked when any of its spending cannot be converted.
func (s *BudgetAlertService) CheckBudgetAlerts(c core.Context, uid int64, categoryIds []int64, now time.Time, timezone *time.Location) ([]*models.BudgetAlert, error) {
	if !s.CurrentConfig().EnableBudgetAlert {
		return nil, nil
	}

	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if len(categoryIds) < 1 {
		return nil, nil
	}

	budgets, err := TransactionCategoryBudgets.GetAllBudgetsByUid(c, uid)

	if err != nil {
		return nil, err
	}

	if TransactionCategoryBudgets.HasBudgetWithoutCurrency(budgets) {
		legacyCurrency, err := TransactionCategoryBudgets.GetBookDefaultCurrency(c, uid)

		if err != nil {
			return nil, err
		}

		TransactionCategoryBudgets.FillBudgetsCurrency(budgets, legacyCurrency)
	}

	budgetMap := make(map[int64]*models.TransactionCategoryBudget, len(budgets))

	for _, budget := range budgets {
		if budget.Amount > 0 {
			budgetMap[budget.CategoryId] = budget
		}
	}

	if len(budgetMap) < 1 {
		return nil, nil
	}

	changedCategories, err := TransactionCategories.GetCategoriesByCategoryIds(c, uid, utils.ToUniqueInt64Slice(categoryIds))

	if err != nil {
		return nil, err
	}

	checkingCategoryIds := make([]int64, 0, len(changedCategories))
	checkingCategoryIdMap := make(map[int64]bool, len(changedCategories))

	for _, category := range changedCategories {
		if category.Type != models.CATEGORY_TYPE_EXPENSE {
			continue
		}

		primaryCategoryId := category.CategoryId

		if category.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
			primaryCategoryId = category.ParentCategoryId
		}

		if _, exists := budgetMap[primaryCategoryId]; exists && !checkingCategoryIdMap[primaryCategoryId] {
			checkingCategoryIdMap[primaryCategoryId] = true
			checkingCategoryIds = append(checkingCategoryIds, primaryCategoryId)
		}
	}

	if len(checkingCategoryIds) < 1 {
		return nil, nil
	}

	period, startUnixTime, endUnixTime := getBudgetAlertPeriod(now, timezone)
	spendingItems, err := s.getPrimaryCategoriesSpendingItems(c, uid, checkingCategoryIdMap, startUnixTime, endUnixTime, timezone)

	if err != nil {
		return nil, err
	}

	spentAmounts := s.getSpentAmountsInBudgetCurrency(c, uid, spendingItems, budgetMap)

	var existedAlerts []*models.BudgetAlert
	err = s.UserDataDB(uid).NewSession(c).Where("uid=? AND period=?", uid, period).In("category_id", checkingCategoryIds).Find(&existedAlerts)

	if err != nil {
		return nil, err
	}

	existedThresholds := make(map[int64]map[int32]bool, len(existedAlerts))

	for _, alert := range existedAlerts {
		if _, exists := existedThresholds[alert.CategoryId]; !exists {
			existedThresholds[alert.CategoryId] = make(map[int32]bool)
		}

		existedThresholds[alert.CategoryId][alert.Threshold] = true
	}

	var newAlerts []*models.BudgetAlert
	nowUnixTime := time.Now().Unix()

	for _, categoryId := range checkingCategoryIds {
		budget := budgetMap[categoryId]
		spentAmount, exists := spentAmounts[categoryId]

		if !exists {
			continue
		}

		reachedThresholds := getNewlyReachedBudgetAlertThresholds(spentAmount, budget.Amount, existedThresholds[categoryId])

		for i, threshold := range reachedThresholds {
			alert := &models.BudgetAlert{
				AlertId:         s.GenerateUuid(uuid.UUID_TYPE_BUDGET_ALERT),
				Uid:             uid,
				CategoryId:      categoryId,
				Period:          period,
				Threshold:       threshold,
				SpentAmount:     spentAmount,
				BudgetAmount:    budget.Amount,
				Currency:        budget.Currency,
				CreatedUnixTime: nowUnixTime,
			}

			// when the spending reaches several thresholds at once, only the highest one needs to be shown
			if i < len(reachedThresholds)-1 {
				alert.Dismissed = true
				alert.DismissedUnixTime = nowUnixTime
			}

			if alert.AlertId < 1 {
				return newAlerts, errs.ErrSystemIsBusy
			}

			inserted, err := s.insertBudgetAlert(c, alert)

			if err != nil {
				return newAlerts, err
			} else if inserted {
				newAlerts = append(newAlerts, alert)
			}
		}
	}

	if len(newAlerts) > 0 {
		log.Infof(c, "[budget_alerts.CheckBudgetAlerts] %d budget alerts of period \"%s\" have been recorded for user \"uid:%d\"", len(newAlerts), period, uid)
		s.sendBudgetAlertEmails(c, uid, newAlerts)
	}

	return newAlerts, nil
}

// GetRecentBudgetAlerts returns the latest budget alerts of the given book
func (s *BudgetAlertService) GetRecentBudgetAlerts(c core.Context, uid int64, count int32, includeDismissed bool) ([]*models.BudgetAlert, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var alerts []*models.BudgetAlert
	sess := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid)

	if !includeDismissed {
		sess = sess.And("dismissed=?", false)
	}

	err := sess.OrderBy("created_unix_time desc, threshold desc").Limit(int(count), 0).Find(&alerts)

	return alerts, err
}

// DismissBudgetAlerts dismisses the given budget alerts of the given book, so that they are no longer shown to members of the book
func (s *BudgetAlertService) DismissBudgetAlerts(c core.Context, uid int64, alertIds []int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if len(alertIds) < 1 {
		return nil
	}

	updateModel := &models.BudgetAlert{
		Dismissed:         true,
		DismissedUnixTime: time.Now().Unix(),
	}

	_, err := s.UserDataDB(uid).NewSession(c).Cols("dismissed", "dismissed_unix_time").Where("uid=? AND dismissed=?", uid, false).In("alert_id", alertIds).Update(updateModel)

	return err
}

// DeleteAllBudgetAlerts deletes all budget alerts of the given book
func (s *BudgetAlertService) DeleteAllBudgetAlerts(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	_, err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).Delete(&models.BudgetAlert{})

	return err
}

// insertBudgetAlert saves the given budget alert, returns false when the same threshold has been recorded at the same time by another check
func (s *BudgetAlertService) insertBudgetAlert(c core.Context, alert *models.BudgetAlert) (bool, error) {
	_, err := s.UserDataDB(alert.Uid).NewSession(c).Insert(alert)

	if err == nil {
		return true, nil
	}

	exists, existsErr := s.UserDataDB(alert.Uid).NewSession(c).Where("uid=? AND category_id=? AND period=? AND threshold=?", alert.Uid, alert.CategoryId, alert.Period, alert.Threshold).Exist(&models.BudgetAlert{})

	if existsErr == nil && exists {
		return false, nil
	}

	return false, err
}

// getPrimaryCategoriesSpendingItems returns the expense of every account in the given primary categories (including their secondary categories) in the given time range
func (s *BudgetAlertService) getPrimaryCategoriesSpendingItems(c core.Context, uid int64, primaryCategoryIds map[int64]bool, startUnixTime int64, endUnixTime int64, timezone *time.Location) ([]*budgetAlertSpendingItem, error) {
	totalAmounts, err := Transactions.GetAccountsAndCategoriesTotalInflowAndOutflow(c, uid, startUnixTime, endUnixTime, nil, false, "", core.MATCH_MODE_DEFAULT, 0, timezone, false)

	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0, len(totalAmounts))
	accountIds := make([]int64, 0, len(totalAmounts))

	for _, totalAmount := range totalAmounts {
		if totalAmount.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			categoryIds = append(categoryIds, totalAmount.CategoryId)
			accountIds = append(accountIds, totalAmount.AccountId)
		}
	}

	if len(categoryIds) < 1 {
		return nil, nil
	}

	categoryMap, err := TransactionCategories.GetCategoriesByCategoryIds(c, uid, utils.ToUniqueInt64Slice(categoryIds))

	if err != nil {
		return nil, err
	}

	accountMap, err := Accounts.GetAccountsUsedByBookByAccountIds(c, uid, utils.ToUniqueInt64Slice(accountIds))

	if err != nil {
		return nil, err
	}

	spendingItems := make([]*budgetAlertSpendingItem, 0, len(totalAmounts))

	for _, totalAmount := range totalAmounts {
		if totalAmount.Type != models.TRANSACTION_DB_TYPE_EXPENSE {
			continue
		}

		category, exists := categoryMap[totalAmount.CategoryId]

		if !exists || category.Type != models.CATEGORY_TYPE_EXPENSE {
			continue
		}

		// the budget page only counts the accounts which the book can use, so accounts no longer shared to the book are not counted either
		account, exists := accountMap[totalAmount.AccountId]

		if !exists || account.SharingRevoked {
			continue
		}

		primaryCategoryId := category.CategoryId

		if category.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
			primaryCategoryId = category.ParentCategoryId
		}

		if !primaryCategoryIds[primaryCategoryId] {
			continue
		}

		spendingItems = append(spendingItems, &budgetAlertSpendingItem{
			primaryCategoryId: primaryCategoryId,
			currency:          account.Currency,
			amount:            totalAmount.Amount,
		})
	}

	return spendingItems, nil
}

// getSpentAmountsInBudgetCurrency returns the total spending of every primary category in the currency of its budget,
// the categories whose spending cannot be converted into the currency of their budgets are not returned
func (s *BudgetAlertService) getSpentAmountsInBudgetCurrency(c core.Context, uid int64, spendingItems []*budgetAlertSpendingItem, budgetMap map[int64]*models.TransactionCategoryBudget) map[int64]int64 {
	spentAmounts := make(map[int64]*big.Int)
	unconvertedCategoryIds := make(map[int64]bool)
	var exchangeRates map[string]*big.Rat
	exchangeRatesLoaded := false

	for _, item := range spendingItems {
		budget, exists := budgetMap[item.primaryCategoryId]

		if !exists || unconvertedCategoryIds[item.primaryCategoryId] {
			continue
		}

		amount := item.amount

		if item.currency != budget.Currency && amount.Sign() != 0 {
			if !exchangeRatesLoaded {
				exchangeRates = s.getLatestExchangeRates(c, uid)
				exchangeRatesLoaded = true
			}

			amount = getBudgetAlertExchangedAmount(amount, item.currency, budget.Currency, exchangeRates)

			if amount == nil {
				log.Warnf(c, "[budget_alerts.getSpentAmountsInBudgetCurrency] cannot convert spending from \"%s\" to \"%s\" for category \"id:%d\" of user \"uid:%d\"", item.currency, budget.Currency, item.primaryCategoryId, uid)
				unconvertedCategoryIds[item.primaryCategoryId] = true
				continue
			}
		}

		if _, exists := spentAmounts[item.primaryCategoryId]; !exists {
			spentAmounts[item.primaryCategoryId] = big.NewInt(0)
		}

		spentAmounts[item.primaryCategoryId].Add(spentAmounts[item.primaryCategoryId], amount)
	}

	result := make(map[int64]int64, len(budgetMap))

	for categoryId := range budgetMap {
		if unconvertedCategoryIds[categoryId] {
			continue
		}

		spentAmount, exists := spentAmounts[categoryId]

		if !exists {
			result[categoryId] = 0
		} else if spentAmount.IsInt64() {
			result[categoryId] = spentAmount.Int64()
		}
	}

	return result
}

// getLatestExchangeRates returns the rates of every currency against the base currency, or nil when there are no exchange rates
func (s *BudgetAlertService) getLatestExchangeRates(c core.Context, uid int64) map[string]*big.Rat {
	if s.latestExchangeRatesGetter == nil {
		return nil
	}

	exchangeRateResponse, err := s.latestExchangeRatesGetter(c, uid)

	if err != nil {
		log.Warnf(c, "[budget_alerts.getLatestExchangeRates] failed to get latest exchange rates for user \"uid:%d\", because %s", uid, err.Error())
		return nil
	}

	if exchangeRateResponse == nil {
		return nil
	}

	exchangeRates := make(map[string]*big.Rat, len(exchangeRateResponse.ExchangeRates))

	for _, exchangeRate := range exchangeRateResponse.ExchangeRates {
		rate, ok := new(big.Rat).SetString(exchangeRate.Rate)

		if ok && rate.Sign() > 0 {
			exchangeRates[exchangeRate.Currency] = rate
		}
	}

	return exchangeRates
}

// sendBudgetAlertEmails sends the alerts which are not dismissed to every member of the book who enables budget alert emails, failures are only logged
func (s *BudgetAlertService) sendBudgetAlertEmails(c core.Context, uid int64, alerts []*models.BudgetAlert) {
	if !s.CurrentConfig().EnableSMTP || s.mailer == nil {
		return
	}

	shownAlerts := make([]*models.BudgetAlert, 0, len(alerts))
	categoryIds := make([]int64, 0, len(alerts))

	for _, alert := range alerts {
		if !alert.Dismissed {
			shownAlerts = append(shownAlerts, alert)
			categoryIds = append(categoryIds, alert.CategoryId)
		}
	}

	if len(shownAlerts) < 1 {
		return
	}

	categoryMap, err := TransactionCategories.GetCategoriesByCategoryIds(c, uid, categoryIds)

	if err != nil {
		log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return
	}

	bookName := ""
	memberUids := []int64{uid}
	ownerUid, err := Books.GetBookOwnerUid(c, uid)

	if err != nil {
		log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] failed to get owner of book \"uid:%d\", because %s", uid, err.Error())
		return
	}

	if ownerUid != uid {
		book, err := Books.GetSharedBookById(c, uid)

		if err != nil {
			log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] failed to get shared book \"id:%d\", because %s", uid, err.Error())
			return
		}

		members, err := Books.GetBookMembers(c, uid)

		if err != nil {
			log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] failed to get members of shared book \"id:%d\", because %s", uid, err.Error())
			return
		}

		bookName = book.Name
		memberUids = []int64{ownerUid}

		for _, member := range members {
			memberUids = append(memberUids, member.MemberUid)
		}
	}

	for _, memberUid := range memberUids {
		user, err := Users.GetUserById(c, memberUid)

		if err != nil {
			log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] failed to get user \"uid:%d\", because %s", memberUid, err.Error())
			continue
		}

		if user.Disabled || user.NoBudgetAlertEmail || user.Email == "" || (s.CurrentConfig().EnableUserVerifyEmail && !user.EmailVerified) {
			continue
		}

		err = s.sendBudgetAlertEmail(c, user, bookName, shownAlerts, categoryMap)

		if err != nil {
			log.Warnf(c, "[budget_alerts.sendBudgetAlertEmails] cannot send budget alert email to user \"uid:%d\", because %s", user.Uid, err.Error())
		} else {
			log.Infof(c, "[budget_alerts.sendBudgetAlertEmails] budget alert email of book \"uid:%d\" has been sent to user \"uid:%d\"", uid, user.Uid)
		}
	}
}

// sendBudgetAlertEmail sends one email containing the given alerts to the given user
func (s *BudgetAlertService) sendBudgetAlertEmail(c core.Context, user *models.User, bookName string, alerts []*models.BudgetAlert, categoryMap map[int64]*models.TransactionCategory) error {
	locale := user.Language

	if locale == "" {
		locale = c.GetClientLocale()
	}

	localeTextItems := locales.GetLocaleTextItems(locale)
	budgetAlertTextItems := localeTextItems.BudgetAlertMailTextItems
	descriptions := make([]string, 0, len(alerts))

	for _, alert := range alerts {
		categoryName := ""

		if category, exists := categoryMap[alert.CategoryId]; exists {
			categoryName = category.Name
		}

		if bookName != "" {
			categoryName = bookName + " - " + categoryName
		}

		spentAmount := utils.FormatAmount(alert.SpentAmount) + " " + alert.Currency
		budgetAmount := utils.FormatAmount(alert.BudgetAmount) + " " + alert.Currency

		if alert.Threshold >= models.BUDGET_ALERT_THRESHOLD_EXCEEDED {
			descriptions = append(descriptions, fmt.Sprintf(budgetAlertTextItems.ExceededDescriptionFormat, categoryName, alert.Period, spentAmount, budgetAmount))
		} else {
			descriptions = append(descriptions, fmt.Sprintf(budgetAlertTextItems.WarningDescriptionFormat, categoryName, alert.Period, alert.Threshold, spentAmount, budgetAmount))
		}
	}

	tmpl, err := templates.GetTemplate(templates.TEMPLATE_BUDGET_ALERT)

	if err != nil {
		return err
	}

	templateParams := map[string]any{
		"AppName": localeTextItems.GlobalTextItems.AppName,
		"BudgetAlertMail": map[string]any{
			"Title":                   budgetAlertTextItems.Title,
			"Salutation":              fmt.Sprintf(budgetAlertTextItems.SalutationFormat, user.Nickname),
			"Descriptions":            descriptions,
			"BudgetUrl":               fmt.Sprintf(budgetAlertBudgetUrlFormat, s.CurrentConfig().RootUrl),
			"ViewBudgets":             budgetAlertTextItems.ViewBudgets,
			"DisableEmailDescription": budgetAlertTextItems.DisableEmailDescription,
		},
	}

	var bodyBuffer bytes.Buffer
	err = tmpl.Execute(&bodyBuffer, templateParams)

	if err != nil {
		return err
	}

	subject := budgetAlertTextItems.Title

	if len(descriptions) == 1 {
		subject = subject + ": " + descriptions[0]
	}

	message := &mail.MailMessage{
		To:      user.Email,
		Subject: subject,
		Body:    bodyBuffer.String(),
	}

	return s.mailer.SendMail(message)
}

// getBudgetAlertPeriod returns the key (yyyy-MM), the first and the last unix time of the budget period (calendar month) of the given time in the given timezone
func getBudgetAlertPeriod(now time.Time, timezone *time.Location) (string, int64, int64) {
	localNow := now.In(timezone)
	periodStartTime := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, timezone)
	periodEndTime := periodStartTime.AddDate(0, 1, 0)

	return fmt.Sprintf("%04d-%02d", localNow.Year(), int(localNow.Month())), periodStartTime.Unix(), periodEndTime.Unix() - 1
}

// getNewlyReachedBudgetAlertThresholds returns the thresholds (in ascending order) which the spending has reached but have not been recorded
func getNewlyReachedBudgetAlertThresholds(spentAmount int64, budgetAmount int64, recordedThresholds map[int32]bool) []int32 {
	if budgetAmount <= 0 || spentAmount <= 0 {
		return nil
	}

	var thresholds []int32
	spent := new(big.Int).Mul(big.NewInt(spentAmount), big.NewInt(100))

	for _, threshold := range models.AllBudgetAlertThresholds {
		thresholdAmount := new(big.Int).Mul(big.NewInt(budgetAmount), big.NewInt(int64(threshold)))

		if spent.Cmp(thresholdAmount) >= 0 && !recordedThresholds[threshold] {
			thresholds = append(thresholds, threshold)
		}
	}

	return thresholds
}

// getBudgetAlertExchangedAmount converts the amount from one currency into another by their rates against the same base currency,
// the result is truncated like the budget page does, returns nil when there is no rate of either currency
func getBudgetAlertExchangedAmount(amount *big.Int, fromCurrency string, toCurrency string, exchangeRates map[string]*big.Rat) *big.Int {
	if fromCurrency == toCurrency {
		return amount
	}

	fromRate, exists := exchangeRates[fromCurrency]

	if !exists {
		return nil
	}

	toRate, exists := exchangeRates[toCurrency]

	if !exists {
		return nil
	}

	exchangedAmount := new(big.Rat).SetInt(amount)
	exchangedAmount.Mul(exchangedAmount, toRate)
	exchangedAmount.Quo(exchangedAmount, fromRate)

	return new(big.Int).Quo(exchangedAmount.Num(), exchangedAmount.Denom())
}
