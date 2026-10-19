package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// the json contents below stand in for the large language model, no model is called in these tests

func newReceiptRecognitionTestMaps() (map[string]*models.Account, map[string]*models.TransactionCategory, map[string]*models.TransactionCategory, map[string]*models.TransactionCategory, map[string]*models.TransactionTag) {
	accountMap := map[string]*models.Account{
		"Visa":   {AccountId: 101, Name: "Visa"},
		"Wallet": {AccountId: 102, Name: "Wallet"},
	}
	expenseCategoryMap := map[string]*models.TransactionCategory{
		"Groceries":  {CategoryId: 201, Name: "Groceries", Type: models.CATEGORY_TYPE_EXPENSE},
		"Restaurant": {CategoryId: 202, Name: "Restaurant", Type: models.CATEGORY_TYPE_EXPENSE},
	}
	incomeCategoryMap := map[string]*models.TransactionCategory{
		"Refund": {CategoryId: 301, Name: "Refund", Type: models.CATEGORY_TYPE_INCOME},
	}
	transferCategoryMap := map[string]*models.TransactionCategory{}
	tagMap := map[string]*models.TransactionTag{
		"Weekly Shop": {TagId: 401, Name: "Weekly Shop"},
	}

	return accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap
}

func parseReceiptRecognitionTestContent(content string) (*models.RecognizedTransactionResponse, *errs.Error) {
	c := newSharedBookTestContext(http.MethodPost, "/api/v1/llm/transactions/recognize_receipt_image.json", nil, 1, 1)
	accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap := newReceiptRecognitionTestMaps()

	return LargeLanguageModels.parseRecognizedReceiptImageContent(c, 1, time.UTC, content, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}

func TestParseRecognizedReceiptImageContent_FillsAmountTimeMerchantCategoryAndAccount(t *testing.T) {
	response, errResp := parseReceiptRecognitionTestContent(`{"type":"expense","time":"2026-10-18 12:34:56","amount":"23.45","account":"Visa","category":"Groceries","tags":["Weekly Shop","Unknown"],"description":"milk and bread","merchant":"  Corner Market  "}`)

	assert.Nil(t, errResp)
	assert.Equal(t, models.TRANSACTION_TYPE_EXPENSE, response.Type)
	assert.Equal(t, time.Date(2026, 10, 18, 12, 34, 56, 0, time.UTC).Unix(), response.Time)
	assert.Equal(t, int64(2345), response.SourceAmount)
	assert.Equal(t, int64(101), response.SourceAccountId)
	assert.Equal(t, int64(201), response.CategoryId)
	assert.Equal(t, []string{"401"}, response.TagIds)
	assert.Equal(t, "milk and bread", response.Comment)
	assert.Equal(t, "Corner Market", response.Merchant)
}

func TestParseRecognizedReceiptImageContent_MatchesNamesIgnoringCase(t *testing.T) {
	response, errResp := parseReceiptRecognitionTestContent(`{"type":"Expense","amount":"8","account":" wallet ","category":"restaurant","merchant":"Noodle Bar"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, models.TRANSACTION_TYPE_EXPENSE, response.Type)
	assert.Equal(t, int64(800), response.SourceAmount)
	assert.Equal(t, int64(102), response.SourceAccountId)
	assert.Equal(t, int64(202), response.CategoryId)
	assert.Equal(t, "Noodle Bar", response.Merchant)
}

func TestParseRecognizedReceiptImageContent_IgnoresNamesNotInBook(t *testing.T) {
	response, errResp := parseReceiptRecognitionTestContent(`{"type":"expense","amount":"5.00","account":"Amex","category":"Coffee"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, int64(500), response.SourceAmount)
	assert.Equal(t, int64(0), response.SourceAccountId)
	assert.Equal(t, int64(0), response.CategoryId)
}

func TestParseRecognizedReceiptImageContent_UsesCategoryOfRecognizedType(t *testing.T) {
	response, errResp := parseReceiptRecognitionTestContent(`{"type":"income","amount":"5.00","category":"Groceries"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, models.TRANSACTION_TYPE_INCOME, response.Type)
	assert.Equal(t, int64(0), response.CategoryId)

	response, errResp = parseReceiptRecognitionTestContent(`{"type":"income","amount":"5.00","category":"Refund"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, int64(301), response.CategoryId)
}

func TestParseRecognizedReceiptImageContent_TreatsTotalWithoutTypeAsExpense(t *testing.T) {
	response, errResp := parseReceiptRecognitionTestContent(`{"amount":"$1,234.50","time":"2026/10/18","merchant":"Hardware Store"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, models.TRANSACTION_TYPE_EXPENSE, response.Type)
	assert.Equal(t, int64(123450), response.SourceAmount)
	assert.Equal(t, time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC).Unix(), response.Time)
	assert.Equal(t, "Hardware Store", response.Merchant)
}

func TestParseRecognizedReceiptImageContent_NoTransactionInformation(t *testing.T) {
	_, errResp := parseReceiptRecognitionTestContent(`{"merchant":"Corner Market"}`)
	assert.Equal(t, errs.ErrNoTransactionInformation, errResp)

	_, errResp = parseReceiptRecognitionTestContent(`not json`)
	assert.NotNil(t, errResp)

	_, errResp = parseReceiptRecognitionTestContent(`{"type":"expense","amount":"about twelve"}`)
	assert.Equal(t, errs.ErrOperationFailed, errResp)
}

func TestParseRecognizedReceiptImageContent_TruncatesLongMerchant(t *testing.T) {
	longMerchant := ""

	for i := 0; i < models.TransactionMerchantMaxLength+10; i++ {
		longMerchant += "m"
	}

	response, errResp := parseReceiptRecognitionTestContent(`{"type":"expense","amount":"1","merchant":"` + longMerchant + `"}`)

	assert.Nil(t, errResp)
	assert.Equal(t, models.TransactionMerchantMaxLength, len(response.Merchant))
}

func TestRecognizedTransactionResponse_KeepsExistingJsonFields(t *testing.T) {
	content, err := json.Marshal(&models.RecognizedTransactionResponse{
		Type:            models.TRANSACTION_TYPE_EXPENSE,
		Time:            1700000000,
		CategoryId:      201,
		SourceAccountId: 101,
		SourceAmount:    2345,
		TagIds:          []string{"401"},
		Comment:         "milk",
		Merchant:        "Corner Market",
	})
	assert.Nil(t, err)

	var fields map[string]any
	assert.Nil(t, json.Unmarshal(content, &fields))

	assert.Equal(t, float64(models.TRANSACTION_TYPE_EXPENSE), fields["type"])
	assert.Equal(t, float64(1700000000), fields["time"])
	assert.Equal(t, "201", fields["categoryId"])
	assert.Equal(t, "101", fields["sourceAccountId"])
	assert.Equal(t, float64(2345), fields["sourceAmount"])
	assert.Equal(t, []any{"401"}, fields["tagIds"])
	assert.Equal(t, "milk", fields["comment"])
	assert.Equal(t, "Corner Market", fields["merchant"])
}

func TestNormalizeRecognizedAmount(t *testing.T) {
	cases := map[string]string{
		"12.34":     "12.34",
		"-12.34":    "-12.34",
		"$12.34":    "12.34",
		"12.34 USD": "12.34",
		"¥ 1,234":   "1234",
		"1,234.56":  "1234.56",
		"1.234,56":  "1234.56",
		"12,50 €":   "12.50",
		"-$5":       "-5",
		"1 234.00":  "1234.00",
		"abc":       "abc",
		"":          "",
	}

	for input, expected := range cases {
		assert.Equal(t, expected, normalizeRecognizedAmount(input), input)
	}
}

func TestParseRecognizedReceiptImageContent_UsesFamilyBookCategoriesAndAccounts(t *testing.T) {
	env := setupSharedBookWithMemberAccount(t)

	// a category only in the member's personal book must not be suggested for an entry in the family book
	personalParentCategoryId := insertSharedBookTestCategory(t, env.memberUid, models.LevelOneTransactionCategoryParentId, "Personal")
	insertSharedBookTestCategory(t, env.memberUid, personalParentCategoryId, "Dining")

	c := newSharedBookTestContext(http.MethodPost, "/api/v1/llm/transactions/recognize_receipt_image.json", nil, env.memberUid, env.bookId)
	accountNames, accountMap, _, expenseCategoryNames, _, incomeCategoryMap, expenseCategoryMap, transferCategoryMap, _, tagMap, err := LargeLanguageModels.getUserEssentialData(c, c.GetCurrentBookUid())
	assert.Nil(t, err)

	assert.Contains(t, expenseCategoryNames, "Groceries")
	assert.NotContains(t, expenseCategoryNames, "Dining")

	// the member's card shared to the family book is offered under its unique name in the book
	sharedAccountName := ""

	for name, account := range accountMap {
		if account.AccountId == env.usedAccountId {
			sharedAccountName = name
		}
	}

	assert.NotEmpty(t, sharedAccountName)
	assert.Contains(t, accountNames, sharedAccountName)

	response, errResp := LargeLanguageModels.parseRecognizedReceiptImageContent(c, env.bookId, time.UTC, `{"type":"expense","time":"2026-10-18 09:30","amount":"42.10","account":"`+sharedAccountName+`","category":"Groceries","merchant":"Family Market"}`, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)

	assert.Nil(t, errResp)
	assert.Equal(t, env.categoryId, response.CategoryId)
	assert.Equal(t, env.usedAccountId, response.SourceAccountId)
	assert.Equal(t, int64(4210), response.SourceAmount)
	assert.Equal(t, time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC).Unix(), response.Time)
	assert.Equal(t, "Family Market", response.Merchant)

	response, errResp = LargeLanguageModels.parseRecognizedReceiptImageContent(c, env.bookId, time.UTC, `{"type":"expense","amount":"12","category":"Dining"}`, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)

	assert.Nil(t, errResp)
	assert.Equal(t, int64(0), response.CategoryId)
}
