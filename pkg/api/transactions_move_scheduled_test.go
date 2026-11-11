package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestGetMovedScheduledTransaction_ExpenseKeepsItsOwnTimeAndAmount(t *testing.T) {
	moved := &models.Transaction{
		Uid:              200,
		Type:             models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:       2001,
		AccountId:        2002,
		Amount:           1000,
		ProjectId:        2003,
		CreatedByUid:     100,
		SourceTemplateId: 9,
	}

	other := &models.Transaction{
		TransactionId:     5,
		Uid:               100,
		Type:              models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:        1001,
		AccountId:         1002,
		TransactionTime:   123456,
		TimezoneUtcOffset: 60,
		Amount:            1250,
		Comment:           "October",
		Merchant:          "Nursery",
		ScheduledCreated:  true,
		SourceTemplateId:  9,
	}

	result := getMovedScheduledTransaction(other, moved)

	assert.Equal(t, int64(0), result.TransactionId)
	assert.Equal(t, int64(200), result.Uid)
	assert.Equal(t, int64(2001), result.CategoryId)
	assert.Equal(t, int64(2002), result.AccountId)
	assert.Equal(t, int64(2003), result.ProjectId)
	assert.Equal(t, int64(123456), result.TransactionTime)
	assert.Equal(t, int16(60), result.TimezoneUtcOffset)
	assert.Equal(t, int64(1250), result.Amount)
	assert.Equal(t, "October", result.Comment)
	assert.Equal(t, "Nursery", result.Merchant)
	assert.Equal(t, true, result.ScheduledCreated)
	assert.Equal(t, int64(9), result.SourceTemplateId)
	assert.Equal(t, int64(100), result.CreatedByUid)
	assert.Equal(t, int64(0), result.RelatedAccountId)
}

func TestGetMovedScheduledTransaction_Transfer(t *testing.T) {
	sameCurrency := &models.Transaction{Uid: 200, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, AccountId: 1, RelatedAccountId: 2, Amount: 1000, RelatedAccountAmount: 1000}
	otherCurrency := &models.Transaction{Uid: 200, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, AccountId: 1, RelatedAccountId: 3, Amount: 1000, RelatedAccountAmount: 9000}

	otherTransfer := &models.Transaction{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, Amount: 1500, RelatedAccountAmount: 13000}
	otherExpense := &models.Transaction{Type: models.TRANSACTION_DB_TYPE_EXPENSE, Amount: 1500}

	result := getMovedScheduledTransaction(otherTransfer, sameCurrency)
	assert.Equal(t, int64(2), result.RelatedAccountId)
	assert.Equal(t, int64(1500), result.Amount)
	assert.Equal(t, int64(1500), result.RelatedAccountAmount)

	result = getMovedScheduledTransaction(otherTransfer, otherCurrency)
	assert.Equal(t, int64(3), result.RelatedAccountId)
	assert.Equal(t, int64(1500), result.Amount)
	assert.Equal(t, int64(13000), result.RelatedAccountAmount)

	result = getMovedScheduledTransaction(otherExpense, otherCurrency)
	assert.Equal(t, int64(1000), result.Amount)
	assert.Equal(t, int64(9000), result.RelatedAccountAmount)
}
