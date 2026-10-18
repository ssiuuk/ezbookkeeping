package models

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func TestProjectToProjectInfoResponse_WithoutStatistic(t *testing.T) {
	project := &Project{
		ProjectId:    1001,
		Uid:          1,
		Name:         "House Renovation",
		Comment:      "Kitchen and bathroom",
		DisplayOrder: 2,
		Hidden:       true,
	}

	actualValue := project.ToProjectInfoResponse(nil)
	assert.Equal(t, int64(1001), actualValue.Id)
	assert.Equal(t, "House Renovation", actualValue.Name)
	assert.Equal(t, "Kitchen and bathroom", actualValue.Comment)
	assert.Equal(t, int32(2), actualValue.DisplayOrder)
	assert.Equal(t, true, actualValue.Hidden)
	assert.Equal(t, int64(0), actualValue.TransactionCount)
	assert.Equal(t, int64(0), actualValue.FirstTransactionTime)
	assert.Equal(t, int64(0), actualValue.LastTransactionTime)
}

func TestProjectToProjectInfoResponse_WithStatistic(t *testing.T) {
	project := &Project{
		ProjectId: 1001,
		Name:      "House Renovation",
	}

	statistic := &ProjectTransactionStatistic{
		ProjectId:          1001,
		TransactionCount:   3,
		MinTransactionTime: utils.GetMinTransactionTimeFromUnixTime(1700000000) + 5,
		MaxTransactionTime: utils.GetMinTransactionTimeFromUnixTime(1710000000),
	}

	actualValue := project.ToProjectInfoResponse(statistic)
	assert.Equal(t, int64(3), actualValue.TransactionCount)
	assert.Equal(t, int64(1700000000), actualValue.FirstTransactionTime)
	assert.Equal(t, int64(1710000000), actualValue.LastTransactionTime)
}

func TestProjectToProjectInfoResponse_EmptyStatistic(t *testing.T) {
	project := &Project{ProjectId: 1001}

	actualValue := project.ToProjectInfoResponse(&ProjectTransactionStatistic{ProjectId: 1001})
	assert.Equal(t, int64(0), actualValue.TransactionCount)
	assert.Equal(t, int64(0), actualValue.FirstTransactionTime)
}

func TestProjectInfoResponseSliceLess(t *testing.T) {
	projectRespSlice := ProjectInfoResponseSlice{
		{Id: 3, DisplayOrder: 2},
		{Id: 2, DisplayOrder: 1},
		{Id: 1, DisplayOrder: 2},
		{Id: 4, DisplayOrder: 0},
	}

	sort.Sort(projectRespSlice)

	assert.Equal(t, int64(4), projectRespSlice[0].Id)
	assert.Equal(t, int64(2), projectRespSlice[1].Id)
	assert.Equal(t, int64(1), projectRespSlice[2].Id)
	assert.Equal(t, int64(3), projectRespSlice[3].Id)
}

func TestTransactionToTransactionInfoResponse_ProjectId(t *testing.T) {
	transaction := &Transaction{
		TransactionId:   1,
		Uid:             1,
		Type:            TRANSACTION_DB_TYPE_EXPENSE,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(1700000000),
		ProjectId:       1001,
	}

	actualValue := transaction.ToTransactionInfoResponse([]int64{}, true)
	assert.Equal(t, int64(1001), actualValue.ProjectId)

	transaction.ProjectId = 0
	actualValue = transaction.ToTransactionInfoResponse([]int64{}, true)
	assert.Equal(t, int64(0), actualValue.ProjectId)
}
