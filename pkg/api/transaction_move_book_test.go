package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

func insertMoveBookTestTag(t *testing.T, uid int64, name string) int64 {
	tag := &models.TransactionTag{
		TagId:           uuid.Container.GenerateUuid(uuid.UUID_TYPE_TAG),
		Uid:             uid,
		Name:            name,
		CreatedUnixTime: time.Now().Unix(),
	}

	_, err := datastore.Container.UserDataStore.Query(core.NewNullContext(), 0).Insert(tag)
	assert.Nil(t, err)

	return tag.TagId
}

func moveTransactionToBook(uid int64, targetBookId int64, sourceBookId int64, transactionId int64, req models.TransactionCreateRequest) (*models.TransactionInfoResponse, *errs.Error) {
	c := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/move/book.json", &models.TransactionMoveToBookRequest{
		TransactionCreateRequest: req,
		Id:                       transactionId,
		SourceBookId:             sourceBookId,
	}, uid, targetBookId)

	result, errResp := Transactions.TransactionMoveToBookHandler(c)

	if errResp != nil {
		return nil, errResp
	}

	return result.(*models.TransactionInfoResponse), nil
}

func findMoveBookTestTransaction(transactions models.TransactionInfoResponseSlice, comment string) *models.TransactionInfoResponse {
	for i := 0; i < len(transactions); i++ {
		if transactions[i].Comment == comment {
			return transactions[i]
		}
	}

	return nil
}

// setupMoveBookTest adds a personal expense of the member on the shared account, next to the family book expense of setupSharedBookWithMemberAccount
func setupMoveBookTest(t *testing.T) (*sharedBookTestEnv, *models.TransactionInfoResponse) {
	env := setupSharedBookWithMemberAccount(t)

	personalParentCategoryId := insertSharedBookTestCategory(t, env.memberUid, models.LevelOneTransactionCategoryParentId, "Food")
	personalCategoryId := insertSharedBookTestCategory(t, env.memberUid, personalParentCategoryId, "Dining")
	personalTagId := insertMoveBookTestTag(t, env.memberUid, "Lunch")

	c := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/add.json", &models.TransactionCreateRequest{
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:      personalCategoryId,
		Time:            time.Now().Unix() - 1800,
		SourceAccountId: env.usedAccountId,
		SourceAmount:    500,
		TagIds:          []string{utils.Int64ToString(personalTagId)},
		Comment:         "lunch for the family",
		Merchant:        "Cafe",
	}, env.memberUid, env.memberUid)

	result, errResp := Transactions.TransactionCreateHandler(c)
	assert.Nil(t, errResp)

	assert.Equal(t, int64(-1700), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))

	return env, result.(*models.TransactionInfoResponse)
}

func moveBookTestRequest(transaction *models.TransactionInfoResponse, categoryId int64, accountId int64, tagIds []string) models.TransactionCreateRequest {
	return models.TransactionCreateRequest{
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:      categoryId,
		Time:            transaction.Time,
		UtcOffset:       transaction.UtcOffset,
		SourceAccountId: accountId,
		SourceAmount:    transaction.SourceAmount,
		TagIds:          tagIds,
		Comment:         transaction.Comment,
		Merchant:        transaction.Merchant,
	}
}

func TestTransactionMoveToBook_MovesPersonalEntryToFamilyBook(t *testing.T) {
	env, personalTransaction := setupMoveBookTest(t)
	familyTagId := insertMoveBookTestTag(t, env.bookId, "Lunch")

	moved, errResp := moveTransactionToBook(env.memberUid, env.bookId, env.memberUid, personalTransaction.Id,
		moveBookTestRequest(personalTransaction, env.categoryId, env.usedAccountId, []string{utils.Int64ToString(familyTagId)}))
	assert.Nil(t, errResp)
	assert.NotNil(t, moved)
	assert.NotEqual(t, personalTransaction.Id, moved.Id)
	assert.Equal(t, env.categoryId, moved.CategoryId)
	assert.Equal(t, []string{utils.Int64ToString(familyTagId)}, moved.TagIds)
	assert.Equal(t, "Cafe", moved.Merchant)
	assert.Equal(t, env.memberUid, moved.CreatedByUid)

	// the entry left the personal book and is in the family book once
	assert.Nil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.memberUid, env.memberUid), "lunch for the family"))
	_, err := services.Transactions.GetTransactionByTransactionId(core.NewNullContext(), env.memberUid, personalTransaction.Id)
	assert.Equal(t, errs.ErrTransactionNotFound, err)

	familyTransaction := findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.ownerUid, env.bookId), "lunch for the family")
	assert.NotNil(t, familyTransaction)
	assert.Equal(t, moved.Id, familyTransaction.Id)

	// the account balance counts the expense once
	assert.Equal(t, int64(-1700), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))
}

func TestTransactionMoveToBook_MovesFamilyEntryToPersonalBook(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)
	personalParentCategoryId := insertSharedBookTestCategory(t, env.memberUid, models.LevelOneTransactionCategoryParentId, "Food")
	personalCategoryId := insertSharedBookTestCategory(t, env.memberUid, personalParentCategoryId, "Groceries")

	moved, errResp := moveTransactionToBook(env.memberUid, env.memberUid, env.bookId, env.oldTransaction.Id,
		moveBookTestRequest(env.oldTransaction, personalCategoryId, env.usedAccountId, nil))
	assert.Nil(t, errResp)
	assert.NotNil(t, moved)

	assert.Nil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.ownerUid, env.bookId), "groceries on bob's visa"))
	assert.NotNil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.memberUid, env.memberUid), "groceries on bob's visa"))
	assert.Equal(t, int64(-1200), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))
}

func TestTransactionMoveToBook_FailedMoveKeepsOriginalEntry(t *testing.T) {
	env, personalTransaction := setupMoveBookTest(t)
	cashAccountId := createSharedBookTestAccount(t, env.memberUid, "Cash")

	// the cash account is not shared to the family book, so the entry cannot be saved there
	_, errResp := moveTransactionToBook(env.memberUid, env.bookId, env.memberUid, personalTransaction.Id,
		moveBookTestRequest(personalTransaction, env.categoryId, cashAccountId, nil))
	assert.NotNil(t, errResp)

	assert.NotNil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.memberUid, env.memberUid), "lunch for the family"))
	assert.Nil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.ownerUid, env.bookId), "lunch for the family"))
	assert.Equal(t, int64(-1700), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))
	assert.Equal(t, int64(0), getSharedBookTestAccountBalance(t, env.memberUid, cashAccountId))
}

func TestTransactionMoveToBook_RejectsSameBook(t *testing.T) {
	env, personalTransaction := setupMoveBookTest(t)

	_, errResp := moveTransactionToBook(env.memberUid, env.memberUid, env.memberUid, personalTransaction.Id,
		moveBookTestRequest(personalTransaction, personalTransaction.CategoryId, env.usedAccountId, nil))
	assert.Equal(t, errs.ErrCannotMoveTransactionToSameBook, errResp)
}

func TestTransactionMoveToBook_RejectsBookTheUserCannotAccess(t *testing.T) {
	env, personalTransaction := setupMoveBookTest(t)

	// the owner of the family book is not a member of the member's personal book
	_, errResp := moveTransactionToBook(env.ownerUid, env.bookId, env.memberUid, personalTransaction.Id,
		moveBookTestRequest(personalTransaction, env.categoryId, env.usedAccountId, nil))
	assert.Equal(t, errs.ErrBookAccessDenied, errResp)

	assert.NotNil(t, findMoveBookTestTransaction(getSharedBookTestTransactionList(t, env.memberUid, env.memberUid), "lunch for the family"))
}
