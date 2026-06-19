package cmds4splitusbot

import (
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

var billSplitModesListCommand = billCallbackCommand("split-modes",
	func(whc botsfw.WebhookContext, _ dal.ReadwriteTransaction, callbackUrl *url.URL, bill models4splitus.BillEntry) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "billSplitModesListCommand.CallbackAction()")
		var mt string
		if mt, err = getBillCardMessageText(ctx, whc.GetBotCode(), whc, bill, true, ""); err != nil {
			return
		}
		if m, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
			return
		}
		callbackData := fmt.Sprintf("split-mode?bill=%v&mode=", bill.ID)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         whc.Translate(trans.SPLIT_MODE_EQUALLY),
					CallbackData: callbackData + string(models4splitus.SplitModeEqually),
				},
			},
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         whc.Translate(trans.SPLIT_MODE_PERCENTAGE),
					CallbackData: callbackData + string(models4splitus.SplitModePercentage),
				},
			},
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         whc.Translate(trans.SPLIT_MODE_SHARES),
					CallbackData: callbackData + string(models4splitus.SplitModeShare),
				},
			},
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         whc.Translate(trans.SPLIT_MODE_EXACT_AMOUNT),
					CallbackData: callbackData + string(models4splitus.SplitModeExactAmount),
				},
			},
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         whc.Translate(trans.ButtonTextCancel),
					CallbackData: billCardCallbackCommandData(bill.ID),
				},
			},
		)
		m.Keyboard = keyboard
		return
	},
)
