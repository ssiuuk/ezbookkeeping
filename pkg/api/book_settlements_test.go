package api

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
)

// setupSettleUpTest builds a family book where the owner paid 3000 from their own shared account, the member paid 1200 from theirs,
// and 5000 was paid from an account of the book itself, which nobody is owed for
func setupSettleUpTest(t *testing.T) *sharedBookTestEnv {
	env := setupSharedBookWithMemberAccount(t)

	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.BookSettlement)))

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("validCurrency", validators.ValidCurrency)
	}

	ownerAccountId := createSharedBookTestAccount(t, env.ownerUid, "Amex")
	assert.Nil(t, shareSharedBookTestAccount(t, env, env.ownerUid, ownerAccountId))

	_, errResp := addSharedBookTestExpense(env, env.ownerUid, ownerAccountId, 3000, "dinner on the owner's amex")
	assert.Nil(t, errResp)

	bookAccountId := createSharedBookTestAccount(t, env.bookId, "Family Cash")
	_, errResp = addSharedBookTestExpense(env, env.memberUid, bookAccountId, 5000, "rent from the family cash")
	assert.Nil(t, errResp)

	return env
}

func getSettleUpTestResult(t *testing.T, uid int64, bookId int64) *models.BookSettleUpResponse {
	result, errResp := Books.BookSettleUpHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/books/settle_up.json", nil, uid, bookId))
	assert.Nil(t, errResp)

	return result.(*models.BookSettleUpResponse)
}

func findSettleUpTestBalance(currency *models.BookSettleUpCurrencyResponse, uid int64) *models.BookSettleUpMemberBalance {
	for i := 0; i < len(currency.Balances); i++ {
		if currency.Balances[i].Uid == uid {
			return currency.Balances[i]
		}
	}

	return nil
}

func addSettleUpTestSettlement(uid int64, bookId int64, req *models.BookSettlementCreateRequest) (*models.BookSettlementInfoResponse, *errs.Error) {
	result, errResp := Books.BookSettlementCreateHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/books/settlements/add.json", req, uid, bookId))

	if errResp != nil {
		return nil, errResp
	}

	return result.(*models.BookSettlementInfoResponse), nil
}

func TestBookSettleUp_RecordingSuggestedPaymentSettlesBalances(t *testing.T) {
	env := setupSettleUpTest(t)

	settleUp := getSettleUpTestResult(t, env.memberUid, env.bookId)
	assert.Equal(t, "USD", settleUp.DefaultCurrency)
	assert.Equal(t, 2, len(settleUp.Members))
	assert.Equal(t, env.ownerUid, settleUp.Members[0].Id)
	assert.True(t, settleUp.Members[0].IsOwner)
	assert.Equal(t, env.memberUid, settleUp.Members[1].Id)
	assert.Equal(t, "Bob", settleUp.Members[1].Nickname)
	assert.False(t, settleUp.Members[1].IsFormer)

	// the expense from the book's own account is left out, the other 4200 is split equally
	assert.Equal(t, 1, len(settleUp.Currencies))
	usd := settleUp.Currencies[0]
	assert.Equal(t, "USD", usd.Currency)
	assert.Equal(t, int64(4200), usd.TotalExpense)

	ownerBalance := findSettleUpTestBalance(usd, env.ownerUid)
	assert.Equal(t, int64(3000), ownerBalance.Paid)
	assert.Equal(t, int64(2100), ownerBalance.FairShare)
	assert.Equal(t, int64(900), ownerBalance.Balance)

	memberBalance := findSettleUpTestBalance(usd, env.memberUid)
	assert.Equal(t, int64(1200), memberBalance.Paid)
	assert.Equal(t, int64(2100), memberBalance.FairShare)
	assert.Equal(t, int64(-900), memberBalance.Balance)

	assert.Equal(t, []*models.BookSettleUpPayment{{FromUid: env.memberUid, ToUid: env.ownerUid, Amount: 900}}, usd.SuggestedPayments)

	// any member can mark the suggested payment as settled
	settlement, errResp := addSettleUpTestSettlement(env.memberUid, env.bookId, &models.BookSettlementCreateRequest{
		FromUid:  env.memberUid,
		ToUid:    env.ownerUid,
		Amount:   900,
		Currency: "USD",
		Comment:  "bank transfer",
	})
	assert.Nil(t, errResp)
	assert.NotZero(t, settlement.Id)
	assert.NotZero(t, settlement.Time)
	assert.Equal(t, env.memberUid, settlement.CreatedByUid)

	settleUp = getSettleUpTestResult(t, env.ownerUid, env.bookId)
	usd = settleUp.Currencies[0]
	assert.Equal(t, int64(0), findSettleUpTestBalance(usd, env.ownerUid).Balance)
	assert.Equal(t, int64(900), findSettleUpTestBalance(usd, env.ownerUid).Received)
	assert.Equal(t, int64(0), findSettleUpTestBalance(usd, env.memberUid).Balance)
	assert.Equal(t, int64(900), findSettleUpTestBalance(usd, env.memberUid).Sent)
	assert.Equal(t, 0, len(usd.SuggestedPayments))

	listResult, errResp := Books.BookSettlementListHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/books/settlements/list.json", nil, env.ownerUid, env.bookId))
	assert.Nil(t, errResp)
	settlements := listResult.([]*models.BookSettlementInfoResponse)
	assert.Equal(t, 1, len(settlements))
	assert.Equal(t, settlement.Id, settlements[0].Id)
	assert.Equal(t, "bank transfer", settlements[0].Comment)

	// deleting the settlement brings the balances back
	_, errResp = Books.BookSettlementDeleteHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/books/settlements/delete.json", &models.BookSettlementDeleteRequest{Id: settlement.Id}, env.ownerUid, env.bookId))
	assert.Nil(t, errResp)

	_, errResp = Books.BookSettlementDeleteHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/books/settlements/delete.json", &models.BookSettlementDeleteRequest{Id: settlement.Id}, env.ownerUid, env.bookId))
	assert.Equal(t, errs.ErrBookSettlementNotFound, errResp)

	settleUp = getSettleUpTestResult(t, env.ownerUid, env.bookId)
	assert.Equal(t, int64(-900), findSettleUpTestBalance(settleUp.Currencies[0], env.memberUid).Balance)
}

func TestBookSettleUp_FormerMemberWhoPaidKeepsBalance(t *testing.T) {
	env := setupSettleUpTest(t)

	assert.Nil(t, services.Books.RemoveBookMember(core.NewNullContext(), env.bookId, env.memberUid))

	settleUp := getSettleUpTestResult(t, env.ownerUid, env.bookId)
	assert.Equal(t, 2, len(settleUp.Members))
	assert.Equal(t, env.memberUid, settleUp.Members[1].Id)
	assert.True(t, settleUp.Members[1].IsFormer)
	assert.Equal(t, int64(-900), findSettleUpTestBalance(settleUp.Currencies[0], env.memberUid).Balance)

	// the former member can still be settled with
	_, errResp := addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{
		FromUid:  env.memberUid,
		ToUid:    env.ownerUid,
		Amount:   900,
		Currency: "USD",
	})
	assert.Nil(t, errResp)

	settleUp = getSettleUpTestResult(t, env.ownerUid, env.bookId)
	assert.Equal(t, int64(0), findSettleUpTestBalance(settleUp.Currencies[0], env.memberUid).Balance)
}

func TestBookSettleUp_InvalidRequests(t *testing.T) {
	env := setupSettleUpTest(t)
	strangerUid := insertSharedBookTestUser(t, "stranger", "Stranger")

	// a personal book has nobody to settle up with
	_, errResp := Books.BookSettleUpHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/books/settle_up.json", nil, env.ownerUid, env.ownerUid))
	assert.Equal(t, errs.ErrNotInSharedBook, errResp)

	_, errResp = addSettleUpTestSettlement(env.ownerUid, env.ownerUid, &models.BookSettlementCreateRequest{FromUid: env.memberUid, ToUid: env.ownerUid, Amount: 100, Currency: "USD"})
	assert.Equal(t, errs.ErrNotInSharedBook, errResp)

	_, errResp = addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{FromUid: env.ownerUid, ToUid: env.ownerUid, Amount: 100, Currency: "USD"})
	assert.Equal(t, errs.ErrBookSettlementSameMember, errResp)

	_, errResp = addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{FromUid: strangerUid, ToUid: env.ownerUid, Amount: 100, Currency: "USD"})
	assert.Equal(t, errs.ErrBookSettlementMemberInvalid, errResp)

	_, errResp = addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{FromUid: env.memberUid, ToUid: env.ownerUid, Amount: 0, Currency: "USD"})
	assert.NotNil(t, errResp)

	_, errResp = addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{FromUid: env.memberUid, ToUid: env.ownerUid, Amount: 100, Currency: "XXXX"})
	assert.NotNil(t, errResp)

	// a member who neither recorded, paid nor received a settlement cannot delete it, unless they own the book
	carolUid := insertSharedBookTestUser(t, "carol", "Carol")
	book, err := services.Books.GetSharedBookById(core.NewNullContext(), env.bookId)
	assert.Nil(t, err)
	_, err = services.Books.AddBookMember(core.NewNullContext(), book, carolUid)
	assert.Nil(t, err)

	settlement, errResp := addSettleUpTestSettlement(env.ownerUid, env.bookId, &models.BookSettlementCreateRequest{FromUid: env.memberUid, ToUid: env.ownerUid, Amount: 100, Currency: "USD"})
	assert.Nil(t, errResp)

	_, errResp = Books.BookSettlementDeleteHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/books/settlements/delete.json", &models.BookSettlementDeleteRequest{Id: settlement.Id}, carolUid, env.bookId))
	assert.Equal(t, errs.ErrCannotDeleteBookSettlement, errResp)

	_, errResp = Books.BookSettlementDeleteHandler(newSharedBookTestContext(http.MethodPost, "/api/v1/books/settlements/delete.json", &models.BookSettlementDeleteRequest{Id: settlement.Id}, env.memberUid, env.bookId))
	assert.Nil(t, errResp)
}
