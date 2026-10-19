package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const defaultBudgetAlertListCount = 10

// BudgetAlertsApi represents budget alert api
type BudgetAlertsApi struct {
	ApiUsingConfig
	budgetAlerts *services.BudgetAlertService
	categories   *services.TransactionCategoryService
}

// Initialize a budget alert api singleton instance
var (
	BudgetAlerts = &BudgetAlertsApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		budgetAlerts: services.BudgetAlerts,
		categories:   services.TransactionCategories,
	}
)

// BudgetAlertListHandler returns the latest budget alerts of the current book
func (a *BudgetAlertsApi) BudgetAlertListHandler(c *core.WebContext) (any, *errs.Error) {
	var alertListReq models.BudgetAlertListRequest
	err := c.ShouldBindQuery(&alertListReq)

	if err != nil {
		log.Warnf(c, "[budget_alerts.BudgetAlertListHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	alertResps := make([]*models.BudgetAlertInfoResponse, 0)

	if !a.CurrentConfig().EnableBudgetAlert {
		return alertResps, nil
	}

	if alertListReq.Count < 1 {
		alertListReq.Count = defaultBudgetAlertListCount
	}

	uid := c.GetCurrentBookUid()
	alerts, err := a.budgetAlerts.GetRecentBudgetAlerts(c, uid, alertListReq.Count, alertListReq.IncludeDismissed)

	if err != nil {
		log.Errorf(c, "[budget_alerts.BudgetAlertListHandler] failed to get budget alerts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if len(alerts) < 1 {
		return alertResps, nil
	}

	categoryIds := make([]int64, 0, len(alerts))

	for _, alert := range alerts {
		categoryIds = append(categoryIds, alert.CategoryId)
	}

	categories, err := a.categories.GetCategoriesByCategoryIds(c, uid, utils.ToUniqueInt64Slice(categoryIds))

	if err != nil {
		log.Errorf(c, "[budget_alerts.BudgetAlertListHandler] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for _, alert := range alerts {
		category, exists := categories[alert.CategoryId]

		// the alerts of deleted categories are not shown
		if !exists {
			continue
		}

		alertResps = append(alertResps, alert.ToBudgetAlertInfoResponse(category.Name))
	}

	return alertResps, nil
}

// BudgetAlertDismissHandler dismisses budget alerts of the current book
func (a *BudgetAlertsApi) BudgetAlertDismissHandler(c *core.WebContext) (any, *errs.Error) {
	var alertDismissReq models.BudgetAlertDismissRequest
	err := c.ShouldBindJSON(&alertDismissReq)

	if err != nil {
		log.Warnf(c, "[budget_alerts.BudgetAlertDismissHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	alertIds, err := utils.StringArrayToInt64Array(alertDismissReq.Ids)

	if err != nil {
		log.Warnf(c, "[budget_alerts.BudgetAlertDismissHandler] parse budget alert ids failed, because %s", err.Error())
		return nil, errs.ErrBudgetAlertIdInvalid
	}

	uid := c.GetCurrentBookUid()
	err = a.budgetAlerts.DismissBudgetAlerts(c, uid, alertIds)

	if err != nil {
		log.Errorf(c, "[budget_alerts.BudgetAlertDismissHandler] failed to dismiss budget alerts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[budget_alerts.BudgetAlertDismissHandler] user \"uid:%d\" has dismissed %d budget alerts of book \"uid:%d\"", c.GetCurrentUid(), len(alertIds), uid)

	return true, nil
}
