package cmds4splitusbot

import (
	"context"
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/strongo/logus"

	"errors"

	"github.com/strongo/decimal"
)

const (
	NEW_BILL_PARAM_I      = "i"
	NEW_BILL_PARAM_V      = "v"
	NEW_BILL_PARAM_C      = "c"
	NEW_BILL_PARAM_I_OWE  = "owe"
	NEW_BILL_PARAM_I_PAID = "paid"
)

const newBillCommandCode = "new_bill"

var newBillCommand = botsfw.Command{
	Code:       newBillCommandCode,
	InputTypes: []botinput.Type{botinput.TypeCallbackQuery},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "newBillCommand.CallbackAction(callbackUrl=%v)", callbackUrl)
		query := callbackUrl.Query()
		paramI := query.Get(NEW_BILL_PARAM_I)
		if paramI != NEW_BILL_PARAM_I_OWE && paramI != NEW_BILL_PARAM_I_PAID {
			err = errors.New("paramI != NEW_BILL_PARAM_I_OWE && paramI != NEW_BILL_PARAM_I_PAID")
			return
		}
		var amountValue, paidAmount decimal.Decimal64p2
		if amountValue, err = decimal.ParseDecimal64p2(query.Get(NEW_BILL_PARAM_V)); err != nil {
			return
		}
		if paramI == NEW_BILL_PARAM_I_PAID {
			paidAmount = amountValue
		}

		strUserID := whc.AppUserID()

		billEntity := models4splitus.NewBillEntity(
			models4splitus.BillCommon{
				Status:        models4splitus.BillStatusDraft,
				SplitMode:     models4splitus.SplitModeEqually,
				CreatorUserID: strUserID,
				AmountTotal:   amountValue,
				Currency:      money.CurrencyCode(query.Get("ctx")),
				UserIDs:       []string{strUserID},
			},
		)
		//tgMessage := whc.Input().(telegram.TelegramWebhookInput).
		//callbackQuery :=
		tgChatMessageID := fmt.Sprintf("%v@%v@%v", whc.Input().(telegram.WebhookCallbackQuery).GetInlineMessageID(), whc.GetBotCode(), whc.Locale().Code5)
		billEntity.TgChatMessageIDs = []string{tgChatMessageID}

		var appUser botsfwmodels.AppUserData
		if appUser, err = whc.AppUserData(); err != nil {
			return
		}
		userData := appUser.(*dbo4userus.UserDbo)
		userName := userData.Names.GetFullName()
		if userName == "" {
			err = errors.New("userData has no name")
			return
		}

		spaceID := userData.GetFamilySpaceID()

		billMember := briefs4splitus.BillMemberBrief{
			Paid: paidAmount,
		}

		//appUserID := whc.AppUserID()

		if err = billEntity.SetBillMembers([]*briefs4splitus.BillMemberBrief{&billMember}); err != nil {
			return
		}

		return m, facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			var bill models4splitus.BillEntry
			if bill, err = facade4splitus.CreateBill(ctx, tx, spaceID, billEntity); err != nil {
				return
			}
			m, err = ShowBillCard(whc, true, bill, "")
			return err
		})
	},
}
