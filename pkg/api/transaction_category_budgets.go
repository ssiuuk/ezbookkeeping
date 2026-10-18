package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// TransactionCategoryBudgetsApi represents transaction category budget api
type TransactionCategoryBudgetsApi struct {
	budgets    *services.TransactionCategoryBudgetService
	categories *services.TransactionCategoryService
	users      *services.UserService
}

// Initialize a transaction category budget api singleton instance
var (
	TransactionCategoryBudgets = &TransactionCategoryBudgetsApi{
		budgets:    services.TransactionCategoryBudgets,
		categories: services.TransactionCategories,
		users:      services.Users,
	}
)

// BudgetListHandler returns all category budgets of the current book
func (a *TransactionCategoryBudgetsApi) BudgetListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentBookUid()
	budgets, err := a.budgets.GetAllBudgetsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[transaction_category_budgets.BudgetListHandler] failed to get budgets for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if a.budgets.HasBudgetWithoutCurrency(budgets) {
		legacyCurrency, err := a.budgets.GetBookDefaultCurrency(c, uid)

		if err != nil {
			log.Errorf(c, "[transaction_category_budgets.BudgetListHandler] failed to get default currency of book \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		a.budgets.FillBudgetsCurrency(budgets, legacyCurrency)
	}

	budgetResps := make([]*models.TransactionCategoryBudgetInfoResponse, len(budgets))

	for i := 0; i < len(budgets); i++ {
		budgetResps[i] = budgets[i].ToTransactionCategoryBudgetInfoResponse()
	}

	return budgetResps, nil
}

// BudgetSaveHandler replaces all category budgets of the current book
func (a *TransactionCategoryBudgetsApi) BudgetSaveHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetSaveReq models.TransactionCategoryBudgetSaveRequest
	err := c.ShouldBindJSON(&budgetSaveReq)

	if err != nil {
		log.Warnf(c, "[transaction_category_budgets.BudgetSaveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentBookUid()
	categoryIds := make([]int64, 0, len(budgetSaveReq.Budgets))
	budgets := make([]*models.TransactionCategoryBudget, 0, len(budgetSaveReq.Budgets))
	addedCategoryIds := make(map[int64]bool, len(budgetSaveReq.Budgets))

	for _, budgetReq := range budgetSaveReq.Budgets {
		if addedCategoryIds[budgetReq.CategoryId] {
			continue
		}

		addedCategoryIds[budgetReq.CategoryId] = true
		categoryIds = append(categoryIds, budgetReq.CategoryId)
		budgets = append(budgets, &models.TransactionCategoryBudget{
			CategoryId: budgetReq.CategoryId,
			Amount:     budgetReq.Amount,
			Currency:   budgetReq.Currency,
		})
	}

	if len(categoryIds) > 0 {
		categories, err := a.categories.GetCategoriesByCategoryIds(c, uid, categoryIds)

		if err != nil {
			log.Errorf(c, "[transaction_category_budgets.BudgetSaveHandler] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		for _, categoryId := range categoryIds {
			category, exists := categories[categoryId]

			if !exists {
				return nil, errs.ErrTransactionCategoryNotFound
			}

			if category.Type != models.CATEGORY_TYPE_EXPENSE || category.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
				return nil, errs.ErrTransactionCategoryTypeInvalid
			}
		}
	}

	user, err := a.users.GetUserById(c, c.GetCurrentUid())

	if err != nil {
		log.Errorf(c, "[transaction_category_budgets.BudgetSaveHandler] failed to get user \"uid:%d\", because %s", c.GetCurrentUid(), err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	legacyCurrency, err := a.budgets.GetBookDefaultCurrency(c, uid)

	if err != nil {
		log.Errorf(c, "[transaction_category_budgets.BudgetSaveHandler] failed to get default currency of book \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	err = a.budgets.SaveBudgets(c, uid, budgets, user.DefaultCurrency, legacyCurrency)

	if err != nil {
		log.Errorf(c, "[transaction_category_budgets.BudgetSaveHandler] failed to save budgets for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transaction_category_budgets.BudgetSaveHandler] user \"uid:%d\" has saved %d budgets", uid, len(budgets))

	return a.BudgetListHandler(c)
}
