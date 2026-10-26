package andromoney

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/converters/csv"
	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

const andromoneyDataHeaderFirstColumnName = "Id"

// the columns of the AndroMoney export, in the order AndroMoney writes them, the header names depend on the app language
const (
	andromoneyColumnIndexId = iota
	andromoneyColumnIndexCurrency
	andromoneyColumnIndexAmount
	andromoneyColumnIndexCategory
	andromoneyColumnIndexSubCategory
	andromoneyColumnIndexDate
	andromoneyColumnIndexFromAccount
	andromoneyColumnIndexToAccount
	andromoneyColumnIndexNote
	andromoneyColumnIndexPeriodic
	andromoneyColumnIndexProject
	andromoneyColumnIndexMerchant
	andromoneyColumnIndexUid
	andromoneyColumnIndexTime
)

const andromoneyMinimumColumnCount = andromoneyColumnIndexUid + 1

// these columns are not in the file, they are added after the AndroMoney columns
const (
	andromoneyTypeColumnName              = "Type"
	andromoneyToAccountCurrencyColumnName = "To Account Currency"
	andromoneyToAmountColumnName          = "To Amount"
)

var andromoneyAddedColumnNames = []string{
	andromoneyTypeColumnName,
	andromoneyToAccountCurrencyColumnName,
	andromoneyToAmountColumnName,
}

// the header names are replaced with these, so that the columns can be found whatever language the file was exported in
var andromoneyColumnNames = []string{
	"Id",
	"Currency",
	"Amount",
	"Category",
	"Sub Category",
	"Date",
	"From Account",
	"To Account",
	"Note",
	"Periodic",
	"Project",
	"Merchant",
	"uid",
	"Time",
}

var andromoneyTransactionDataColumnNameMapping = map[datatable.TransactionDataTableColumn]string{
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME:         andromoneyColumnNames[andromoneyColumnIndexDate],
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE:         andromoneyTypeColumnName,
	datatable.TRANSACTION_DATA_TABLE_CATEGORY:                 andromoneyColumnNames[andromoneyColumnIndexCategory],
	datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY:             andromoneyColumnNames[andromoneyColumnIndexSubCategory],
	datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME:             andromoneyColumnNames[andromoneyColumnIndexFromAccount],
	datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY:         andromoneyColumnNames[andromoneyColumnIndexCurrency],
	datatable.TRANSACTION_DATA_TABLE_AMOUNT:                   andromoneyColumnNames[andromoneyColumnIndexAmount],
	datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME:     andromoneyColumnNames[andromoneyColumnIndexToAccount],
	datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_CURRENCY: andromoneyToAccountCurrencyColumnName,
	datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT:           andromoneyToAmountColumnName,
	datatable.TRANSACTION_DATA_TABLE_DESCRIPTION:              andromoneyColumnNames[andromoneyColumnIndexNote],
	datatable.TRANSACTION_DATA_TABLE_PROJECT:                  andromoneyColumnNames[andromoneyColumnIndexProject],
	datatable.TRANSACTION_DATA_TABLE_MERCHANT:                 andromoneyColumnNames[andromoneyColumnIndexMerchant],
}

var andromoneyTransactionTypeNameMapping = map[models.TransactionType]string{
	models.TRANSACTION_TYPE_INCOME:   "Income",
	models.TRANSACTION_TYPE_EXPENSE:  "Expense",
	models.TRANSACTION_TYPE_TRANSFER: "Transfer",
}

// andromoneyTransactionDataCsvFileImporter defines the structure of AndroMoney csv importer for transaction data
type andromoneyTransactionDataCsvFileImporter struct{}

// Initialize a AndroMoney transaction data csv file importer singleton instance
var (
	AndroMoneyTransactionDataCsvFileImporter = &andromoneyTransactionDataCsvFileImporter{}
)

// ParseImportedData returns the imported data by parsing the AndroMoney transaction csv data
func (c *andromoneyTransactionDataCsvFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	csvReader := csv.CreateNewCsvReaderFromData(data, getAndroMoneyFileEncoding(data).NewDecoder(), ',', true)
	allLines, err := csvReader.ReadAllLines(ctx)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// the file begins with a line naming the app and the export date, the data begins at the header line
	headerLineIndex := -1

	for i, line := range allLines {
		if len(line.Items) >= andromoneyMinimumColumnCount && line.Items[andromoneyColumnIndexId] == andromoneyDataHeaderFirstColumnName {
			headerLineIndex = i
			break
		}
	}

	if headerLineIndex < 0 {
		log.Errorf(ctx, "[andromoney_transaction_data_csv_file_importer.ParseImportedData] cannot find the header line")
		return nil, nil, nil, nil, nil, nil, errs.ErrInvalidCSVFile
	}

	if len(allLines) < headerLineIndex+2 {
		log.Errorf(ctx, "[andromoney_transaction_data_csv_file_importer.ParseImportedData] cannot parse import data, because data table row count is less 1")
		return nil, nil, nil, nil, nil, nil, errs.ErrNotFoundTransactionDataInFile
	}

	dataLines := make([]*datatable.PlainTextDataTableLineData, 0, len(allLines)-headerLineIndex)
	dataLines = append(dataLines, &datatable.PlainTextDataTableLineData{
		LineNumber: allLines[headerLineIndex].LineNumber,
		Items:      append(append([]string{}, andromoneyColumnNames...), andromoneyAddedColumnNames...),
	})

	accountCurrencies := getAndroMoneyAccountCurrencies(allLines[headerLineIndex+1:])

	for _, line := range allLines[headerLineIndex+1:] {
		items := convertAndroMoneyDataLine(line.Items, accountCurrencies)
		toAccountCurrency := items[len(andromoneyColumnNames)+1]

		// the file only has the amount paid from, so the amount received in another currency is unknown,
		// the transfer is refused instead of importing a wrong amount
		if toAccountCurrency != "" && toAccountCurrency != items[andromoneyColumnIndexCurrency] {
			log.Errorf(ctx, "[andromoney_transaction_data_csv_file_importer.ParseImportedData] cannot import transfer from \"%s\" (%s) to \"%s\" (%s) in line %d, because the received amount is unknown", items[andromoneyColumnIndexFromAccount], items[andromoneyColumnIndexCurrency], items[andromoneyColumnIndexToAccount], toAccountCurrency, line.LineNumber)
			return nil, nil, nil, nil, nil, nil, errs.NewImportError(errs.ErrAccountCurrencyInvalid, fmt.Sprintf("line:%d", line.LineNumber), items[andromoneyColumnIndexToAccount]+" ("+toAccountCurrency+")")
		}

		dataLines = append(dataLines, &datatable.PlainTextDataTableLineData{
			LineNumber: line.LineNumber,
			Items:      items,
		})
	}

	dataTable := datatable.CreateNewPlainTextDataTableWithLineNumbers(dataLines, true)
	transactionDataTable := datatable.CreateNewTransactionDataTableFromBasicDataTable(dataTable, andromoneyTransactionDataColumnNameMapping)
	dataTableImporter := converter.CreateNewImporterWithTypeNameMapping(andromoneyTransactionTypeNameMapping, "", "", ",")

	return dataTableImporter.ParseImportedData(ctx, user, transactionDataTable, defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}

// getAndroMoneyFileEncoding returns the encoding of the file, AndroMoney writes Big5 when the app is in traditional Chinese
func getAndroMoneyFileEncoding(data []byte) encoding.Encoding {
	if utf8.Valid(data) {
		return unicode.UTF8BOM
	}

	return traditionalchinese.Big5
}

// getAndroMoneyAccountCurrencies returns the currency of each account paid from or paid into in the file,
// the currency of a row is the currency of the account paid from, or of the account paid into for an income
func getAndroMoneyAccountCurrencies(lines []*datatable.PlainTextDataTableLineData) map[string]string {
	accountCurrencies := make(map[string]string)

	for _, line := range lines {
		if len(line.Items) <= andromoneyColumnIndexToAccount || line.Items[andromoneyColumnIndexCurrency] == "" {
			continue
		}

		fromAccount := line.Items[andromoneyColumnIndexFromAccount]
		toAccount := line.Items[andromoneyColumnIndexToAccount]

		if fromAccount != "" {
			accountCurrencies[fromAccount] = line.Items[andromoneyColumnIndexCurrency]
		} else if toAccount != "" {
			accountCurrencies[toAccount] = line.Items[andromoneyColumnIndexCurrency]
		}
	}

	return accountCurrencies
}

// convertAndroMoneyDataLine returns the items of a data line with the date and time joined, and the transaction type and the account paid into of a transfer added
func convertAndroMoneyDataLine(originalItems []string, accountCurrencies map[string]string) []string {
	items := make([]string, len(andromoneyColumnNames)+len(andromoneyAddedColumnNames))
	copy(items, originalItems)

	// expenses only have the account paid from, incomes only the account paid into, and transfers have both
	fromAccount := items[andromoneyColumnIndexFromAccount]
	toAccount := items[andromoneyColumnIndexToAccount]
	transactionType := ""

	if fromAccount != "" && toAccount != "" {
		transactionType = andromoneyTransactionTypeNameMapping[models.TRANSACTION_TYPE_TRANSFER]
	} else if fromAccount != "" {
		transactionType = andromoneyTransactionTypeNameMapping[models.TRANSACTION_TYPE_EXPENSE]
	} else if toAccount != "" {
		transactionType = andromoneyTransactionTypeNameMapping[models.TRANSACTION_TYPE_INCOME]
		items[andromoneyColumnIndexFromAccount] = toAccount
		items[andromoneyColumnIndexToAccount] = ""
	}

	items[len(andromoneyColumnNames)] = transactionType

	if fromAccount != "" && toAccount != "" {
		// the account paid into has the currency it is used with elsewhere in the file, otherwise the currency of the transfer
		toAccountCurrency := accountCurrencies[toAccount]

		if toAccountCurrency == "" {
			toAccountCurrency = items[andromoneyColumnIndexCurrency]
		}

		items[len(andromoneyColumnNames)+1] = toAccountCurrency
		items[len(andromoneyColumnNames)+2] = items[andromoneyColumnIndexAmount]
	}
	items[andromoneyColumnIndexDate] = getAndroMoneyDateTime(items[andromoneyColumnIndexDate], items[andromoneyColumnIndexTime])

	return items
}

// getAndroMoneyDateTime returns the long date time of the date (yyyyMMdd) and time (HHmm without leading zeros, may be empty), or the date as is when it cannot be read
func getAndroMoneyDateTime(date string, hourMinute string) string {
	if len(date) != 8 || !isAndroMoneyDigits(date) {
		return date
	}

	if len(hourMinute) > 4 || !isAndroMoneyDigits(hourMinute) {
		hourMinute = ""
	}

	hourMinute = fmt.Sprintf("%04s", hourMinute)
	hourMinute = strings.ReplaceAll(hourMinute, " ", "0")

	return fmt.Sprintf("%s-%s-%s %s:%s:00", date[0:4], date[4:6], date[6:8], hourMinute[0:2], hourMinute[2:4])
}

func isAndroMoneyDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}
