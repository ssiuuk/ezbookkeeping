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

// GetAllBudgetsByUid returns all category budgets of user (book), the currency of budgets saved before budgets had a currency is empty
func (s *TransactionCategoryBudgetService) GetAllBudgetsByUid(c core.Context, uid int64) ([]*models.TransactionCategoryBudget, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var budgets []*models.TransactionCategoryBudget
	err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).Find(&budgets)

	return budgets, err
}

// GetBookDefaultCurrency returns the default currency of the owner of the given book (the user itself for a personal book, the owner for a shared book),
// it is the currency of budgets which were saved before budgets had a currency
func (s *TransactionCategoryBudgetService) GetBookDefaultCurrency(c core.Context, bookUid int64) (string, error) {
	ownerUid, err := Books.GetBookOwnerUid(c, bookUid)

	if err != nil {
		return "", err
	}

	owner, err := Users.GetUserById(c, ownerUid)

	if err != nil {
		return "", err
	}

	return owner.DefaultCurrency, nil
}

// HasBudgetWithoutCurrency returns whether any of the given budgets has no currency
func (s *TransactionCategoryBudgetService) HasBudgetWithoutCurrency(budgets []*models.TransactionCategoryBudget) bool {
	for _, budget := range budgets {
		if budget.Currency == "" {
			return true
		}
	}

	return false
}

// FillBudgetsCurrency sets the given currency to the budgets which have no currency
func (s *TransactionCategoryBudgetService) FillBudgetsCurrency(budgets []*models.TransactionCategoryBudget, currency string) {
	for _, budget := range budgets {
		if budget.Currency == "" {
			budget.Currency = currency
		}
	}
}

// SaveBudgets replaces all category budgets of user (book), budgets with zero amount are removed.
// A budget without currency (saved by a client which does not know budget currency) keeps the currency of the existing budget
// of the same category when its amount is unchanged, otherwise it is in the currency of the saver (defaultCurrency).
// Existing budgets without currency are treated as in legacyCurrency (the default currency of the book owner).
func (s *TransactionCategoryBudgetService) SaveBudgets(c core.Context, uid int64, budgets []*models.TransactionCategoryBudget, defaultCurrency string, legacyCurrency string) error {
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

		existedBudgetMap := make(map[int64]*models.TransactionCategoryBudget, len(existedBudgets))

		for _, budget := range existedBudgets {
			existedBudgetMap[budget.CategoryId] = budget
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

			existedBudget, exists := existedBudgetMap[budget.CategoryId]

			if exists {
				budget.CreatedUnixTime = existedBudget.CreatedUnixTime
			}

			if budget.Currency == "" {
				budget.Currency = defaultCurrency

				if exists && existedBudget.Amount == budget.Amount {
					budget.Currency = existedBudget.Currency

					if budget.Currency == "" {
						budget.Currency = legacyCurrency
					}
				}
			}

			if budget.Currency == "" {
				budget.Currency = legacyCurrency
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
