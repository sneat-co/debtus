package dtb_transfer

import (
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

const cancelTransferWizardCommandCode = "cancel_transfer_wizard"

var cancelTransferWizardCommand = botsfw.Command{
	Code:       cancelTransferWizardCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   trans.Commands(trans.COMMAND_TEXT_CANCEL, "/cancel", emoji.NO_ENTRY_SIGN_ICON),
	Action:     cancelTransferWizardCommandAction,
}

func cancelTransferWizardCommandAction(whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
	whc.ChatData().SetAwaitingReplyTo("")
	//var m botmsg.MessageFromBot
	//userKey, _, err := whc.GetUser()
	//if err != nil {
	//	return m, err
	//}
	//var api4transfers []models.Transfer
	//ctx := whc.Context()
	//transferKeys, err := datastore.NewQuery(models.TransferKind).Filter("UserID =", userKey.IntID()).Limit(1).GetAll(ctx, &api4transfers)
	//if err != nil {
	//	return m, err
	//}
	m := whc.NewMessageByCode(trans.MESSAGE_TEXT_TRANSFER_CREATION_CANCELED)
	//if len(transferKeys) == 0 {
	//	m = tgbotapi.NewMessage(whc.ChatID(), Translate(trans.MESSAGE_TEXT_NOTHING_TO_CANCEL, whc))
	//} else {
	//	err := datastore.Delete(ctx, transferKeys[0])
	//	if err != nil {
	//		return m, err
	//	}
	//	//transfer := api4transfers[0]
	//}
	if err := dtb_general.SetMainMenuKeyboard(whc, &m); err != nil {
		return m, err
	}
	return m, nil
}
