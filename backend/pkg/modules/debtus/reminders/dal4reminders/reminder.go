package dal4reminders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/strongo/logus"
)

func GetReminderByID(ctx context.Context, tx dal.ReadSession, id string) (reminder dbo4reminders.Reminder, err error) {
	reminder = dbo4reminders.NewReminder(id, nil)
	return reminder, tx.Get(ctx, reminder.Record)
}

func SaveReminder(ctx context.Context, tx dal.ReadwriteTransaction, reminder dbo4reminders.Reminder) (err error) {
	return tx.Set(ctx, reminder.Record)
}

func SetReminderIsSent(ctx context.Context, reminderID string, sentAt time.Time, messageIntID int64, messageStrID, locale, errDetails string) (err error) {
	//gaehost.GaeLogger.Debugf(ctx, "DelayedSetReminderIsSent(reminderID=%v, sentAt=%v, messageIntID=%v, messageStrID=%v)", reminderID, sentAt, messageIntID, messageStrID)
	if err := ValidateSetReminderIsSentMessageIDs(messageIntID, messageStrID, sentAt); err != nil {
		return err
	}
	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		reminder, err := GetReminderByID(ctx, tx, reminderID)
		if err != nil {
			if dal.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get reminder by ContactID: %w", err)
		}
		return SetReminderIsSentInTransaction(ctx, tx, reminder, sentAt, messageIntID, messageStrID, locale, errDetails)
	})
}

func SetReminderIsSentInTransaction(ctx context.Context, tx dal.ReadwriteTransaction, reminder dbo4reminders.Reminder, sentAt time.Time, messageIntID int64, messageStrID, locale, errDetails string) (err error) {
	if reminder.Data == nil {
		reminder, err = GetReminderByID(ctx, tx, reminder.ID)
		if err != nil {
			if dal.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get reminder by ContactID: %w", err)
		}
	}
	if reminder.Data.Status != dbo4reminders.ReminderStatusSending {
		logus.Errorf(ctx, "reminder.Status:%v != models.ReminderStatusSending:%v", reminder.Data.Status, dbo4reminders.ReminderStatusSending)
		return nil
	} else {
		reminder.Data.Status = dbo4reminders.ReminderStatusSent
		reminder.Data.DtSent = sentAt
		reminder.Data.DtScheduled = reminder.Data.DtNext
		reminder.Data.DtNext = time.Time{}
		reminder.Data.ErrDetails = errDetails
		reminder.Data.Locale = locale
		if messageIntID != 0 {
			reminder.Data.MessageIntID = messageIntID
		}
		if messageStrID != "" {
			reminder.Data.MessageStrID = messageStrID
		}
		if err = tx.Set(ctx, reminder.Record); err != nil {
			err = fmt.Errorf("failed to save reminder to datastore: %w", err)
		}
		return err
	}
}

func RescheduleReminder(ctx context.Context, reminderID string, remindInDuration time.Duration) (oldReminder, newReminder dbo4reminders.Reminder, err error) {
	return dbo4reminders.Reminder{}, dbo4reminders.Reminder{}, errors.New("not implemented - needs to be refactored")
}
