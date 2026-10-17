package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// TransactionCategoryBudgetService represents transaction category budget service
type TransactionCategoryBudgetService struct {
	ServiceUsingDB
}

// Initialize a transaction category budget service singleton instance
var (
	TransactionCategoryBudgets = &TransactionCategoryBudgetService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// GetAllBudgetsByUid returns all category budgets of user
func (s *TransactionCategoryBudgetService) GetAllBudgetsByUid(c core.Context, uid int64) ([]*models.TransactionCategoryBudget, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var budgets []*models.TransactionCategoryBudget
	err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).Find(&budgets)

	return budgets, err
}

// SaveBudgets replaces all category budgets of user, budgets with zero amount are removed
func (s *TransactionCategoryBudgetService) SaveBudgets(c core.Context, uid int64, budgets []*models.TransactionCategoryBudget) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		var existedBudgets []*models.TransactionCategoryBudget
		err := sess.Where("uid=?", uid).Find(&existedBudgets)

		if err != nil {
			return err
		}

		createdTimes := make(map[int64]int64, len(existedBudgets))

		for _, budget := range existedBudgets {
			createdTimes[budget.CategoryId] = budget.CreatedUnixTime
		}

		_, err = sess.Where("uid=?", uid).Delete(&models.TransactionCategoryBudget{})

		if err != nil {
			return err
		}

		for _, budget := range budgets {
			if budget.Amount <= 0 {
				continue
			}

			budget.Uid = uid
			budget.CreatedUnixTime = now
			budget.UpdatedUnixTime = now

			if createdTime, exists := createdTimes[budget.CategoryId]; exists {
				budget.CreatedUnixTime = createdTime
			}

			_, err = sess.Insert(budget)

			if err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteAllBudgets deletes all category budgets of user
func (s *TransactionCategoryBudgetService) DeleteAllBudgets(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Where("uid=?", uid).Delete(&models.TransactionCategoryBudget{})
		return err
	})
}
