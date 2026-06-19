package dtb_transfer

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

var sendReceiptCallbackCommand = botsfw.NewCallbackCommand(SendReceiptCallbackPath, sendReceiptCallbackAction)

func sendReceiptCallbackAction(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	q := callbackUrl.Query()
	sendBy := q.Get("by")
	spaceID := coretypes.SpaceID(q.Get("spaceID"))
	logus.Debugf(ctx, "sendReceiptCallbackAction(callbackUrl=%v)", callbackUrl)
	return m, facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		var (
			transferID string
			transfer   models4debtus.TransferEntry
		)
		transferID = q.Get(WizardParamTransfer)
		if transferID == "" {
			return fmt.Errorf("missing transfer ContactID")
		}
		transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
		if err != nil {
			return fmt.Errorf("failed to get transfer by ContactID: %w", err)
		}
		//chatEntity := whc.ChatData() //TODO: Need this to get appUser, has to be refactored
		//appUser, err := whc.GetAppUser()
		counterparty, err := facade4debtus.GetDebtusSpaceContactByID(ctx, tx, spaceID, transfer.Data.Counterparty().ContactID)
		if err != nil {
			return err
		}
		if IsTransferNotificationsBlockedForChannel(counterparty.Data, sendBy) {
			m = whc.NewMessage(trans.MESSAGE_TEXT_USER_BLOCKED_TRANSFER_NOTIFICATIONS_BY)
			return err
		}
		chatEntity := whc.ChatData()
		switch sendBy {
		case SendReceiptByChooseChannel:
			m, err = createSendReceiptOptionsMessage(whc, transfer)
			return
		case ReceiptActionDoNotSend:
			logus.Debugf(ctx, "sendReceiptCallbackAction(): do-not-send")
			if m, err = whc.NewEditMessage(whc.Translate(trans.MESSAGE_TEXT_RECEIPT_WILL_NOT_BE_SENT), botmsg.FormatHTML); err != nil {
				return
			}

			// TODO: do type assertion with botsfw.CallbackQuery interface
			callbackMessage := whc.Input().(telegram.WebhookCallbackQuery).GetMessage().(botinput.TextMessage)
			if callbackMessage != nil && callbackMessage.Text() == m.Text {
				m.Text += " (double clicked)"
			}
			m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					{
						Text:         whc.Translate(trans.COMMAND_TEXT_I_HAVE_CHANGED_MY_MIND),
						CallbackData: fmt.Sprintf("%v?by=%v&%v=%v", SendReceiptCallbackPath, SendReceiptByChooseChannel, WizardParamTransfer, transferID),
					},
				},
			)
			return err
		case string(models4debtus.InviteByTelegram):
			err = fmt.Errorf("unsupported send-receipt option: %v", models4debtus.InviteByTelegram)
			logus.Errorf(ctx, err.Error())
			return err
		case string(models4debtus.InviteByLinkToTelegram):
			m, err = showLinkForReceiptInTelegram(whc, transfer)
			return err
		case string(models4debtus.InviteBySms):

			if counterparty.Data.PhoneNumber > 0 {
				m, err = sendReceiptBySms(whc, tx, spaceID, counterparty.Data.PhoneContact, transfer, counterparty)
				return err
			} else {
				var updateMessage botmsg.MessageFromBot
				if updateMessage, err = whc.NewEditMessage(whc.Translate(trans.MESSAGE_TEXT_LETS_SEND_SMS), botmsg.FormatHTML); err != nil {
					return
				}
				if _, err = whc.Responder().SendMessage(ctx, updateMessage, botsfw.BotAPISendMessageOverHTTPS); err != nil {
					logus.Errorf(ctx, fmt.Errorf("failed to update Telegram message: %w", err).Error())
					err = nil
				}

				chatEntity.SetAwaitingReplyTo(askPhoneNumberForReceiptCommandCode)
				chatEntity.AddWizardParam(WizardParamTransfer, transferID)
				mt := strings.Join([]string{
					whc.Translate(trans.MESSAGE_TEXT_ASK_PHONE_NUMBER_OF_COUNTERPARTY, html.EscapeString(transfer.Data.Counterparty().ContactName)),
					whc.Translate(trans.MESSAGE_TEXT_USE_CONTACT_TO_SEND_PHONE_NUMBER, emoji.PAPERCLIP_ICON),
					whc.Translate(trans.MESSAGE_TEXT_ABOUT_PHONE_NUMBER_FORMAT),
					whc.Translate(trans.MESSAGE_TEXT_THIS_NUMBER_WILL_BE_USED_TO_SEND_RECEIPT),
				}, "\n\n")
				//mt += "\n\n" + whc.Translate(trans.MESSAGE_TEXT_VIEW_MY_NUMBER_IN_INTERNATIONAL_FORMAT)

				m = whc.NewMessage(mt)
				m.Format = botmsg.FormatHTML
				keyboard := [][]tgbotapi.KeyboardButton{
					{
						{RequestContact: true, Text: whc.Translate(trans.COMMAND_TEXT_VIEW_MY_NUMBER_IN_INTERNATIONAL_FORMAT)},
					},
				}
				lastName := whc.Input().GetSender().GetLastName()
				if lastName == "Trakhimenok" || lastName == "Paltseva" {
					for k := range common4debtus.TwilioTestNumbers {
						keyboard = append(keyboard, []tgbotapi.KeyboardButton{{Text: k}})

					}
				}
				m.Keyboard = &tgbotapi.ReplyKeyboardMarkup{
					Keyboard: keyboard,
				}
			}
		case string(models4debtus.InviteByEmail):
			chatEntity.SetAwaitingReplyTo(askEmailForReceiptCommandCode)
			chatEntity.AddWizardParam(WizardParamTransfer, transferID)
			m = whc.NewMessage(whc.Translate(trans.MESSAGE_TEXT_INVITE_ASK_EMAIL_FOR_RECEIPT, transfer.Data.Counterparty().ContactName))
		default:
			err = errors.New("Unknown channel to send receipt: " + sendBy)
			logus.Errorf(ctx, err.Error())
		}
		return err
	})
}

func showLinkForReceiptInTelegram(whc botsfw.WebhookContext, transfer models4debtus.TransferEntry) (m botmsg.MessageFromBot, err error) {
	receiptData := models4debtus.NewReceiptEntity(whc.AppUserID(), transfer.ID, transfer.Data.Counterparty().UserID, whc.Locale().Code5, "link", "telegram", general4debtus.CreatedOn{
		CreatedOnPlatform: whc.BotPlatform().ID(),
		CreatedOnID:       whc.GetBotCode(),
	})
	var receipt models4debtus.ReceiptEntry
	if receipt, err = dal4debtus.Default.Receipt.CreateReceipt(whc.Context(), receiptData); err != nil {
		return m, err
	}
	receiptUrl := GetUrlForReceiptInTelegram(whc.GetBotCode(), receipt.ID, whc.Locale().Code5)
	m.Text = "Send this link to counterparty:\n\n" + fmt.Sprintf(`<a href="%v">%v</a>`, receiptUrl, receiptUrl) + "\n\nPlease be aware that the first person opening this link will be treated as counterparty for this debt."
	m.Format = botmsg.FormatHTML
	m.IsEdit = true
	return
}

func IsTransferNotificationsBlockedForChannel(counterparty *models4debtus.DebtusSpaceContactDbo, channel string) bool {
	for _, blockedBy := range counterparty.NoTransferUpdatesBy {
		if blockedBy == channel {
			return true
		}
	}
	return false
}
