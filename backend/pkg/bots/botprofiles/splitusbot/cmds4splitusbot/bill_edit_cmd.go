package cmds4splitusbot

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
	"github.com/strongo/logus"
)

const editBillCommandCode = "edit_bill"

var editBillCommand = billCallbackCommand(editBillCommandCode,
	func(whc botsfw.WebhookContext, _ dal.ReadwriteTransaction, callbackUrl *url.URL, bill models4splitus.BillEntry) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "editBillCommand.CallbackAction()")
		var mt string

		if mt, err = getBillCardMessageText(ctx, whc.GetBotCode(), whc, bill, true, ""); err != nil {
			return
		}
		if m, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
			return
		}
		m.Keyboard = getPrivateBillCardInlineKeyboard(whc, whc.GetBotCode(), bill)
		return
	},
)
