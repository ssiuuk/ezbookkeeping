package _default

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestDefaultTransactionDataJsonFileImporterParseImportedData_ParseMerchant(t *testing.T) {
	importer := DefaultTransactionDataJsonFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "CNY",
	}

	allNewTransactions, _, _, _, _, _, err := importer.ParseImportedData(context, user, []byte(`{"transactions":[`+
		`{"time":"2024-09-01 12:34:56","utcOffset":"480","type":"Expense","categoryName":"Test Category","sourceAccountName":"Test Account","sourceAmount":"123.45","comment":"Coffee","merchant":"Test Shop"},`+
		`{"time":"2024-09-01 12:34:57","utcOffset":"480","type":"Expense","categoryName":"Test Category","sourceAccountName":"Test Account","sourceAmount":"1.23","comment":"Tea"}`+
		`]}`), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(allNewTransactions))
	assert.Equal(t, "Coffee", allNewTransactions[0].Comment)
	assert.Equal(t, "Test Shop", allNewTransactions[0].Merchant)
	assert.Equal(t, "Tea", allNewTransactions[1].Comment)
	assert.Equal(t, "", allNewTransactions[1].Merchant)
}
