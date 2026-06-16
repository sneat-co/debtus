package cmds4splitusbot

import (
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const ChangeBillPayerCommandCode = "change_bill_payer"

var changeBillPayerCommand = billCallbackCommand(ChangeBillPayerCommandCode,
	func(whc botsfw.WebhookContext, _ dal.ReadwriteTransaction, callbackUrl *url.URL, bill models4splitus.BillEntry) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "changeBillPayerCommand.CallbackAction()")
		var (
			mt string
			//editedMessage *tgbotapi.EditMessageTextConfig
		)
		if mt, err = getBillCardMessageText(ctx, whc.GetBotCode(), whc, bill, true, whc.Translate(trans.MESSAGE_TEXT_BILL_ASK_WHO_PAID)); err != nil {
			return
		}
		if m, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
			return
		}
		markup := tgbotapi.NewInlineKeyboardMarkup()

		for _, member := range bill.Data.GetBillMembers() {
			s := member.Name
			if member.Paid > 0 {
				s = "✔ " + s
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
				{
					Text:         s,
					CallbackData: billCardCallbackCommandData(bill.ID),
				},
			})
		}

		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			{
				Text:         whc.Translate(trans.ButtonTextCancel),
				CallbackData: billCardCallbackCommandData(bill.ID),
			},
		})

		m.Keyboard = markup
		return
	},
)
