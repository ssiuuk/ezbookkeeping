package locales

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
)

var pl = &LocaleTextItems{
	GlobalTextItems: &GlobalTextItems{
		AppName: "ezBookkeeping",
	},
	DefaultTypes: &DefaultTypes{
		DecimalSeparator:    core.DECIMAL_SEPARATOR_COMMA,
		DigitGroupingSymbol: core.DIGIT_GROUPING_SYMBOL_SPACE,
	},
	DataConverterTextItems: &DataConverterTextItems{
		Alipay:       "Alipay",
		WeChatWallet: "Portfel",
	},
	VerifyEmailTextItems: &VerifyEmailTextItems{
		Title:                     "Potwierdź adres e-mail",
		SalutationFormat:          "Cześć %s,",
		DescriptionAboveBtn:       "Kliknij poniższy odnośnik, aby potwierdzić swój adres e-mail.",
		VerifyEmail:               "Potwierdź adres e-mail",
		DescriptionBelowBtnFormat: "Jeżeli nie zakładałeś konta %s, zignoruj tę wiadomość. Jeżeli nie możesz kliknąć powyższego odnośnika, skopiuj jego adres i wklej go do przeglądarki. Odnośnik do potwierdzenia adresu e-mail wygaśnie po %v minutach.",
	},
	ForgetPasswordMailTextItems: &ForgetPasswordMailTextItems{
		Title:                     "Zresetuj hasło",
		SalutationFormat:          "Cześć %s,",
		DescriptionAboveBtn:       "Otrzymaliśmy prośbę o zresetowanie Twojego hasła.\nMożesz kliknąć poniższy odnośnik, aby ustawić nowe hasło.",
		ResetPassword:             "Zresetuj hasło",
		DescriptionBelowBtnFormat: "Jeżeli nie prosiłeś o zresetowanie hasła, zignoruj tę wiadomość. Jeżeli nie możesz kliknąć powyższego odnośnika, skopiuj jego adres i wklej go do przeglądarki. Odnośnik do resetowania hasła wygaśnie po %v minutach.",
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
