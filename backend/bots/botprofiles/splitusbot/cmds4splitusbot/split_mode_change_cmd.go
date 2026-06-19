package cmds4splitusbot

import (
	"context"
	"net/url"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/logus"
)

var billChangeSplitModeCommand = botsfw.Command{
	Code:       "split-mode",
	InputTypes: []botinput.Type{botinput.TypeCallbackQuery},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "billChangeSplitModeCommand.CallbackAction()")
		var bill models4splitus.BillEntry
		if bill.ID, err = GetBillID(callbackUrl); err != nil {
			return
		}

		var db dal.DB
		if db, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
		if bill, err = facade4splitus.GetBillByID(ctx, db, bill.ID); err != nil {
			return
		}
		splitMode := models4splitus.SplitMode(callbackUrl.Query().Get("mode"))
		if bill.Data.SplitMode != splitMode {
			bill.Data.SplitMode = splitMode
			if err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
				if err = facade4splitus.SaveBill(ctx, tx, bill); err != nil {
					return
				}
				return
			}); err != nil {
				return
			}
		}
		return ShowBillCard(whc, true, bill, "")
	},
}
