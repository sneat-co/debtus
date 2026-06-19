package dtb_transfer

import (
	"strings"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/email4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const askEmailForReceiptCommandCode = "ask_email4receipt"

var askEmailForReceiptCommand = botsfw.Command{
	Code:       askEmailForReceiptCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()

		logus.Debugf(ctx, "askEmailForReceiptCommand.Action()")
		email := whc.Input().(botinput.TextMessage).Text()
		if !strings.Contains(email, "@") {
			return whc.NewMessage(whc.Translate(trans.MESSAGE_TEXT_INVALID_EMAIL)), nil
		}

		chatEntity := whc.ChatData()
		transferID := chatEntity.GetWizardParam(WizardParamTransfer)
		transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, nil, transferID)
		if err != nil {
			return m, err
		}
		m, err = sendReceiptByEmail(whc, email, "", transfer)
		return
	},
}

func sendReceiptByEmail(whc botsfw.WebhookContext, toEmail, toName string, transfer models4debtus.TransferEntry) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	receiptEntity := models4debtus.NewReceiptEntity(whc.AppUserID(), transfer.ID, transfer.Data.Counterparty().UserID, whc.Locale().Code5, string(models4debtus.InviteByEmail), toEmail, general4debtus.CreatedOn{
		CreatedOnPlatform: whc.BotPlatform().ID(),
		CreatedOnID:       whc.GetBotCode(),
	})
	var receipt models4debtus.ReceiptEntry
	if receipt, err = dal4debtus.Default.Receipt.CreateReceipt(ctx, receiptEntity); err != nil {
		return m, err
	}

	emailID := ""
	if emailID, err = email4debtus.SendReceiptByEmail(
		ctx,
		whc,
		receipt,
		whc.Input().GetSender().GetFirstName(),
		toName,
		toEmail,
	); err != nil {
		return m, err
	}

	m = whc.NewMessageByCode(trans.MESSAGE_TEXT_RECEIPT_SENT_THROW_EMAIL, emailID)

	return m, err
}
