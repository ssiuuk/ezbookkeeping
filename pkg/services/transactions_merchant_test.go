package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestGetRelatedTransferTransaction_KeepsMerchant(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:    1,
		Uid:              1,
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		RelatedId:        2,
		AccountId:        10,
		RelatedAccountId: 11,
		Merchant:         "Coffee Shop",
	}

	relatedTransaction := Transactions.GetRelatedTransferTransaction(transaction)
	assert.NotNil(t, relatedTransaction)
	assert.Equal(t, "Coffee Shop", relatedTransaction.Merchant)
}

func TestGetRelatedUpdateColumns_KeepsMerchantColumn(t *testing.T) {
	actualValue := Transactions.getRelatedUpdateColumns([]string{"amount", "merchant"})
	assert.Equal(t, []string{"related_account_amount", "merchant"}, actualValue)
}

func TestGetMerchantsSortedByLastUsedTime(t *testing.T) {
	lastUsedTimes := map[string]int64{
		"Tesco":  300,
		"Amazon": 500,
		"Boots":  300,
		"Argos":  100,
	}

	assert.Equal(t, []string{"Amazon", "Boots", "Tesco", "Argos"}, getMerchantsSortedByLastUsedTime(lastUsedTimes, 0))
	assert.Equal(t, []string{"Amazon", "Boots"}, getMerchantsSortedByLastUsedTime(lastUsedTimes, 2))
	assert.Equal(t, []string{}, getMerchantsSortedByLastUsedTime(map[string]int64{}, 10))
}

func TestGetMovedScheduledTemplateTransferAmounts(t *testing.T) {
	transfer := &models.TransactionTemplate{Type: models.TRANSACTION_TYPE_TRANSFER, Amount: 10000, RelatedAccountAmount: 11500}
	expense := &models.TransactionTemplate{Type: models.TRANSACTION_TYPE_EXPENSE, Amount: 10000}
	moved := &models.Transaction{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, Amount: 9000, RelatedAccountAmount: 10400}

	// moved to two accounts of the same currency, so the amounts must be equal
	amount, relatedAccountAmount := getMovedScheduledTemplateTransferAmounts(transfer, moved, true)
	assert.Equal(t, int64(10000), amount)
	assert.Equal(t, int64(10000), relatedAccountAmount)

	amount, relatedAccountAmount = getMovedScheduledTemplateTransferAmounts(transfer, moved, false)
	assert.Equal(t, int64(10000), amount)
	assert.Equal(t, int64(11500), relatedAccountAmount)

	amount, relatedAccountAmount = getMovedScheduledTemplateTransferAmounts(expense, moved, false)
	assert.Equal(t, int64(9000), amount)
	assert.Equal(t, int64(10400), relatedAccountAmount)
}
