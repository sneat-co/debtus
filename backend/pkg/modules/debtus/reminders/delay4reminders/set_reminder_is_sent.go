package delay4reminders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dal4reminders"
	"github.com/strongo/delaying"
)

func DelaySetReminderIsSent(ctx context.Context, reminderID string, sentAt time.Time, messageIntID int64, messageStrID, locale, errDetails string) error {
	if reminderID == "" {
		return errors.New("reminderID == 0")
	}
	if sentAt.IsZero() {
		return errors.New("sentAt.IsZero()")
	}
	if err := dal4reminders.ValidateSetReminderIsSentMessageIDs(messageIntID, messageStrID, sentAt); err != nil {
		return err
	}
	if err := delayer4debtus.SetReminderIsSent.EnqueueWork(ctx, delaying.With(queues.QueueReminders, "set-reminder-is-sent", 0), reminderID, sentAt, messageIntID, messageStrID, locale, errDetails); err != nil {
		return fmt.Errorf("failed to delay execution of DelayedSetReminderIsSent: %w", err)
	}
	return nil
}
