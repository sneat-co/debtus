package dal4reminders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/strongo/logus"
)

var ErrDuplicateAttemptToDiscardReminder = errors.New("duplicate attempt to close reminder by same return transfer")

func SetReminderStatus(ctx context.Context, reminderID, returnTransferID string, status string, when time.Time) (reminder dbo4reminders.Reminder, err error) {
	var (
		changed        bool
		previousStatus string
	)
	err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if reminder, err = GetReminderByID(ctx, tx, reminderID); err != nil {
			return
		} else {
			switch status {
			case dbo4reminders.ReminderStatusDiscarded:
				reminder.Data.DtDiscarded = when
			case dbo4reminders.ReminderStatusSent:
				reminder.Data.DtSent = when
			case dbo4reminders.ReminderStatusSending:
				// pass
			case dbo4reminders.ReminderStatusViewed:
				reminder.Data.DtViewed = when
			case dbo4reminders.ReminderStatusUsed:
				reminder.Data.DtUsed = when
			default:
				return errors.New("unsupported status: " + status)
			}
			previousStatus = reminder.Data.Status
			changed = previousStatus != status
			//if returnTransferID != "" && status == dbo4reminders.ReminderStatusDiscarded {
			//	for _, id := range reminder.Data.ClosedByTransferIDs { // TODO: WTF are we doing here?
			//		if id == returnTransferID {
			//			logus.Infof(ctx, "new status: '%v', Reminder{Status: '%v', ClosedByTransferIDs: %v}", status, reminder.Data.Status, reminder.Data.ClosedByTransferIDs)
			//			return ErrDuplicateAttemptToDiscardReminder
			//		}
			//	}
			//	reminder.Data.ClosedByTransferIDs = append(reminder.Data.ClosedByTransferIDs, returnTransferID)
			//	changed = true
			//}
			if changed {
				reminder.Data.Status = status
				if err = tx.Set(ctx, reminder.Record); err != nil {
					err = fmt.Errorf("failed to save reminder to db (id=%v): %w", reminderID, err)
				}
			}
			return
		}
	}, nil)
	if err == nil {
		if changed {
			logus.Debugf(ctx, "Reminder(id=%v) status changed from '%v' to '%v'", reminderID, previousStatus, status)
		} else {
			logus.Debugf(ctx, "Reminder(id=%v) status not changed as already '%v'", reminderID, status)
		}
	}
	return
}
