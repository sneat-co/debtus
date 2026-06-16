package cmds4splitusbot

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/strongo/logus"
)

const closeBillCommandCode = "close_bill"

var closeBillCommand = billCallbackCommand(closeBillCommandCode,
	func(whc botsfw.WebhookContext, _ dal.ReadwriteTransaction, callbackUrl *url.URL, bill models4splitus.BillEntry) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "closeBillCommand.CallbackAction()")
		return ShowBillCard(whc, true, bill, "Sorry, not implemented yet.")
	},
)
