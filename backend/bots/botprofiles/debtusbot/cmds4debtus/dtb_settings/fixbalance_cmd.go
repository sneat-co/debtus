package dtb_settings

import (
	"context"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
)

var getUser = dal4userus.GetUser

var runReadwriteTransaction = facade.RunReadwriteTransaction

var fixBalanceCommand = botsfw.Command{
	Code:       "fix_balance",
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/fix_balance"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		user := dbo4userus.NewUserEntry(whc.AppUserID())
		if err = getUser(ctx, nil, user); err != nil {
			return
		}
		spaceID := user.Data.GetFamilySpaceID()
		if err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			debtusSpace := models4debtus.NewDebtusSpaceEntry(spaceID)
			if err != nil {
				return err
			}
			debtusSpace.Data.Balance = make(money.Balance, len(debtusSpace.Data.Balance))
			for _, contact := range debtusSpace.Data.Contacts {
				for k, v := range contact.Balance {
					debtusSpace.Data.Balance[k] += v
				}
			}
			return tx.Set(ctx, debtusSpace.Record)
		}); err != nil {
			return
		}
		m = whc.NewMessage("Balance fixed")
		return
	},
}
