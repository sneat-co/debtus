package debtusdal

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	dalgorecord "github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/delayer4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
)

func NewReminderKey(reminderID string) *dal.Key {
	if reminderID == "" {
		panic("reminderID == 0")
	}
	return dal.NewKeyWithID(dbo4reminders.ReminderKind, reminderID)
}

type ReminderDal struct {
}

func NewReminderDal() ReminderDal {
	return ReminderDal{}
}

var _ dal4debtus.ReminderDal = (*ReminderDal)(nil)

var reminderCollection = dal.CollectionAt[string, dbo4reminders.ReminderDbo](dbo4reminders.ReminderKind)

func (reminderDal ReminderDal) GetReminderByID(ctx context.Context, tx dal.ReadSession, id string) (reminder dbo4reminders.Reminder, err error) {
	var dbo dbo4reminders.ReminderDbo
	if dbo, err = reminderCollection.GetData(ctx, tx, id); err != nil {
		return
	}
	reminder = dalgorecord.NewDataWithID(id, NewReminderKey(id), &dbo)
	return
}

func (reminderDal ReminderDal) GetSentReminderIDsByTransferID(ctx context.Context, tx dal.ReadSession, transferID string) ([]string, error) {
	q := dal.From(dal.NewRootCollectionRef(dbo4reminders.ReminderKind, "")).
		NewQuery().
		Where(
			dal.WhereField("targetID", dal.Equal, transferID),
			dal.WhereField("status", dal.Equal, dbo4reminders.ReminderStatusSent),
		).SelectKeysOnly(reflect.String)

	records, err := dal.ExecuteQueryAndReadAllToRecords(ctx, q, tx)
	if err != nil {
		return nil, err
	}
	return reminderIDsFromRecords(records)
}

// reminderIDsFromRecords extracts string reminder IDs from query result records.
func reminderIDsFromRecords(records []dal.Record) ([]string, error) {
	reminderIDs := make([]string, len(records))
	for i, r := range records {
		id, ok := r.Key().ID.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type of reminder key ID: %T", r.Key().ID)
		}
		reminderIDs[i] = id
	}
	return reminderIDs, nil
}

func (reminderDal ReminderDal) GetActiveReminderIDsByTransferID(ctx context.Context, tx dal.ReadSession, transferID string) ([]string, error) {
	q := dal.From(dal.NewRootCollectionRef(dbo4reminders.ReminderKind, "")).
		NewQuery().
		Where(
			dal.WhereField("targetID", dal.Equal, transferID),
			dal.WhereField("dtNext", dal.GreaterThen, time.Time{}),
		).SelectKeysOnly(reflect.String)
	records, err := dal.ExecuteQueryAndReadAllToRecords(ctx, q, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active reminders by transfer id=%v: %w", transferID, err)
	}
	return reminderIDsFromRecords(records)
}

func (ReminderDal) DelayCreateReminderForTransferUser(ctx context.Context, transferID string, userID string) (err error) {
	if transferID == "" {
		panic("transferID == 0")
	}
	if userID == "" {
		panic("userID == 0")
	}
	//if !dal4debtus.DB.IsInTransaction(ctx) {
	//	panic("This function should be called within transaction")
	//}
	if err = delayer4debtus.CreateReminderForTransferUser.EnqueueWork(ctx, delaying.With(queues.QueueReminders, "create-reminder-4-transfer-user", 0), transferID, userID); err != nil {
		return fmt.Errorf("failed to create a task for reminder creation. transferID=%v, userID=%v: %w", transferID, userID, err)
	}
	logus.Debugf(ctx, "Added task to create reminder for transfer id=%s", transferID)
	return
}
func (ReminderDal) DelayDiscardRemindersForTransfers(ctx context.Context, transferIDs []string, returnTransferID string) error {
	if len(transferIDs) > 0 {
		return delayer4debtus.DiscardRemindersForTransfers.EnqueueWork(ctx, delaying.With(queues.QueueReminders, "discard-reminders", 0), transferIDs, returnTransferID)
	} else {
		logus.Warningf(ctx, "DelayDiscardRemindersForTransfers(): len(transferIDs)==0")
		return nil
	}
}
