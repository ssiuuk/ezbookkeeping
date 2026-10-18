package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestGetRelatedTransferTransaction_KeepsProjectId(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:    1,
		Uid:              1,
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		RelatedId:        2,
		AccountId:        10,
		RelatedAccountId: 11,
		ProjectId:        1001,
	}

	relatedTransaction := Transactions.GetRelatedTransferTransaction(transaction)
	assert.NotNil(t, relatedTransaction)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_IN, relatedTransaction.Type)
	assert.Equal(t, int64(1001), relatedTransaction.ProjectId)
}

func TestGetRelatedUpdateColumns_KeepsProjectIdColumn(t *testing.T) {
	actualValue := Transactions.getRelatedUpdateColumns([]string{"account_id", "project_id", "amount"})
	assert.Equal(t, []string{"related_account_id", "project_id", "related_account_amount"}, actualValue)
}

func TestGetProjectMapByList(t *testing.T) {
	projects := []*models.Project{
		{ProjectId: 1, Name: "A"},
		{ProjectId: 2, Name: "B"},
	}

	actualValue := Projects.GetProjectMapByList(projects)
	assert.Equal(t, 2, len(actualValue))
	assert.Equal(t, "A", actualValue[1].Name)
	assert.Equal(t, "B", actualValue[2].Name)
}
