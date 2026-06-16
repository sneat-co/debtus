package reminders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/dtb_common"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/facade4debtusbot"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/analytics2debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/common4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/utmconsts"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dal4reminders"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/delay4reminders"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

// getLocale is a seam so tests can inject a fake locale lookup.
var getLocale = func(ctx context.Context, tgBot string, tgChatID int64, userID string) (i18n.Locale, error) {
	return facade4debtusbot.GetLocale(ctx, tgBot, tgChatID, userID)
}

// getBotSettingsByCode is a seam so tests can inject fake bot settings.
var getBotSettingsByCode = func(ctx context.Context, code string) (*botsfw.BotSettings, error) {
	return botsettings.GetBotSettingsByCode(ctx, code)
}

// newTgBotAPIFromSettings is a seam so tests can skip real HTTP-client creation.
var newTgBotAPIFromSettings = func(ctx context.Context, token string) *tgbotapi.BotAPI {
	return tgbotapi.NewBotAPIWithClient(token, dal4debtus.Default.HttpClient(ctx))
}

// tgBotAPISend is a seam so tests can inject a fake Telegram Send call.
var tgBotAPISend = func(bot *tgbotapi.BotAPI, c tgbotapi.Sendable) (tgbotapi.Message, error) {
	return bot.Send(c)
}

// reminderSent is a seam so tests can skip the real GA analytics call.
var reminderSent = func(ctx context.Context, userID string, userLanguage, platform string) {
	analytics2debtus.ReminderSent(ctx, userID, userLanguage, platform)
}

// delaySetChatIsForbiddenFn is a seam for tests to inject errors from DelaySetChatIsForbidden.
var delaySetChatIsForbiddenFn = func(ctx context.Context, botID string, tgChatID int64, at time.Time) error {
	return DelaySetChatIsForbidden(ctx, botID, tgChatID, at)
}

// setReminderIsSentInTx is a seam for tests to inject errors from SetReminderIsSentInTransaction.
var setReminderIsSentInTx = func(ctx context.Context, tx dal.ReadwriteTransaction, reminder dbo4reminders.Reminder, at time.Time, messageID int64, via, locale5, recipientID string) error {
	return dal4reminders.SetReminderIsSentInTransaction(ctx, tx, reminder, at, messageID, via, locale5, recipientID)
}

// delaySetReminderIsSent is a seam for tests to skip the real task-queue call.
var delaySetReminderIsSent = func(ctx context.Context, reminderID string, at time.Time, messageID int64, via, locale5, recipientID string) error {
	return delay4reminders.DelaySetReminderIsSent(ctx, reminderID, at, messageID, via, locale5, recipientID)
}

func sendReminderByTelegram(ctx context.Context, transfer models4debtus.TransferEntry, reminder dbo4reminders.Reminder, tgChatID int64, tgBot string) (sent, channelDisabledByUser bool, err error) {
	logus.Debugf(ctx, "sendReminderByTelegram(transfer.ContactID=%v, reminder.ContactID=%v, tgChatID=%v, tgBot=%v)", transfer.ID, reminder.ID, tgChatID, tgBot)

	if tgChatID == 0 {
		panic("tgChatID == 0")
	}
	if tgBot == "" {
		panic("tgBot is empty string")
	}

	var locale i18n.Locale

	if locale, err = getLocale(ctx, tgBot, tgChatID, reminder.Data.UserID); err != nil {
		return
	}

	//if !tgChat.DtForbidden.IsZero() {
	//	logus.Infof(ctx, "Telegram chat(id=%v) is not available since: %v", tgChatID, tgChat.DtForbidden)
	//	return false
	//}

	translator := i18n.NewSingleMapTranslator(locale, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))

	if botSettings, settingsErr := getBotSettingsByCode(ctx, tgBot); settingsErr != nil {
		err = fmt.Errorf("bot settings not found (tgBotID=%v): %w", tgBot, settingsErr)
		return
	} else {
		tgBotApi := newTgBotAPIFromSettings(ctx, botSettings.Token)
		messageText := fmt.Sprintf(
			"<b>%v</b>\n%v\n\n",
			translator.Translate(trans.MESSAGE_TEXT_REMINDER),
			translator.Translate(trans.MESSAGE_TEXT_REMINDER_ASK_IF_RETURNED),
		)

		utmParams := utm.Params{
			Source:   "TODO",
			Medium:   string(telegram.PlatformID),
			Campaign: utmconsts.UTM_CAMPAIGN_REMINDER,
		}
		messageText += common4debtus.TextReceiptForTransfer(ctx, translator, transfer, reminder.Data.UserID, common4debtus.ShowReceiptToAutodetect, utmParams)

		messageConfig := tgbotapi.NewMessage(tgChatID, messageText)

		err = facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			reminder, err = dal4reminders.GetReminderByID(ctx, tx, reminder.ID)
			if err != nil {
				return err
			}
			callbackData := fmt.Sprintf(dtb_common.DebtReturnCallbackData, dtb_common.CallbackDebtReturnedPath, reminder.ID, "%v")
			messageConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					{Text: translator.Translate(trans.COMMAND_TEXT_REMINDER_RETURNED_IN_FULL), CallbackData: fmt.Sprintf(callbackData, dtb_common.ReturnedFully)},
				},
				[]tgbotapi.InlineKeyboardButton{
					{Text: translator.Translate(trans.COMMAND_TEXT_REMINDER_RETURNED_PARTIALLY), CallbackData: fmt.Sprintf(callbackData, dtb_common.ReturnedPartially)},
				},
				[]tgbotapi.InlineKeyboardButton{
					{Text: translator.Translate(trans.COMMAND_TEXT_REMINDER_NOT_RETURNED), CallbackData: fmt.Sprintf(callbackData, dtb_common.ReturnedNothing)},
				},
			)
			messageConfig.ParseMode = "HTML"
			message, err := tgBotAPISend(tgBotApi, messageConfig)
			if err != nil {
				var errAPIForbidden tgbotapi.ErrAPIForbidden
				if errors.As(err, &errAPIForbidden) { // TODO: Mark chat as deleted?
					logus.Infof(ctx, "Telegram bot API returned status 'forbidden' - either issue with token or chat deleted by user")
					if err2 := delaySetChatIsForbiddenFn(ctx, botSettings.Code, tgChatID, time.Now()); err2 != nil {
						logus.Errorf(ctx, "Failed to delay to set chat as forbidden: %v", err2)
					}
					channelDisabledByUser = true
					return nil // Do not pass error up
				}
			}
			sent = true
			logus.Infof(ctx, "Sent message to telegram. MessageID: %v", message.MessageID)

			if err = setReminderIsSentInTx(tctx, tx, reminder, time.Now(), int64(message.MessageID), "", locale.Code5, ""); err != nil {
				err = delaySetReminderIsSent(tctx, reminder.ID, time.Now(), int64(message.MessageID), "", locale.Code5, "")
			}
			//
			return
		}, nil)

		if err != nil {
			logus.Errorf(ctx, fmt.Errorf("error while sending by Telegram: %w", err).Error())
			return
		}
		if sent {
			reminderSent(ctx, reminder.Data.UserID, translator.Locale().Code5, string(telegram.PlatformID))
		}
	}
	return
}

func DelaySetChatIsForbidden(ctx context.Context, botID string, tgChatID int64, at time.Time) error {
	return delaySetChatIsForbidden.EnqueueWork(ctx, delaying.With(queues.QueueChats, "set-chat-is-forbidden", 0), botID, tgChatID, at)
}

func SetChatIsForbidden(ctx context.Context, botID string, tgChatID int64, at time.Time) error {
	logus.Debugf(ctx, "SetChatIsForbidden(tgChatID=%v, at=%v)", tgChatID, at)
	_ = botID
	panic("TODO: Implement SetChatIsForbidden")
	//err := gaehost.MarkTelegramChatAsForbidden(ctx, botID, tgChatID, at)
	//if err == nil {
	//	logus.Infof(ctx, "Success")
	//} else {
	//	logus.Errorf(ctx, err.Error())
	//	if err == datastore.ErrNoSuchEntity {
	//		return nil // Do not re-try
	//	}
	//}
	//return err
}
