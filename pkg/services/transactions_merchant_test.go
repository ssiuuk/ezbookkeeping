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
