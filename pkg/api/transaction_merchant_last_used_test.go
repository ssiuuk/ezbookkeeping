package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func getMerchantLastUsed(t *testing.T, uid int64, bookId int64, query string) *models.TransactionMerchantLastUsedResponse {
	c := newSharedBookTestContext(http.MethodGet, "/api/v1/transactions/merchants/last_used.json?"+query, nil, uid, bookId)
	result, errResp := Transactions.TransactionMerchantLastUsedHandler(c)
	assert.Nil(t, errResp)

	return result.(*models.TransactionMerchantLastUsedResponse)
}

func TestTransactionMerchantLastUsedHandler_ReturnsLatestTransactionOfMerchant(t *testing.T) {
	env, cafeLunch := setupMoveBookTest(t)

	// a later purchase at the same shop with another card
	c := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/add.json", &models.TransactionCreateRequest{
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:      cafeLunch.CategoryId,
		Time:            time.Now().Unix() - 600,
		SourceAccountId: env.unusedAccountId,
		SourceAmount:    300,
		Merchant:        "Cafe",
	}, env.memberUid, env.memberUid)
	_, errResp := Transactions.TransactionCreateHandler(c)
	assert.Nil(t, errResp)

	lastUsed := getMerchantLastUsed(t, env.memberUid, env.memberUid, "merchant=Cafe&type=3")
	assert.Equal(t, env.unusedAccountId, lastUsed.AccountId)
	assert.Equal(t, cafeLunch.CategoryId, lastUsed.CategoryId)

	// nothing is known for another shop, for income at this shop, or in another book
	assert.Equal(t, int64(0), getMerchantLastUsed(t, env.memberUid, env.memberUid, "merchant=Bakery&type=3").AccountId)
	assert.Equal(t, int64(0), getMerchantLastUsed(t, env.memberUid, env.memberUid, "merchant=Cafe&type=2").AccountId)
	assert.Equal(t, int64(0), getMerchantLastUsed(t, env.memberUid, env.bookId, "merchant=Cafe&type=3").AccountId)
}

func TestTransactionMerchantLastUsedHandler_RejectsTransfers(t *testing.T) {
	env, _ := setupMoveBookTest(t)

	c := newSharedBookTestContext(http.MethodGet, "/api/v1/transactions/merchants/last_used.json?merchant=Cafe&type=4", nil, env.memberUid, env.memberUid)
	_, errResp := Transactions.TransactionMerchantLastUsedHandler(c)
	assert.NotNil(t, errResp)
}
