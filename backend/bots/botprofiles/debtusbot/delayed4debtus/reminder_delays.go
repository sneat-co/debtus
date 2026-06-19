package delayed4debtus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/utmconsts"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dal4reminders"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func DelayedCreateReminderForTransferUser(ctx context.Context, transferID string, userID string) (err error) {
	logus.Debugf(ctx, "DelayedCreateReminderForTransferUser(transferID=%s, userID=%s)", transferID, userID)
	if transferID == "" {
		logus.Errorf(ctx, "transferID == 0")
		return nil
	}
	if userID == "" {
		logus.Errorf(ctx, "userID == 0")
		return nil
	}

	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		var transfer models4debtus.TransferEntry
		transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
		if err != nil {
			if dal.IsNotFound(err) {
				logus.Errorf(ctx, fmt.Errorf("not able to create reminder for specified transfer: %w", err).Error())
				return
			}
			return fmt.Errorf("failed to get transfer by id: %w", err)
		}
		transferUserInfo := transfer.Data.UserInfoByUserID(userID)
		if transferUserInfo.UserID != userID {
			return fmt.Errorf("transferUserInfo.UserID != userID: %v != %v", transferUserInfo.UserID, userID)
		}

		if transferUserInfo.ReminderID != "" {
			logus.Warningf(ctx, "TransferEntry user already has reminder # %v", transferUserInfo.ReminderID)
			return
		}

		if transferUserInfo.TgChatID == 0 { // TODO: Try to get TgChat from user record or check other channels?
			logus.Warningf(ctx, "TransferEntry user has no associated TgChatID: %+v", transferUserInfo)
			return
		}

		//reminderKey := NewReminderIncompleteKey(ctx)
		next := transfer.Data.DtDueOn
		isAutomatic := next.IsZero()
		if isAutomatic {
			if strings.Contains(strings.ToLower(transfer.Data.CreatedOnID), "dev") {
				next = time.Now().Add(2 * time.Minute)
			} else {
				next = time.Now().Add(7 * 24 * time.Hour)
			}
		}
		reminder := dbo4reminders.NewReminder("", dbo4reminders.NewReminderViaTelegram(transferUserInfo.TgBotID, transferUserInfo.TgChatID, userID, transferID, isAutomatic, next))
		if err = tx.Insert(ctx, reminder.Record); err != nil {
			return fmt.Errorf("failed to save reminder to db: %w", err)
		}
		reminderID := reminder.Key.ID.(string)
		logus.Infof(ctx, "Created reminder id=%v", reminderID)
		if err = debtusdal.QueueSendReminder(ctx, reminderID, time.Until(next)); err != nil {
			return fmt.Errorf("failed to queue reminder for sending: %w", err)
		}
		transferUserInfo.ReminderID = reminderID

		if err = facade4debtus.Transfers.SaveTransfer(ctx, tx, transfer); err != nil {
			return fmt.Errorf("failed to save transfer to db: %w", err)
		}

		return
	})
}

func DelayedDiscardRemindersForTransfers(ctx context.Context, transferIDs []string, returnTransferID string) error {
	logus.Debugf(ctx, "DelayedDiscardRemindersForTransfers(transferIDs=%+v, returnTransferID=%s)", transferIDs, returnTransferID)
	if len(transferIDs) == 0 {
		return errors.New("len(transferIDs) == 0")
	}
	const queueName = queues.QueueReminders
	args := make([][]interface{}, len(transferIDs))
	for i, transferID := range transferIDs {
		args[i] = []interface{}{transferID, returnTransferID}
	}
	return delayer4debtus.DiscardRemindersForTransfer.EnqueueWorkMulti(ctx, delaying.With(queueName, "discard-reminders-for-transfer", 0), args...)
}

func DelayedDiscardRemindersForTransfer(ctx context.Context, transferID, returnTransferID string) error {
	logus.Debugf(ctx, "DelayedDiscardRemindersForTransfers(transferID=%v, returnTransferID=%v)", transferID, returnTransferID)
	if transferID == "" {
		logus.Errorf(ctx, "transferID is empty")
		return nil
	}
	delayDuration := time.Millisecond * 10
	var _discard = func(
		getIDs func(context.Context, dal.ReadSession, string) ([]string, error),
		loadedFormat, notLoadedFormat string,
	) error {
		if reminderIDs, err := getIDs(ctx, nil, transferID); err != nil {
			return err
		} else if len(reminderIDs) > 0 {
			logus.Debugf(ctx, loadedFormat, len(reminderIDs), transferID)
			for _, reminderID := range reminderIDs {
				if err := delayer4debtus.DiscardReminderForTransfer.EnqueueWork(ctx, delaying.With(queues.QueueReminders, "discard-reminder", delayDuration), reminderID, transferID, returnTransferID); err != nil {
					return fmt.Errorf("failed to create a task for reminder ContactID=%v: %w", reminderID, err)
				}
				delayDuration += time.Millisecond * 10
			}
		} else {
			logus.Infof(ctx, notLoadedFormat, transferID)
		}
		return nil
	}
	if err := _discard(dal4debtus.Default.Reminder.GetActiveReminderIDsByTransferID, "Loaded %v keys of active reminders for transfer id=%v", "The are no ative reminders for transfer id=%v"); err != nil {
		return err
	}
	if err := _discard(dal4debtus.Default.Reminder.GetSentReminderIDsByTransferID, "Loaded %v keys of sent reminders for transfer id=%v", "The are no sent reminders for transfer id=%v"); err != nil {
		return err
	}
	return nil
}

func DiscardReminder(ctx context.Context, reminderID, transferID, returnTransferID string) (err error) {
	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		return discardReminder(ctx, tx, reminderID, transferID, returnTransferID)
	})
}

func DelayedDiscardReminderForTransfer(ctx context.Context, reminderID, transferID, returnTransferID string) (err error) {
	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if err = discardReminder(ctx, tx, reminderID, transferID, returnTransferID); errors.Is(err, dal4reminders.ErrDuplicateAttemptToDiscardReminder) {
			logus.Errorf(ctx, err.Error())
			return nil
		}
		return err
	})
}

func discardReminder(ctx context.Context, tx dal.ReadwriteTransaction, reminderID, transferID, returnTransferID string) (err error) {
	logus.Debugf(ctx, "discardReminder(reminderID=%v, transferID=%v, returnTransferID=%v)", reminderID, transferID, returnTransferID)

	var (
		transfer = models4debtus.NewTransfer(transferID, nil)
		reminder = dbo4reminders.NewReminder(reminderID, new(dbo4reminders.ReminderDbo))
	)

	if returnTransferID > "" {
		//returnTransferKey := models.NewTransferKey(returnTransferID)
		returnTransfer := models4debtus.NewTransfer(returnTransferID, nil)
		//keys := []*datastore.Key{reminderKey, transferKey, returnTransferKey}
		if err = tx.GetMulti(ctx, []dal.Record{reminder.Record, transfer.Record, returnTransfer.Record}); err != nil {
			return err
		}
	} else {
		if err = tx.GetMulti(ctx, []dal.Record{reminder.Record, transfer.Record}); err != nil {
			return err
		}
	}

	if reminder, err = dal4reminders.SetReminderStatus(ctx, reminderID, returnTransferID, dbo4reminders.ReminderStatusDiscarded, time.Now()); err != nil {
		return err // DO NOT WRAP as there is check in DelayedDiscardReminderForTransfer() errors.Wrapf(err, "Failed to set reminder status to '%v'", models.ReminderStatusDiscarded)
	}

	switch reminder.Data.SentVia {
	case "telegram": // We need to update a reminder message if it was already sent out
		if reminder.Data.BotID == "" {
			logus.Errorf(ctx, "reminder.BotID == ''")
			return nil
		}
		if reminder.Data.MessageIntID == 0 {
			//logus.Infof(ctx, "No need to update reminder message in Telegram as a reminder is not sent yet")
			return nil
		}
		logus.Infof(ctx, "Will try to update a reminder message as it was already sent to user, reminder.MessageIntID: %v", reminder.Data.MessageIntID)
		tgBotApi, err := getTelegramBotApiFn(ctx, reminder.Data.BotID)
		if err != nil {
			return fmt.Errorf("not able to create API client as there no settings for telegram bot with id '%v': %w", reminder.Data.BotID, err)
		}

		if reminder.Data.Locale == "" {
			logus.Errorf(ctx, "reminder.Locale == ''")
			user := dbo4userus.NewUserEntry(reminder.Data.UserID)
			if err = dal4userus.GetUser(ctx, nil, user); err != nil { // Intentionally do not use transaction
				return err
			}
			if user.Data.PreferredLocale != "" {
				reminder.Data.Locale = user.Data.PreferredLocale
			} else if s, sErr := botsettings.GetBotSettingsByCode(ctx, reminder.Data.BotID); sErr == nil {
				reminder.Data.Locale = s.Locale.Code5
			}
		}

		translator := GetTranslatorForReminder(ctx, reminder.Data)

		utmParams := utm.Params{
			Source:   "TODO", // TODO: Get bot ContactID
			Medium:   "telegram",
			Campaign: utmconsts.UTM_CAMPAIGN_RECEIPT_DISCARD,
		}

		receiptMessageText := common4debtus.TextReceiptForTransfer(
			ctx,
			translator,
			transfer,
			reminder.Data.UserID,
			common4debtus.ShowReceiptToAutodetect,
			utmParams,
		)

		locale := i18n.GetLocaleByCode5(reminder.Data.Locale) // TODO: Check for supported locales

		transferUrlForUser := common4debtus.GetTransferUrlForUser(ctx, transferID, reminder.Data.UserID, locale, utmParams)

		receiptMessageText += "\n\n" + strings.Join([]string{
			translator.Translate(trans.MESSAGE_TEXT_DEBT_IS_RETURNED),
			fmt.Sprintf(`<a href="%v">%v</a>`, transferUrlForUser, translator.Translate(trans.MESSAGE_TEXT_DETAILS_ARE_HERE)),
		}, "\n")

		tgMessage := tgbotapi.NewEditMessageText(reminder.Data.ChatIntID, int(reminder.Data.MessageIntID), "", receiptMessageText)
		tgMessage.ParseMode = "HTML"
		if _, err = tgBotApi.Send(tgMessage); err != nil {
			return fmt.Errorf("failed to send message to Telegram: %w", err)
		}

	default:
		return errors.New("Unknown reminder channel: %v" + reminder.Data.SentVia)
	}

	return err
}

func GetTranslatorForReminder(ctx context.Context, reminder *dbo4reminders.ReminderDbo) i18n.SingleLocaleTranslator {
	return i18n.NewSingleMapTranslator(i18n.GetLocaleByCode5(reminder.Locale), i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
}
