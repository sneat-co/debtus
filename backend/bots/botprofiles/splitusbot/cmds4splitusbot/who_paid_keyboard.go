package cmds4splitusbot

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func getWhoPaidInlineKeyboard(translator i18n.SingleLocaleTranslator, billID string) *tgbotapi.InlineKeyboardMarkup {
	callbackDataPrefix := billCallbackCommandData(joinBillCommandCode, billID)
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "✋ " + translator.Translate(trans.BUTTON_TEXT_I_PAID_FOR_THE_BILL), CallbackData: callbackDataPrefix + "&i=paid"}},
			{{Text: "🙏 " + translator.Translate(trans.BUTTON_TEXT_I_OWE_FOR_THE_BILL), CallbackData: callbackDataPrefix + "&i=owe"}},
			{{Text: "🚫 " + translator.Translate(trans.BUTTON_TEXT_I_DO_NOT_SHARE_THIS_BILL), CallbackData: billCallbackCommandData(leaveBillCommandCode, billID)}},
		},
	}
}
