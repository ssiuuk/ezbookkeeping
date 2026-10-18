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
