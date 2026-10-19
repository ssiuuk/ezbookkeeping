package models

// Budget alert thresholds, in percent of the budget
const (
	BUDGET_ALERT_THRESHOLD_WARNING  int32 = 80
	BUDGET_ALERT_THRESHOLD_EXCEEDED int32 = 100
)

// AllBudgetAlertThresholds represents all budget alert thresholds in ascending order
var AllBudgetAlertThresholds = []int32{BUDGET_ALERT_THRESHOLD_WARNING, BUDGET_ALERT_THRESHOLD_EXCEEDED}

// BudgetAlert represents a budget alert threshold which the spending of a category in a budget period has reached,
// each threshold of a category budget alerts at most once in a budget period
type BudgetAlert struct {
	AlertId           int64  `xorm:"PK"`
	Uid               int64  `xorm:"UNIQUE(UQE_budget_alert_uid_category_period_threshold) INDEX(IDX_budget_alert_uid_dismissed_time) NOT NULL"`
	CategoryId        int64  `xorm:"UNIQUE(UQE_budget_alert_uid_category_period_threshold) NOT NULL"`
	Period            string `xorm:"UNIQUE(UQE_budget_alert_uid_category_period_threshold) VARCHAR(7) NOT NULL"`
	Threshold         int32  `xorm:"UNIQUE(UQE_budget_alert_uid_category_period_threshold) NOT NULL"`
	SpentAmount       int64  `xorm:"NOT NULL"`
	BudgetAmount      int64  `xorm:"NOT NULL"`
	Currency          string `xorm:"VARCHAR(3) NOT NULL"`
	Dismissed         bool   `xorm:"INDEX(IDX_budget_alert_uid_dismissed_time) NOT NULL"`
	CreatedUnixTime   int64  `xorm:"INDEX(IDX_budget_alert_uid_dismissed_time)"`
	DismissedUnixTime int64
}

// BudgetAlertListRequest represents all parameters of budget alert listing request
type BudgetAlertListRequest struct {
	Count            int32 `form:"count" binding:"omitempty,min=1,max=50"`
	IncludeDismissed bool  `form:"include_dismissed"`
}

// BudgetAlertDismissRequest represents all parameters of budget alert dismissing request
type BudgetAlertDismissRequest struct {
	Ids []string `json:"ids" binding:"required,min=1,max=100"`
}

// BudgetAlertInfoResponse represents a view-object of budget alert
type BudgetAlertInfoResponse struct {
	Id           int64  `json:"id,string"`
	CategoryId   int64  `json:"categoryId,string"`
	CategoryName string `json:"categoryName"`
	Period       string `json:"period"`
	Threshold    int32  `json:"threshold"`
	Spent        int64  `json:"spent"`
	Budget       int64  `json:"budget"`
	Currency     string `json:"currency"`
	Dismissed    bool   `json:"dismissed"`
	Time         int64  `json:"time"`
}

// ToBudgetAlertInfoResponse returns a view-object according to database model
func (a *BudgetAlert) ToBudgetAlertInfoResponse(categoryName string) *BudgetAlertInfoResponse {
	return &BudgetAlertInfoResponse{
		Id:           a.AlertId,
		CategoryId:   a.CategoryId,
		CategoryName: categoryName,
		Period:       a.Period,
		Threshold:    a.Threshold,
		Spent:        a.SpentAmount,
		Budget:       a.BudgetAmount,
		Currency:     a.Currency,
		Dismissed:    a.Dismissed,
		Time:         a.CreatedUnixTime,
	}
}
