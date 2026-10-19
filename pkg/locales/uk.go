package locales

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
)

var uk = &LocaleTextItems{
	GlobalTextItems: &GlobalTextItems{
		AppName: "ezBookkeeping",
	},
	DefaultTypes: &DefaultTypes{
		DecimalSeparator:    core.DECIMAL_SEPARATOR_COMMA,
		DigitGroupingSymbol: core.DIGIT_GROUPING_SYMBOL_SPACE,
	},
	DataConverterTextItems: &DataConverterTextItems{
		Alipay:       "Alipay",
		WeChatWallet: "Wallet",
	},
	VerifyEmailTextItems: &VerifyEmailTextItems{
		Title:                     "Підтвердіть електронну пошту",
		SalutationFormat:          "Вітаємо, %s!",
		DescriptionAboveBtn:       "Натисніть на посилання нижче, щоб підтвердити вашу електронну адресу.",
		VerifyEmail:               "Підтвердити електронну пошту",
		DescriptionBelowBtnFormat: "Якщо ви не створювали обліковий запис %s, просто проігноруйте цей лист. Якщо ви не можете натиснути на посилання вище, скопіюйте вказану URL-адресу та вставте її у свій браузер. Посилання для підтвердження електронної пошти буде дійсне протягом %v хвилин.",
	},
	ForgetPasswordMailTextItems: &ForgetPasswordMailTextItems{
		Title:                     "Скидання пароля",
		SalutationFormat:          "Вітаємо, %s!",
		DescriptionAboveBtn:       "Нещодавно ми отримали запит на скидання вашого пароля. Натисніть на посилання нижче, щоб скинути пароль.",
		ResetPassword:             "Скинути пароль",
		DescriptionBelowBtnFormat: "Якщо ви не надсилали запит на скидання пароля, просто проігноруйте цей лист. Якщо ви не можете натиснути на посилання вище, скопіюйте вказану URL-адресу та вставте її у свій браузер. Посилання для скидання пароля буде дійсне протягом %v хвилин.",
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
