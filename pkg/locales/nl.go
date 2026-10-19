package locales

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
)

var nl = &LocaleTextItems{
	GlobalTextItems: &GlobalTextItems{
		AppName: "ezBookkeeping",
	},
	DefaultTypes: &DefaultTypes{
		DecimalSeparator:    core.DECIMAL_SEPARATOR_COMMA,
		DigitGroupingSymbol: core.DIGIT_GROUPING_SYMBOL_DOT,
	},
	DataConverterTextItems: &DataConverterTextItems{
		Alipay:       "Alipay",
		WeChatWallet: "Wallet",
	},
	VerifyEmailTextItems: &VerifyEmailTextItems{
		Title:                     "Verifieer e-mail",
		SalutationFormat:          "Hallo %s,",
		DescriptionAboveBtn:       "Klik op de onderstaande link om je e-mailadres te bevestigen.",
		VerifyEmail:               "Verifieer e-mail",
		DescriptionBelowBtnFormat: "Als je geen %s account hebt aangemaakt, kun je deze e-mail negeren. Als je niet op de bovenstaande link kunt klikken, kopieer dan de URL hierboven en plak deze in je browser. De verificatielink verloopt na  %v minuten.",
	},
	ForgetPasswordMailTextItems: &ForgetPasswordMailTextItems{
		Title:                     "Wachtwoord opnieuw instellen",
		SalutationFormat:          "Hallo %s,",
		DescriptionAboveBtn:       "We hebben onlangs een verzoek ontvangen om je wachtwoord opnieuw in te stellen. Klik op de onderstaande link om je wachtwoord te resetten.",
		ResetPassword:             "Wachtwoord opnieuw instellen",
		DescriptionBelowBtnFormat: "Als je geen verzoek hebt gedaan om je wachtwoord te resetten, kun je deze e-mail negeren. Als je niet op de bovenstaande link kunt klikken, kopieer dan de URL hierboven en plak deze in je browser. De link voor het opnieuw instellen van het wachtwoord verloopt na  %v minuten.",
	},
	BudgetAlertMailTextItems: &BudgetAlertMailTextItems{
		Title:                     "Budget Alert",
		SalutationFormat:          "Hi %s,",
		WarningDescriptionFormat:  "The spending of \"%s\" in %s has reached %d%% of its budget, %s of %s has been spent.",
		ExceededDescriptionFormat: "The spending of \"%s\" in %s has exceeded its budget, %s of %s has been spent.",
		ViewBudgets:               "View Budgets",
		DisableEmailDescription:   "You can turn off budget alert emails in user settings.",
	},
}
