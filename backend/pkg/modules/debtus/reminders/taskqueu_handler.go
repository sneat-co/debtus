package reminders

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/delayed4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dal4reminders"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/strongo/logus"
)

// getTransferByID is a seam so tests can inject a fake transfer lookup.
var getTransferByID = func(ctx context.Context, tx dal.ReadSession, id string) (models4debtus.TransferEntry, error) {
	return facade4debtus.Transfers.GetTransferByID(ctx, tx, id)
}

// discardReminder is a seam so tests can inject a fake discard operation.
var discardReminder = func(ctx context.Context, reminderID, transferID, reason string) error {
	return delayed4debtus.DiscardReminder(ctx, reminderID, transferID, reason)
}

// sendReminderToUserFn is a seam so tests can bypass the full Telegram/email sending logic.
var sendReminderToUserFn = func(ctx context.Context, reminderID string, transfer models4debtus.TransferEntry) error {
	return sendReminderToUser(ctx, reminderID, transfer)
}

// getTelegramChatByUserID is a seam so tests can inject a fake Telegram chat lookup.
var getTelegramChatByUserID = func(ctx context.Context, userID string) (string, botsfwtgmodels.TgChatData, error) {
	return delayed4debtus.GetTelegramChatByUserID(ctx, userID)
}

// sendReminderByTelegramFn is a seam so tests can inject a fake Telegram sending operation.
var sendReminderByTelegramFn = func(ctx context.Context, transfer models4debtus.TransferEntry, reminder dbo4reminders.Reminder, tgChatID int64, tgBot string) (sent, channelDisabledByUser bool, err error) {
	return sendReminderByTelegram(ctx, transfer, reminder, tgChatID, tgBot)
}

func SendReminderHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	//func sendNotificationForDueTransfer(ctx context.Context, key *datastore.Key) {
	err := r.ParseForm()
	if err != nil {
		logus.Errorf(ctx, "Failed to parse form")
		return
	}
	reminderID := r.FormValue("id")
	if reminderID == "" {
		logus.Errorf(ctx, "Failed to convert reminder ContactID to int")
		return
	}
	if err = sendReminder(ctx, reminderID); err != nil {
		logus.Errorf(ctx, err.Error())
		if !dal.IsNotFound(err) {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func sendReminder(ctx context.Context, reminderID string) (err error) {
	logus.Debugf(ctx, "sendReminder(reminderID=%v)", reminderID)
	if reminderID == "" {
		return errors.New("reminderID == 0")
	}

	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	reminder, err := dal4reminders.GetReminderByID(ctx, db, reminderID)
	if err != nil {
		return err
	}
	if reminder.Data.Status != dbo4reminders.ReminderStatusCreated {
		logus.Infof(ctx, "reminder.Status:%v != models.ReminderStatusCreated", reminder.Data.Status)
		return nil
	}

	transfer, err := getTransferByID(ctx, nil, reminder.Data.TargetID)
	if err != nil {
		if dal.IsNotFound(err) {
			logus.Errorf(ctx, err.Error())
			if err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
				if reminder, err = dal4reminders.GetReminderByID(ctx, tx, reminderID); err != nil {
					return
				}
				reminder.Data.Status = "invalid:no-transfer"
				reminder.Data.DtUpdated = time.Now()
				reminder.Data.DtNext = time.Time{}
				if err = dal4reminders.SaveReminder(ctx, tx, reminder); err != nil {
					return
				}
				return
			}); err != nil {
				return fmt.Errorf("failed to update reminder: %w", err)
			}
			return nil
		} else {
			return fmt.Errorf("failed to load transfer: %w", err)
		}
	}

	if !transfer.Data.IsOutstanding {
		logus.Infof(ctx, "TransferEntry(id=%v) is not outstanding, transfer.Amount=%v, transfer.AmountInCentsReturned=%v", reminder.Data.TargetID, transfer.Data.AmountInCents, transfer.Data.AmountReturned())

		if err := discardReminder(ctx, reminderID, reminder.Data.TargetID, ""); err != nil {
			return fmt.Errorf("failed to discard a reminder for non outstanding transfer id=%v: %w", reminder.Data.TargetID, err)
		}
		return nil
	}

	if err = sendReminderToUserFn(ctx, reminderID, transfer); err != nil {
		logus.Errorf(ctx, "Failed to send reminder (id=%v) for transfer %v: %v", reminderID, reminder.Data.TargetID, err.Error())
	}

	return nil
}

var errReminderAlreadySentOrIsBeingSent = errors.New("reminder already sent or is being sent")

func sendReminderToUser(ctx context.Context, reminderID string, transfer models4debtus.TransferEntry) (err error) {

	var reminder dbo4reminders.Reminder

	// If sending notification failed, do not try to resend - to prevent spamming.
	if err = facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if reminder, err = dal4reminders.GetReminderByID(ctx, tx, reminderID); err != nil {
			return fmt.Errorf("failed to get reminder by id=%v: %w", reminderID, err)
		}
		if reminder.Data.Status != dbo4reminders.ReminderStatusCreated {
			return errReminderAlreadySentOrIsBeingSent
		}
		reminder.Data.Status = dbo4reminders.ReminderStatusSending
		if err = dal4reminders.SaveReminder(tctx, tx, reminder); err != nil { // TODO: UserEntry dal4debtus.Default.Reminder.SaveReminder()
			return fmt.Errorf("failed to save reminder with new status to db: %w", err)
		}
		return
	}, nil); err != nil {
		if errors.Is(err, errReminderAlreadySentOrIsBeingSent) {
			logus.Infof(ctx, err.Error())
		} else {
			err = fmt.Errorf("failed to update reminder status to '%v': %w", dbo4reminders.ReminderStatusSending, err)
			logus.Errorf(ctx, err.Error())
		}
		return
	} else {
		logus.Infof(ctx, "Updated Reminder(id=%v) status to '%v'.", reminderID, dbo4reminders.ReminderStatusSending)
	}

	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	user := dbo4userus.NewUserEntry(reminder.Data.UserID)
	if err = dal4userus.GetUser(ctx, db, user); err != nil {
		return err
	}

	var reminderIsSent, channelDisabledByUser bool
	if user.Data.HasAccount(string(telegram.PlatformID), "") {
		var (
			tgChatID int64
			tgBotID  string
		)
		if transferUserInfo := transfer.Data.UserInfoByUserID(reminder.Data.UserID); transferUserInfo.TgChatID != 0 {
			tgChatID = transferUserInfo.TgChatID
			tgBotID = transferUserInfo.TgBotID
		} else {
			var tgChat botsfwtgmodels.TgChatData
			_, tgChat, err = getTelegramChatByUserID(ctx, reminder.Data.UserID) // TODO: replace with DAL method
			if err != nil {
				if dal.IsNotFound(err) { // TODO: Get rid of datastore reference
					err = fmt.Errorf("failed to call delayed4debtus.GetTelegramChatByUserID(userID=%v): %w", reminder.Data.UserID, err)
					return
				}
			} else {
				if tgChatID, err = strconv.ParseInt(tgChat.BaseTgChatData().BotUserIDs[0], 10, 64); err != nil {
					err = fmt.Errorf("failed to parse tgChat.BaseTgChatData().BotUserID=%v: %w", tgChat.BaseTgChatData().BotUserIDs[0], err)
					return
				}
				tgBotID = "TODO:setup_bot_id_for_reminder"
			}
		}
		if tgChatID != 0 {
			if reminderIsSent, channelDisabledByUser, err = sendReminderByTelegramFn(ctx, transfer, reminder, tgChatID, tgBotID); err != nil {
				return
			} else if !reminderIsSent && !channelDisabledByUser {
				logus.Warningf(ctx, "Reminder is not sent to Telegram, err=%v", err)
			}
		}
	}
	if !reminderIsSent { // TODO: This is wrong to send same reminder by email if Telegram failed, complex and will screw up stats <= Are you sure?
		if user.Data.Email != "" {
			if err = sendReminderByEmail(ctx, reminder, user.Data.Email, transfer, user); err != nil {
				logus.Errorf(ctx, "Failure in sendReminderByEmail()")
			}
		} else {
			if !channelDisabledByUser {
				logus.Errorf(ctx, "Can't send reminder")
			}
			err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
				if reminder, err = dal4reminders.GetReminderByID(ctx, tx, reminderID); err != nil {
					return err
				}
				reminder.Data.Status = dbo4reminders.ReminderStatusFailed
				return dal4reminders.SaveReminder(ctx, tx, reminder)
			}, nil)
			if err != nil {
				logus.Errorf(ctx, fmt.Errorf("failed to set reminder status to '%v': %w", dbo4reminders.ReminderStatusFailed, err).Error())
			} else {
				logus.Infof(ctx, "Reminder status set to '%v'", reminder.Data.Status)
			}
		}
	}
	return nil // TODO: Handle errors!
}
