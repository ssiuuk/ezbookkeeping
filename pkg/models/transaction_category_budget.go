package models

// TransactionCategoryBudget represents the monthly budget of a primary expense category
type TransactionCategoryBudget struct {
	Uid             int64 `xorm:"PK"`
	CategoryId      int64 `xorm:"PK"`
	Amount          int64 `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
}

// TransactionCategoryBudgetItemRequest represents the budget of one category in a save request
type TransactionCategoryBudgetItemRequest struct {
	CategoryId int64 `json:"categoryId,string" binding:"required,min=1"`
	Amount     int64 `json:"amount" binding:"min=0,max=99999999999"`
}

// TransactionCategoryBudgetSaveRequest represents all parameters of category budgets save request
type TransactionCategoryBudgetSaveRequest struct {
	Budgets []*TransactionCategoryBudgetItemRequest `json:"budgets" binding:"omitempty,dive"`
}

// TransactionCategoryBudgetInfoResponse represents a view-object of category budget
type TransactionCategoryBudgetInfoResponse struct {
	CategoryId int64 `json:"categoryId,string"`
	Amount     int64 `json:"amount"`
}

// ToTransactionCategoryBudgetInfoResponse returns a view-object according to database model
func (b *TransactionCategoryBudget) ToTransactionCategoryBudgetInfoResponse() *TransactionCategoryBudgetInfoResponse {
	return &TransactionCategoryBudgetInfoResponse{
		CategoryId: b.CategoryId,
		Amount:     b.Amount,
	}
}
