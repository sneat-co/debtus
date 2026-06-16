package delayed4debtus

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botsdal"
	"github.com/bots-go-framework/bots-fw/botsfwconst"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
	"github.com/strongo/strongoapp/appuser"
)

func delaySendReceiptToCounterpartyByTelegram(ctx context.Context, receiptID string, tgChatID int64, localeCode string) error {
	return delayer4debtus.SendReceiptToCounterpartyByTelegram.EnqueueWork(ctx, delaying.With(const4debtus.QueueReceipts, "send-receipt-to-counterparty-by-telegram", time.Second/10), receiptID, tgChatID, localeCode)
}

func DelayedSendReceiptToCounterpartyByTelegram(ctx context.Context, receiptID string, tgChatID int64, localeCode string) (err error) {
	logus.Debugf(ctx, "DelayedSendReceiptToCounterpartyByTelegram(receiptID=%v, tgChatID=%v, localeCode=%v)", receiptID, tgChatID, localeCode)

	if err := facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		var receipt models4debtus.ReceiptEntry

		if receipt, err = updateReceiptStatus(ctx, tx, receiptID, models4debtus.ReceiptStatusCreated, models4debtus.ReceiptStatusSending); err != nil {
			logus.Errorf(ctx, err.Error())
			err = nil // Always stop!
			return
		}

		var transfer models4debtus.TransferEntry
		if transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, receipt.Data.TransferID); err != nil {
			logus.Errorf(ctx, err.Error())
			if dal.IsNotFound(err) {
				err = nil
				return
			}
			return
		}

		counterpartyUser := dbo4userus.NewUserEntry(receipt.Data.CounterpartyUserID)
		if err = dal4userus.GetUser(ctx, tx, counterpartyUser); err != nil {
			return
		}

		var (
			tgChat         anybot.SneatAppTgChatEntry
			failedToSend   bool
			chatsForbidden bool
		)

		creatorTgChatID, creatorTgMsgID := transfer.Data.Creator().TgChatID, int(transfer.Data.CreatorTgReceiptByTgMsgID)

		var tgAccounts []appuser.AccountKey

		if tgAccounts, err = counterpartyUser.Data.GetAccounts("telegram"); err != nil {
			return err
		}
		for _, telegramAccount := range tgAccounts {
			if telegramAccount.App == "" {
				logus.Warningf(ctx, "UserEntry %v has account with missing bot id => %v", counterpartyUser.ID, telegramAccount.String())
				continue
			}
			// For Telegram private chats the chat ID equals the Telegram user ID stored in the user account record.
			tgChatKey := botsdal.NewBotChatKey(botsfwconst.PlatformTelegram, telegramAccount.App, telegramAccount.ID)
			tgChatData := new(anybot.SneatAppTgChatDbo)
			tgChat = anybot.SneatAppTgChatEntry{
				RecordWithID: record.NewWithID(telegramAccount.ID, tgChatKey, tgChatData),
				Data:   tgChatData,
			}
			if err = tx.Get(ctx, tgChat.Record); err != nil {
				logus.Errorf(ctx, "failed to load user's Telegram chat entity: %v", err)
				err = nil
				continue
			}
			if tgChat.Data.DtForbiddenLast.IsZero() {
				if err = sendReceiptToTelegramChat(ctx, receipt, transfer, tgChat); err != nil {
					failedToSend = true
					var errAPIForbidden tgbotapi.ErrAPIForbidden
					if errors.As(err, &errAPIForbidden) || strings.Contains(err.Error(), "Bad Request: chat not found") {
						chatsForbidden = true
						logus.Infof(ctx, "Telegram chat not found or disabled (%v): %v", tgChat.ID, err)
						tgChat.Data.DtForbiddenLast = time.Now()
						if err2 := tx.Set(ctx, tgChat.Record); err2 != nil {
							logus.Errorf(ctx, "Failed to mark Telegram chat as forbidden: %v", err2.Error())
						}
						continue // try next telegram account of the counterparty
					}
					return
				}
				if err = delayOnReceiptSentSuccess(ctx, time.Now(), receipt.ID, transfer.ID, creatorTgChatID, creatorTgMsgID, tgChat.Key.Parent().ID.(string), localeCode); err != nil {
					logus.Errorf(ctx, fmt.Errorf("failed to call delayOnReceiptSentSuccess(): %w", err).Error())
				}
				return
			} else {
				logus.Debugf(ctx, "tgChat is forbidden: %v", telegramAccount.String())
			}
			break
		}

		if failedToSend { // Notify creator that receipt has not been sent
			sendErr := err // capture before getTranslator overwrites err
			var translator i18n.SingleLocaleTranslator
			if translator, err = getTranslator(ctx, localeCode); err != nil {
				return err
			}

			locale := translator.Locale()
			if chatsForbidden {
				msgTextToCreator := emoji.ERROR_ICON + translator.Translate(trans.MESSAGE_TEXT_RECEIPT_NOT_SENT_AS_COUNTERPARTY_HAS_DISABLED_TG_BOT, transfer.Data.Counterparty().ContactName)
				if err2 := delayOnReceiptSendFail(ctx, receipt.ID, creatorTgChatID, creatorTgMsgID, time.Now(), translator.Locale().Code5, msgTextToCreator); err2 != nil {
					logus.Errorf(ctx, fmt.Errorf("failed to update receipt entity with error info: %w", err2).Error())
				}
			}
			logus.Errorf(ctx, "Failed to send notification to creator by Telegram (creatorTgChatID=%v, creatorTgMsgID=%v): %v", creatorTgChatID, creatorTgMsgID, sendErr)
			if sendErr != nil {
				msgTextToCreator := emoji.ERROR_ICON + " " + sendErr.Error()
				if err2 := delayOnReceiptSendFail(ctx, receipt.ID, creatorTgChatID, creatorTgMsgID, time.Now(), locale.Code5, msgTextToCreator); err2 != nil {
					logus.Errorf(ctx, fmt.Errorf("failed to update receipt entity with error info: %w", err2).Error())
				}
			}
			err = nil
		}
		return err
	}); err != nil {
		return err
	}
	return err
}

// sendReceiptToTelegramChat is a var to allow stubbing in unit tests.
var sendReceiptToTelegramChat = sendReceiptToTelegramChatReal

func sendReceiptToTelegramChatReal(ctx context.Context, receipt models4debtus.ReceiptEntry, transfer models4debtus.TransferEntry, tgChat anybot.SneatAppTgChatEntry) (err error) {
	var messageToTranslate string
	switch transfer.Data.Direction() {
	case models4debtus.TransferDirectionUser2Counterparty:
		messageToTranslate = trans.TELEGRAM_RECEIPT
	case models4debtus.TransferDirectionCounterparty2User:
		messageToTranslate = trans.TELEGRAM_RECEIPT
	default:
		return fmt.Errorf("unknown direction: %v", transfer.Data.Direction())
	}

	templateData := struct {
		FromName         string
		TransferCurrency string
	}{
		FromName:         transfer.Data.Creator().ContactName,
		TransferCurrency: string(transfer.Data.Currency),
	}

	var translator i18n.SingleLocaleTranslator
	if translator, err = getTranslator(ctx, tgChat.Data.GetPreferredLanguage()); err != nil {
		return err
	}

	messageText, err := common4all.TextTemplates.RenderTemplate(ctx, translator, messageToTranslate, templateData)
	if err != nil {
		return err
	}
	messageText = emoji.INCOMING_ENVELOP_ICON + " " + messageText

	logus.Debugf(ctx, "Message: %v", messageText)

	btnViewReceiptText := emoji.CLIPBOARD_ICON + " " + translator.Translate(trans.BUTTON_TEXT_SEE_RECEIPT_DETAILS)
	btnViewReceiptData := fmt.Sprintf("view-receipt?id=%s", receipt.ID) // TODO: Pass simple digits!

	var telegramUserID int64
	if telegramUserID, err = strconv.ParseInt(tgChat.Data.BotUserIDs[0], 10, 64); err != nil {
		return err
	}
	tgMessage := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: telegramUserID,
			ReplyMarkup: tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(btnViewReceiptText, btnViewReceiptData)),
				},
			},
		},
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		Text:                  messageText,
	}

	var tgBotApi *tgbotapi.BotAPI
	if tgBotApi, err = getTelegramBotApiFn(ctx, tgChat.Key.Parent().ID.(string)); err != nil {
		return
	}

	if _, err = tgBotApi.Send(tgMessage); err != nil {
		return
	} else {
		logus.Infof(ctx, "ReceiptEntry %v sent to user by Telegram bot @%v", receipt.ID, tgChat.Key.Parent().ID.(string))
	}

	err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if receipt, err = updateReceiptStatus(ctx, tx, receipt.ID, models4debtus.ReceiptStatusSending, models4debtus.ReceiptStatusSent); err != nil {
			logus.Errorf(ctx, err.Error())
			err = nil
			return
		}
		return err
	})
	return
}

func DelayedCreateAndSendReceiptToCounterpartyByTelegram(ctx context.Context, env string, transferID string, toUserID string) error {
	logus.Debugf(ctx, "delayerCreateAndSendReceiptToCounterpartyByTelegram(transferID=%v, toUserID=%v)", transferID, toUserID)
	_ = env
	if transferID == "" {
		logus.Errorf(ctx, "transferID == 0")
		return nil
	}
	if toUserID == "" {
		logus.Errorf(ctx, "toUserID == 0")
		return nil
	}
	chatEntityID, tgChat, err := getTelegramChatByUserIDFn(ctx, toUserID)
	if err != nil {
		err2 := fmt.Errorf("failed to get Telegram chat for user (id=%v): %w", toUserID, err)
		if dal.IsNotFound(err) {
			logus.Infof(ctx, "No telegram for user or user not found")
			return nil
		} else {
			return err2
		}
	}
	if chatEntityID == "" {
		logus.Infof(ctx, "No telegram for user")
		return nil
	}
	localeCode := tgChat.BaseTgChatData().PreferredLanguage

	if err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		var transfer models4debtus.TransferEntry
		transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
		if err != nil {
			if dal.IsNotFound(err) {
				logus.Errorf(ctx, err.Error())
				return nil
			}
			return fmt.Errorf("failed to get transfer by id=%v: %v", transferID, err)
		}
		if localeCode == "" {
			toUser, err := dal4userus.GetUserByID(ctx, tx, toUserID)
			if err != nil {
				return err
			}
			localeCode = toUser.Data.GetPreferredLocale()
		}

		var translator i18n.SingleLocaleTranslator
		if translator, err = getTranslator(ctx, localeCode); err != nil {
			return err
		}
		locale := translator.Locale()

		var receiptID string
		receipt := models4debtus.NewReceipt("", models4debtus.NewReceiptEntity(transfer.Data.CreatorUserID, transferID, transfer.Data.Counterparty().UserID, locale.Code5, "telegram", tgChat.BaseTgChatData().BotUserIDs[0], general4debtus.CreatedOn{
			CreatedOnID:       transfer.Data.Creator().TgBotID, // TODO: Replace with method call.
			CreatedOnPlatform: transfer.Data.CreatedOnPlatform,
		}))
		if err := tx.Set(ctx, receipt.Record); err != nil {
			return fmt.Errorf("failed to save receipt to DB: %w", err)
		} else {
			receiptID = receipt.Record.Key().ID.(string)
		}
		if err != nil {
			return fmt.Errorf("failed to create receipt entity: %w", err)
		}
		var tgChatID int64
		if tgChatID, err = strconv.ParseInt(tgChat.BaseTgChatData().BotUserIDs[0], 10, 64); err != nil {
			return err
		}
		if err = delaySendReceiptToCounterpartyByTelegram(ctx, receiptID, tgChatID, localeCode); err != nil { // TODO: ideally should be called inside transaction
			logus.Errorf(ctx, "failed to queue receipt sending: %v", err)
			return nil
		}
		return err
	}); err != nil {
		return err
	}
	return nil
}
