package models

import "github.com/mayswind/ezbookkeeping/pkg/utils"

// Project represents a project which groups transactions from different categories, stored in database
type Project struct {
	ProjectId       int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_project_uid_deleted_order) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_project_uid_deleted_order) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	Comment         string `xorm:"VARCHAR(255) NOT NULL"`
	DisplayOrder    int32  `xorm:"INDEX(IDX_project_uid_deleted_order) NOT NULL"`
	Hidden          bool   `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// ProjectTransactionStatistic represents the transaction count and time range of a project
type ProjectTransactionStatistic struct {
	ProjectId          int64 `xorm:"project_id"`
	TransactionCount   int64 `xorm:"transaction_count"`
	MinTransactionTime int64 `xorm:"min_transaction_time"`
	MaxTransactionTime int64 `xorm:"max_transaction_time"`
}

// ProjectGetRequest represents all parameters of project getting request
type ProjectGetRequest struct {
	Id int64 `form:"id,string" binding:"required,min=1"`
}

// ProjectCreateRequest represents all parameters of project creation request
type ProjectCreateRequest struct {
	Name    string `json:"name" binding:"required,notBlank,max=64"`
	Comment string `json:"comment" binding:"max=255"`
}

// ProjectModifyRequest represents all parameters of project modification request
type ProjectModifyRequest struct {
	Id      int64  `json:"id,string" binding:"required,min=1"`
	Name    string `json:"name" binding:"required,notBlank,max=64"`
	Comment string `json:"comment" binding:"max=255"`
}

// ProjectHideRequest represents all parameters of project hiding request
type ProjectHideRequest struct {
	Id     int64 `json:"id,string" binding:"required,min=1"`
	Hidden bool  `json:"hidden"`
}

// ProjectMoveRequest represents all parameters of project moving request
type ProjectMoveRequest struct {
	NewDisplayOrders []*ProjectNewDisplayOrderRequest `json:"newDisplayOrders" binding:"required,min=1"`
}

// ProjectNewDisplayOrderRequest represents a data pair of id and display order
type ProjectNewDisplayOrderRequest struct {
	Id           int64 `json:"id,string" binding:"required,min=1"`
	DisplayOrder int32 `json:"displayOrder"`
}

// ProjectDeleteRequest represents all parameters of project deleting request
type ProjectDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// ProjectInfoResponse represents a view-object of project
type ProjectInfoResponse struct {
	Id                   int64  `json:"id,string"`
	Name                 string `json:"name"`
	Comment              string `json:"comment"`
	DisplayOrder         int32  `json:"displayOrder"`
	Hidden               bool   `json:"hidden"`
	TransactionCount     int64  `json:"transactionCount"`
	FirstTransactionTime int64  `json:"firstTransactionTime,omitempty"`
	LastTransactionTime  int64  `json:"lastTransactionTime,omitempty"`
}

// ToProjectInfoResponse returns a view-object according to database model and its transaction statistic
func (p *Project) ToProjectInfoResponse(statistic *ProjectTransactionStatistic) *ProjectInfoResponse {
	resp := &ProjectInfoResponse{
		Id:           p.ProjectId,
		Name:         p.Name,
		Comment:      p.Comment,
		DisplayOrder: p.DisplayOrder,
		Hidden:       p.Hidden,
	}

	if statistic != nil && statistic.TransactionCount > 0 {
		resp.TransactionCount = statistic.TransactionCount
		resp.FirstTransactionTime = utils.GetUnixTimeFromTransactionTime(statistic.MinTransactionTime)
		resp.LastTransactionTime = utils.GetUnixTimeFromTransactionTime(statistic.MaxTransactionTime)
	}

	return resp
}

// ProjectInfoResponseSlice represents the slice data structure of ProjectInfoResponse
type ProjectInfoResponseSlice []*ProjectInfoResponse

// Len returns the count of items
func (s ProjectInfoResponseSlice) Len() int {
	return len(s)
}

// Swap swaps two items
func (s ProjectInfoResponseSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the first item is less than the second one
func (s ProjectInfoResponseSlice) Less(i, j int) bool {
	if s[i].DisplayOrder != s[j].DisplayOrder {
		return s[i].DisplayOrder < s[j].DisplayOrder
	}

	return s[i].Id < s[j].Id
}
