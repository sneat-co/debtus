package cmds4splitusbot

import (
	"context"
	"fmt"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/const4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/strongo/logus"
)

const outstandingBalanceCommandCode = "outstanding_balance"

var outstandingBalanceCommand = botsfw.Command{
	Code:       outstandingBalanceCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/outstanding"},
	Action:     outstandingBalanceAction,
}

func outstandingBalanceAction(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "outstandingBalanceAction()")
	userID := whc.AppUserID()
	err = dal4userus.RunUserExtWorker[models4splitus.SplitusUserDbo](
		ctx, userID, const4splitus.ModuleID, new(models4splitus.SplitusUserDbo),
		func(ctx context.Context, tx dal.ReadwriteTransaction, param *dal4userus.UserExtWorkerParams[models4splitus.SplitusUserDbo]) (err error) {
			outstandingBalance := param.UserExt.Data.GetOutstandingBalance()
			m.Text = fmt.Sprintf("Outstanding balance: %v", outstandingBalance)
			return err
		})
	if err != nil {
		return m, err
	}
	return
}
