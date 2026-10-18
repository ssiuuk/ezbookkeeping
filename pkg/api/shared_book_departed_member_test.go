package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
)

// sharedBookTestEnv is a family book of an owner and a member, with a category the book can use and two accounts the member shares to the book
type sharedBookTestEnv struct {
	ownerUid        int64
	memberUid       int64
	bookId          int64
	categoryId      int64
	usedAccountId   int64
	unusedAccountId int64
	oldTransaction  *models.TransactionInfoResponse
}

func setupSharedBookTestDatabase(t *testing.T) {
	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:          settings.Sqlite3DbType,
			DatabasePath:          filepath.Join(t.TempDir(), "ezbookkeeping.db"),
			MaxIdleConnection:     2,
			ConnectionMaxLifeTime: 14400,
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
	}

	settings.SetCurrentConfig(config)
	assert.Nil(t, datastore.InitializeDataStore(config))
	assert.Nil(t, uuid.InitializeUuidGenerator(config))

	assert.Nil(t, datastore.Container.UserStore.SyncStructs(new(models.User), new(models.SharedBook), new(models.BookMember), new(models.SharedBookAccount)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction), new(models.TransactionCategory), new(models.TransactionTag), new(models.TransactionTagIndex), new(models.TransactionPictureInfo), new(models.TransactionTemplate), new(models.Project)))

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("notBlank", validators.NotBlank)
		_ = v.RegisterValidation("validUsername", validators.ValidUsername)
		_ = v.RegisterValidation("validAmountFilter", validators.ValidAmountFilter)
		_ = v.RegisterValidation("validTransactionAmount", validators.ValidTransactionAmount)
		_ = v.RegisterValidation("validTagFilter", validators.ValidTagFilter)
	}
}

func newSharedBookTestContext(method string, target string, body any, uid int64, bookId int64) *core.WebContext {
	var reader io.Reader

	if body != nil {
		content, _ := json.Marshal(body)
		reader = bytes.NewReader(content)
	}

	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest(method, target, reader)
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Request.Header.Set(core.ClientTimezoneOffsetHeaderName, "0")

	c := core.WrapWebContext(ginCtx, nil)
	c.SetTokenClaims(&core.UserTokenClaims{Uid: uid})
	c.SetCurrentBookUid(bookId)

	return c
}

func insertSharedBookTestUser(t *testing.T, username string, nickname string) int64 {
	user := &models.User{
		Uid:                  uuid.Container.GenerateUuid(uuid.UUID_TYPE_USER),
		Username:             username,
		Email:                username + "@example.com",
		Nickname:             nickname,
		Password:             "password",
		Salt:                 "salt",
		TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL,
		DefaultCurrency:      "USD",
		CreatedUnixTime:      time.Now().Unix(),
	}

	_, err := datastore.Container.UserStore.Query(core.NewNullContext(), 0).Insert(user)
	assert.Nil(t, err)

	return user.Uid
}

func insertSharedBookTestCategory(t *testing.T, uid int64, parentId int64, name string) int64 {
	category := &models.TransactionCategory{
		CategoryId:       uuid.Container.GenerateUuid(uuid.UUID_TYPE_CATEGORY),
		Uid:              uid,
		Type:             models.CATEGORY_TYPE_EXPENSE,
		ParentCategoryId: parentId,
		Name:             name,
		Icon:             1,
		Color:            "000000",
	}

	_, err := datastore.Container.UserDataStore.Query(core.NewNullContext(), 0).Insert(category)
	assert.Nil(t, err)

	return category.CategoryId
}

func createSharedBookTestAccount(t *testing.T, uid int64, name string) int64 {
	account := &models.Account{
		Uid:      uid,
		Name:     name,
		Category: models.ACCOUNT_CATEGORY_CREDIT_CARD,
		Type:     models.ACCOUNT_TYPE_SINGLE_ACCOUNT,
		Icon:     1,
		Color:    "000000",
		Currency: "USD",
	}

	assert.Nil(t, services.Accounts.CreateAccounts(core.NewNullContext(), account, 0, nil, nil, time.UTC))

	return account.AccountId
}

func shareSharedBookTestAccount(t *testing.T, env *sharedBookTestEnv, uid int64, accountId int64) *errs.Error {
	c := newSharedBookTestContext(http.MethodPost, "/api/v1/books/accounts/share.json", &models.BookAccountShareRequest{AccountId: accountId}, uid, env.bookId)
	_, errResp := Books.BookShareAccountHandler(c)
	return errResp
}

func addSharedBookTestExpense(env *sharedBookTestEnv, uid int64, accountId int64, amount int64, comment string) (*models.TransactionInfoResponse, *errs.Error) {
	c := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/add.json", &models.TransactionCreateRequest{
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:      env.categoryId,
		Time:            time.Now().Unix() - 3600,
		SourceAccountId: accountId,
		SourceAmount:    amount,
		Comment:         comment,
	}, uid, env.bookId)

	result, errResp := Transactions.TransactionCreateHandler(c)

	if errResp != nil {
		return nil, errResp
	}

	return result.(*models.TransactionInfoResponse), nil
}

func getSharedBookTestAccountList(t *testing.T, uid int64, bookId int64, includeRevokedShared bool) models.AccountInfoResponseSlice {
	target := "/api/v1/accounts/list.json?visible_only=false"

	if includeRevokedShared {
		target += "&include_revoked_shared=true"
	}

	result, errResp := Accounts.AccountListHandler(newSharedBookTestContext(http.MethodGet, target, nil, uid, bookId))
	assert.Nil(t, errResp)

	return result.(models.AccountInfoResponseSlice)
}

func findSharedBookTestAccount(accounts models.AccountInfoResponseSlice, accountId int64) *models.AccountInfoResponse {
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Id == accountId {
			return accounts[i]
		}
	}

	return nil
}

func getSharedBookTestTransactionList(t *testing.T, uid int64, bookId int64) models.TransactionInfoResponseSlice {
	result, errResp := Transactions.TransactionListHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/transactions/list.json?count=50&trim_account=false", nil, uid, bookId))
	assert.Nil(t, errResp)

	return result.(*models.TransactionInfoPageWrapperResponse).Items
}

func getSharedBookTestSharingRow(t *testing.T, bookId int64, accountId int64) *models.SharedBookAccount {
	sharedAccount := &models.SharedBookAccount{}
	has, err := datastore.Container.UserStore.Query(core.NewNullContext(), 0).Where("book_id=? AND account_id=?", bookId, accountId).Get(sharedAccount)
	assert.Nil(t, err)

	if !has {
		return nil
	}

	return sharedAccount
}

func getSharedBookTestAccountBalance(t *testing.T, uid int64, accountId int64) int64 {
	account, err := services.Accounts.GetAccountByAccountId(core.NewNullContext(), uid, accountId)
	assert.Nil(t, err)

	return account.Balance
}

// setupSharedBookWithMemberAccount builds the owner's reported scenario: a member shares two accounts to a family book and records an expense on one of them
func setupSharedBookWithMemberAccount(t *testing.T) *sharedBookTestEnv {
	setupSharedBookTestDatabase(t)
	c := core.NewNullContext()

	env := &sharedBookTestEnv{
		ownerUid:  insertSharedBookTestUser(t, "owner", "Owner"),
		memberUid: insertSharedBookTestUser(t, "member", "Bob"),
	}

	book, err := services.Books.CreateSharedBook(c, env.ownerUid, "Family")
	assert.Nil(t, err)
	env.bookId = book.BookId

	_, err = services.Books.AddBookMember(c, book, env.memberUid)
	assert.Nil(t, err)

	parentCategoryId := insertSharedBookTestCategory(t, env.bookId, models.LevelOneTransactionCategoryParentId, "Food")
	env.categoryId = insertSharedBookTestCategory(t, env.bookId, parentCategoryId, "Groceries")

	env.usedAccountId = createSharedBookTestAccount(t, env.memberUid, "Visa")
	env.unusedAccountId = createSharedBookTestAccount(t, env.memberUid, "Debit")

	assert.Nil(t, shareSharedBookTestAccount(t, env, env.memberUid, env.usedAccountId))
	assert.Nil(t, shareSharedBookTestAccount(t, env, env.memberUid, env.unusedAccountId))

	transaction, errResp := addSharedBookTestExpense(env, env.memberUid, env.usedAccountId, 1200, "groceries on bob's visa")
	assert.Nil(t, errResp)
	assert.NotNil(t, transaction)
	env.oldTransaction = transaction

	assert.Equal(t, int64(-1200), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))

	// before the member departs, the book shows the shared account with its balance
	account := findSharedBookTestAccount(getSharedBookTestAccountList(t, env.ownerUid, env.bookId, false), env.usedAccountId)
	assert.NotNil(t, account)
	assert.Equal(t, "-1200", account.Balance)

	return env
}

// assertDepartedMemberAccountHasNoLiveAccess checks what the book can still do with the account of a member who is no longer in it
func assertDepartedMemberAccountHasNoLiveAccess(t *testing.T, env *sharedBookTestEnv) {
	// the sharing of the account used by a book transaction is revoked, the unused one is removed
	usedSharing := getSharedBookTestSharingRow(t, env.bookId, env.usedAccountId)
	assert.NotNil(t, usedSharing)
	assert.True(t, usedSharing.IsRevoked())
	assert.Nil(t, getSharedBookTestSharingRow(t, env.bookId, env.unusedAccountId))

	// a remaining member can no longer record a new transaction against the departed member's account
	_, errResp := addSharedBookTestExpense(env, env.ownerUid, env.usedAccountId, 500, "new expense on departed member's visa")
	assert.Equal(t, errs.ErrSharedAccountRevoked, errResp)

	_, errResp = addSharedBookTestExpense(env, env.ownerUid, env.unusedAccountId, 500, "new expense on departed member's debit")
	assert.Equal(t, errs.ErrSourceAccountNotFound, errResp)

	// nor change or delete the existing transaction, which would move money on the departed member's account
	modifyCtx := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/modify.json", &models.TransactionModifyRequest{
		Id:              env.oldTransaction.Id,
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:      env.categoryId,
		Time:            env.oldTransaction.Time,
		SourceAccountId: env.usedAccountId,
		SourceAmount:    99999,
		Comment:         "changed",
	}, env.ownerUid, env.bookId)
	_, errResp = Transactions.TransactionModifyHandler(modifyCtx)
	assert.Equal(t, errs.ErrSharedAccountRevoked, errResp)

	deleteCtx := newSharedBookTestContext(http.MethodPost, "/api/v1/transactions/delete.json", &models.TransactionDeleteRequest{Id: env.oldTransaction.Id}, env.ownerUid, env.bookId)
	_, errResp = Transactions.TransactionDeleteHandler(deleteCtx)
	assert.Equal(t, errs.ErrSharedAccountRevoked, errResp)

	// the service refuses it too, whichever way a transaction gets there
	err := services.Transactions.CreateTransaction(core.NewNullContext(), &models.Transaction{
		Uid:             env.bookId,
		CreatedByUid:    env.ownerUid,
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:      env.categoryId,
		TransactionTime: env.oldTransaction.Time*1000 + 5,
		AccountId:       env.usedAccountId,
		Amount:          300,
	}, nil, nil)
	assert.Equal(t, errs.ErrSharedAccountRevoked, err)

	assert.Equal(t, int64(-1200), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))

	// the account and its balance are no longer in the book's account list
	assert.Nil(t, findSharedBookTestAccount(getSharedBookTestAccountList(t, env.ownerUid, env.bookId, false), env.usedAccountId))
	assert.Nil(t, findSharedBookTestAccount(getSharedBookTestAccountList(t, env.ownerUid, env.bookId, false), env.unusedAccountId))

	accounts, err := services.Accounts.GetAllAccountsOfBook(core.NewNullContext(), env.bookId)
	assert.Nil(t, err)

	for i := 0; i < len(accounts); i++ {
		assert.NotEqual(t, env.usedAccountId, accounts[i].AccountId)
	}

	// the web app can still name the account of the existing transactions, but sees no balance and cannot choose it
	revokedAccount := findSharedBookTestAccount(getSharedBookTestAccountList(t, env.ownerUid, env.bookId, true), env.usedAccountId)
	assert.NotNil(t, revokedAccount)
	assert.True(t, revokedAccount.SharingRevoked)
	assert.True(t, revokedAccount.Hidden)
	assert.Equal(t, "0", revokedAccount.Balance)
	assert.Equal(t, "Visa (Bob)", revokedAccount.Name)

	// the existing transaction still lists with its account, read only
	transactions := getSharedBookTestTransactionList(t, env.ownerUid, env.bookId)
	assert.Equal(t, 1, len(transactions))
	assert.Equal(t, env.oldTransaction.Id, transactions[0].Id)
	assert.Equal(t, env.usedAccountId, transactions[0].SourceAccountId)
	assert.NotNil(t, transactions[0].SourceAccount)
	assert.Equal(t, "Visa", transactions[0].SourceAccount.Name)
	assert.Equal(t, "0", transactions[0].SourceAccount.Balance)
	assert.False(t, transactions[0].Editable)

	getCtx := newSharedBookTestContext(http.MethodGet, "/api/v1/transactions/get.json?id="+utils.Int64ToString(env.oldTransaction.Id), nil, env.ownerUid, env.bookId)
	transactionResult, errResp := Transactions.TransactionGetHandler(getCtx)
	assert.Nil(t, errResp)
	assert.False(t, transactionResult.(*models.TransactionInfoResponse).Editable)

	// the departed member keeps their own account and balance
	ownAccounts := getSharedBookTestAccountList(t, env.memberUid, env.memberUid, false)
	ownAccount := findSharedBookTestAccount(ownAccounts, env.usedAccountId)
	assert.NotNil(t, ownAccount)
	assert.Equal(t, "-1200", ownAccount.Balance)
}

func TestSharedBookRemoveMember_RevokesLiveAccessToMemberAccount(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)

	c := newSharedBookTestContext(http.MethodPost, "/api/v1/books/members/remove.json", &models.BookMemberRemoveRequest{MemberId: env.memberUid}, env.ownerUid, env.bookId)
	_, errResp := Books.BookMemberRemoveHandler(c)
	assert.Nil(t, errResp)

	assertDepartedMemberAccountHasNoLiveAccess(t, env)
}

func TestSharedBookLeave_RevokesLiveAccessToMemberAccount(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)

	c := newSharedBookTestContext(http.MethodPost, "/api/v1/books/leave.json", &models.BookLeaveRequest{BookId: env.bookId}, env.memberUid, env.memberUid)
	_, errResp := Books.BookLeaveHandler(c)
	assert.Nil(t, errResp)

	assertDepartedMemberAccountHasNoLiveAccess(t, env)
}

func TestSharedBookRejoin_SharingAgainReactivatesAccount(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)
	c := core.NewNullContext()

	assert.Nil(t, services.Books.RemoveBookMember(c, env.bookId, env.memberUid))
	assert.True(t, getSharedBookTestSharingRow(t, env.bookId, env.usedAccountId).IsRevoked())

	book, err := services.Books.GetSharedBookById(c, env.bookId)
	assert.Nil(t, err)
	_, err = services.Books.AddBookMember(c, book, env.memberUid)
	assert.Nil(t, err)

	// rejoining alone does not give the book the account back
	_, errResp := addSharedBookTestExpense(env, env.ownerUid, env.usedAccountId, 500, "before sharing again")
	assert.Equal(t, errs.ErrSharedAccountRevoked, errResp)

	assert.Nil(t, shareSharedBookTestAccount(t, env, env.memberUid, env.usedAccountId))
	assert.False(t, getSharedBookTestSharingRow(t, env.bookId, env.usedAccountId).IsRevoked())
	assert.Equal(t, errs.ErrAccountAlreadySharedToBook, shareSharedBookTestAccount(t, env, env.memberUid, env.usedAccountId))

	_, errResp = addSharedBookTestExpense(env, env.ownerUid, env.usedAccountId, 500, "after sharing again")
	assert.Nil(t, errResp)
	assert.Equal(t, int64(-1700), getSharedBookTestAccountBalance(t, env.memberUid, env.usedAccountId))

	account := findSharedBookTestAccount(getSharedBookTestAccountList(t, env.ownerUid, env.bookId, true), env.usedAccountId)
	assert.NotNil(t, account)
	assert.False(t, account.SharingRevoked)
	assert.Equal(t, "-1700", account.Balance)
}

func TestSharedBookUnshareAccount_InUseAccountIsRevokedInsteadOfFailing(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)

	unshare := func(accountId int64) *errs.Error {
		c := newSharedBookTestContext(http.MethodPost, "/api/v1/books/accounts/unshare.json", &models.BookAccountShareRequest{AccountId: accountId}, env.memberUid, env.bookId)
		_, errResp := Books.BookUnshareAccountHandler(c)
		return errResp
	}

	assert.Nil(t, unshare(env.usedAccountId))
	assert.True(t, getSharedBookTestSharingRow(t, env.bookId, env.usedAccountId).IsRevoked())

	assert.Nil(t, unshare(env.unusedAccountId))
	assert.Nil(t, getSharedBookTestSharingRow(t, env.bookId, env.unusedAccountId))

	// a revoked sharing is not listed as shared any more and cannot be stopped twice
	assert.Equal(t, errs.ErrSharedAccountNotFound, unshare(env.usedAccountId))

	listResult, errResp := Books.BookSharedAccountListHandler(newSharedBookTestContext(http.MethodGet, "/api/v1/books/accounts/list.json", nil, env.memberUid, env.bookId))
	assert.Nil(t, errResp)
	assert.Equal(t, 0, len(listResult.([]*models.BookSharedAccountInfoResponse)))

	_, errResp = addSharedBookTestExpense(env, env.memberUid, env.usedAccountId, 500, "after stopping sharing")
	assert.Equal(t, errs.ErrSharedAccountRevoked, errResp)

	transactions := getSharedBookTestTransactionList(t, env.memberUid, env.bookId)
	assert.Equal(t, 1, len(transactions))
	assert.Equal(t, "Visa", transactions[0].SourceAccount.Name)
}
