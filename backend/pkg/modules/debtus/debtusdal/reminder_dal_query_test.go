package debtusdal

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestReminderDalGae_GetReminderIDsByTransferID(t *testing.T) {
	ctx := context.Background()
	const transferID = "transfer-1"

	seed := func(t *testing.T, db dal.DB, id string, mutate func(*dbo4reminders.ReminderDbo)) {
		t.Helper()
		dbo := dbo4reminders.NewReminderViaTelegram("test_bot", 123, "user-1", transferID, false, time.Now().Add(time.Hour))
		if mutate != nil {
			mutate(dbo)
		}
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, dbo4reminders.NewReminder(id, dbo).Record)
		})
		if err != nil {
			t.Fatalf("failed to seed reminder %s: %v", id, err)
		}
	}

	db := sneattesting.SetupMemoryDB(t)
	seed(t, db, "active-1", nil)
	seed(t, db, "sent-1", func(dbo *dbo4reminders.ReminderDbo) {
		dbo.Status = dbo4reminders.ReminderStatusSent
		dbo.DtScheduled = dbo.DtNext
		dbo.DtNext = time.Time{}
	})
	seed(t, db, "other-transfer", func(dbo *dbo4reminders.ReminderDbo) {
		dbo.TargetID = "transfer-2"
	})

	t.Run("active", func(t *testing.T) {
		ids, err := NewReminderDal().GetActiveReminderIDsByTransferID(ctx, db, transferID)
		if err != nil {
			t.Fatalf("GetActiveReminderIDsByTransferID() returned error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "active-1" {
			t.Errorf("ids = %v, want [active-1]", ids)
		}
	})

	t.Run("sent", func(t *testing.T) {
		ids, err := NewReminderDal().GetSentReminderIDsByTransferID(ctx, db, transferID)
		if err != nil {
			t.Fatalf("GetSentReminderIDsByTransferID() returned error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "sent-1" {
			t.Errorf("ids = %v, want [sent-1]", ids)
		}
	})
}
