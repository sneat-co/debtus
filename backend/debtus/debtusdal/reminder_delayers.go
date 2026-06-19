package debtusdal

import (
	"context"
	"errors"
	"time"

	"github.com/sneat-co/debtus/backend/debtus/reminders/dal4reminders"
	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/strongo/delaying"
)

func DelayedSetReminderIsSent(ctx context.Context, reminderID string, sentAt time.Time, messageIntID int64, messageStrID, locale, errDetails string) error {
	return dal4reminders.SetReminderIsSent(ctx, reminderID, sentAt, messageIntID, messageStrID, locale, errDetails)
}

// DelayerSendReminder sends a due reminder. It is assigned by
// reminders.InitDelaying (pkg/reminders imports this package, so the delayed
// func cannot live here without an import cycle).
var DelayerSendReminder delaying.Delayer

func CreateSendReminderTask(ctx context.Context, reminderID string) (err error) {
	return createSendReminderTask(ctx, reminderID, 0)
}

func createSendReminderTask(ctx context.Context, reminderID string, delay time.Duration) error {
	if reminderID == "" {
		return errors.New("reminderID is empty string")
	}
	if DelayerSendReminder == nil {
		return errors.New("debtusdal.DelayerSendReminder is not initialized")
	}
	return DelayerSendReminder.EnqueueWork(ctx, delaying.With(queues.QueueReminders, "send-reminder", delay), reminderID)
}

func QueueSendReminder(ctx context.Context, reminderID string, dueIn time.Duration) error {
	if dueIn < 3*time.Hour {
		var delay time.Duration
		if dueIn > 0 {
			delay = dueIn + 3*time.Second
		}
		return createSendReminderTask(ctx, reminderID, delay)
	}
	return nil
}
