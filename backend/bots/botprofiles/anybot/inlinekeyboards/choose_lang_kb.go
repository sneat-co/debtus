package inlinekeyboards

import (
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func GetChooseLangInlineKeyboard(format string, currentLocaleCode5 string) (kbRows [][]tgbotapi.InlineKeyboardButton) {
	kbRows = make([][]tgbotapi.InlineKeyboardButton, 0, len(trans.SupportedLocales))

	for _, code5 := range trans.SupportedLocales {
		if code5 != currentLocaleCode5 {
			locale := i18n.GetLocaleByCode5(code5)
			btnRow := []tgbotapi.InlineKeyboardButton{
				{
					Text:         locale.TitleWithIcon(),
					CallbackData: fmt.Sprintf(format, locale.Code5),
				},
			}
			kbRows = append(kbRows, btnRow)
		}
	}

	return
}
