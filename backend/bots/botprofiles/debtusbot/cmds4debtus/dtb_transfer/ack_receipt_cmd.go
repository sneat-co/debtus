package dtb_transfer

import (
	"errors"
	"html"
	"strings"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/errors4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/utmconsts"
	"github.com/sneat-co/sneat-bots/pkg/bots/sneatbots/facade4bots"
	"github.com/sneat-co/sneat-bots/pkg/bots/utm4bots"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/analytics"
	"github.com/strongo/logus"
)

func AcknowledgeReceipt(whc botsfw.WebhookContext, receiptID, operation string) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()

	userCtx := facade4bots.GetUserContext(whc)
	_, transfer, isCounterpartiesJustConnected, err := facade4debtus.AcknowledgeReceipt(ctx, userCtx, receiptID, operation)
	if err != nil {
		if errors.Is(err, errors4debtus.ErrSelfAcknowledgement) {
			m = whc.NewMessage(whc.Translate(trans.MESSAGE_TEXT_SELF_ACKNOWLEDGEMENT, html.EscapeString(transfer.Data.Counterparty().ContactName)))
			return m, nil
		}
		return m, err
	} else {

		{ // Reporting to Google Analytics
			wha := whc.Analytics()
			wha.Enqueue(analytics.NewEvent("receipt-acknowledged", "receipts", "acknowledge-receipt").SetLabel(strings.ToLower(operation)))

			if isCounterpartiesJustConnected {
				wha.Enqueue(analytics.NewEvent("counterparties-connected", "counterparties", "acknowledge-receipt").SetLabel(strings.ToLower(operation)))
			}
		}

		var operationMessage string
		switch operation {
		case dal4debtus.AckAccept:
			operationMessage = whc.Translate(trans.MESSAGE_TEXT_TRANSFER_ACCEPTED_BY_YOU)
		case dal4debtus.AckDecline:
			operationMessage = whc.Translate(trans.MESSAGE_TEXT_TRANSFER_DECLINED_BY_YOU)
		default:
			err = errors.New("Expected accept or decline as operation, got: " + operation)
			return
		}

		utm := utm4bots.NewParams(whc, utmconsts.UTM_CAMPAIGN_RECEIPT)
		if whc.Input().InputType() == botinput.TypeCallbackQuery {
			if m, err = whc.NewEditMessage(common4debtus.TextReceiptForTransfer(ctx, whc, transfer, "", common4debtus.ShowReceiptToCounterparty, utm)+"\n\n"+operationMessage, botmsg.FormatHTML); err != nil {
				return
			}
		} else {
			m = whc.NewMessage(operationMessage + "\n\n" + common4debtus.TextReceiptForTransfer(ctx, whc, transfer, "", common4debtus.ShowReceiptToCounterparty, utm))
			m.Keyboard = dtb_general.MainMenuKeyboardOnReceiptAck(whc)
			m.Format = botmsg.FormatHTML
		}

		if transfer.Data.Creator().TgChatID != 0 {
			askMsgToCreator := whc.NewMessage("")
			askMsgToCreator.ToChat = botmsg.ChatIntID(transfer.Data.Creator().TgChatID)
			var operationMsg string
			counterpartyName := transfer.Data.Counterparty().ContactName
			switch operation {
			case "accept":
				operationMsg = whc.Translate(trans.MESSAGE_TEXT_TRANSFER_ACCEPTED_BY_COUNTERPARTY, html.EscapeString(counterpartyName))
			case "decline":
				operationMsg = whc.Translate(trans.MESSAGE_TEXT_TRANSFER_DECLINED_BY_COUNTERPARTY, html.EscapeString(counterpartyName))
			default:
				err = errors.New("Expected accept or decline as operation, got: " + operation)
			}
			askMsgToCreator.Text = operationMsg + "\n\n" + common4debtus.TextReceiptForTransfer(ctx, whc, transfer, transfer.Data.CreatorUserID, common4debtus.ShowReceiptToAutodetect, utm)

			if transfer.Data.Creator().TgBotID != whc.GetBotCode() {
				logus.Warningf(ctx, "TODO: transferEntity.Creator().TgBotID != whc.GetBotCode(): "+askMsgToCreator.Text)
			} else {
				if _, err = whc.Responder().SendMessage(ctx, askMsgToCreator, botsfw.BotAPISendMessageOverHTTPS); err != nil {
					logus.Errorf(ctx, "Failed to send acknowledge to creator: %v", err)
					err = nil // This is not that critical to report the error to user
				}
			}
		}
		// Seems we can edit message just once after callback :(
		//if transferEntity.CounterpartyTgReceiptInlineMessageID != "" {
		//	mt = anybot.TextReceiptForTransfer(whc, transferID, transferEntity, transferEntity.CounterpartyContactID)
		//	editMessage := tgbotapi.NewEditMessageTextByInlineMessageID(transferEntity.CounterpartyTgReceiptInlineMessageID, mt + fmt.Sprintf("\n\n Acknowledged by %v", transferEntity.DebtusSpaceContactEntry().ContactName))
		//
		//	if values, err := editMessage.Values(); err != nil {
		//		logus.Errorf(ctx, "Failed to get values for editMessage: %v", err)
		//	} else {
		//		logus.Debugf(ctx, "editMessage.Values(): %v", values)
		//	}
		//	updateMessage := whc.NewMessage("")
		//	updateMessage.TelegramEditMessageText = &editMessage
		//	_, err := whc.Responder().SendMessage(ctx, updateMessage, botsfw.BotAPISendMessageOverHTTPS)
		//	if err != nil {
		//		logus.Errorf(ctx, "Failed to update counterparty receipt message: %v", err)
		//	}
		//}
		return m, err
	}
}
