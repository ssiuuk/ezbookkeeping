package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func getBudgetAlertTestList(t *testing.T, uid int64, bookId int64) []*models.BudgetAlertInfoResponse {
	result, errResp := BudgetAlerts.BudgetAlertListHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/budgets/alerts/list.json", nil, uid, bookId))
	assert.Nil(t, errResp)

	return result.([]*models.BudgetAlertInfoResponse)
}

func TestTransactionCreate_RecordsBudgetAlertOfFamilyBook(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionCategoryBudget), new(models.BudgetAlert)))
	settings.Container.GetCurrentConfig().EnableBudgetAlert = true

	category, err := Transactions.transactionCategories.GetCategoryByCategoryId(core.NewNullContext(), env.bookId, env.categoryId)
	assert.Nil(t, err)

	// the family book has spent 12.00 of a food budget of 20.00 before the budget is saved
	_, err = datastore.Container.UserDataStore.Query(core.NewNullContext(), env.bookId).Insert(&models.TransactionCategoryBudget{
		Uid:        env.bookId,
		CategoryId: category.ParentCategoryId,
		Amount:     2000,
		Currency:   "USD",
	})
	assert.Nil(t, err)

	assert.Empty(t, getBudgetAlertTestList(t, env.ownerUid, env.bookId))

	_, errResp := addSharedBookTestExpense(env, env.memberUid, env.usedAccountId, 400, "more groceries")
	assert.Nil(t, errResp)

	// the alert is checked in background after the transaction is saved
	var alerts []*models.BudgetAlertInfoResponse
	assert.Eventually(t, func() bool {
		alerts = getBudgetAlertTestList(t, env.ownerUid, env.bookId)
		return len(alerts) > 0
	}, 5*time.Second, 20*time.Millisecond)

	assert.Len(t, alerts, 1)
	assert.Equal(t, category.ParentCategoryId, alerts[0].CategoryId)
	assert.Equal(t, "Food", alerts[0].CategoryName)
	assert.Equal(t, int32(80), alerts[0].Threshold)
	assert.Equal(t, int64(1600), alerts[0].Spent)
	assert.Equal(t, int64(2000), alerts[0].Budget)
	assert.Equal(t, "USD", alerts[0].Currency)

	// the member sees the same alert and can dismiss it for the book
	assert.Len(t, getBudgetAlertTestList(t, env.memberUid, env.bookId), 1)

	result, errResp := BudgetAlerts.BudgetAlertDismissHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/budgets/alerts/dismiss.json", &models.BudgetAlertDismissRequest{
		Ids: []string{utils.Int64ToString(alerts[0].Id)},
	}, env.memberUid, env.bookId))
	assert.Nil(t, errResp)
	assert.Equal(t, true, result)

	assert.Empty(t, getBudgetAlertTestList(t, env.ownerUid, env.bookId))

	// the personal book of the member has no alerts
	assert.Empty(t, getBudgetAlertTestList(t, env.memberUid, env.memberUid))
}
