package debtusdal

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/delaying"
)

func TestReminderDalGae_GetReminderByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_reminder_when_exists", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			// dbo4reminders.NewReminder passes nil key — use NewReminderKey to build
			// a proper keyed record for seeding.
			key := NewReminderKey("rem1")
			record := dal.NewRecordWithData(key, &dbo4reminders.ReminderDbo{})
			return tx.Set(ctx, record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		reminder, err := NewReminderDal().GetReminderByID(ctx, db, "rem1")
		if err != nil {
			t.Fatalf("GetReminderByID() returned error: %v", err)
		}
		if reminder.ID != "rem1" {
			t.Errorf("reminder.ID = %q, want rem1", reminder.ID)
		}
	})

	t.Run("returns_not_found_for_missing_reminder", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		_, err := NewReminderDal().GetReminderByID(ctx, db, "nosuchreminder")
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}

func TestReminderDalGae_DelayDiscardRemindersForTransfers_empty_slice(t *testing.T) {
	ctx := context.Background()
	// When transferIDs is empty the method logs a warning and returns nil
	// without touching the delayer — no delaying.Init needed.
	err := NewReminderDal().DelayDiscardRemindersForTransfers(ctx, []string{}, "rt1")
	if err != nil {
		t.Errorf("expected nil for empty transferIDs, got: %v", err)
	}
}

func TestQueueSendReminder_noop_when_due_in_exceeds_threshold(t *testing.T) {
	ctx := context.Background()
	// dueIn >= 3h — the function takes the else branch and returns nil without
	// calling CreateSendReminderTask.
	const threeHours = 3 * 60 * 60 * 1e9 // 3h in nanoseconds
	if err := QueueSendReminder(ctx, "rem1", threeHours); err != nil {
		t.Errorf("expected nil when dueIn >= 3h, got: %v", err)
	}
}

func TestCreateSendReminderTask(t *testing.T) {
	ctx := context.Background()

	t.Run("empty_reminder_id", func(t *testing.T) {
		if err := CreateSendReminderTask(ctx, ""); err == nil {
			t.Error("expected error for empty reminder ID, got nil")
		}
	})

	t.Run("delayer_not_initialized", func(t *testing.T) {
		original := DelayerSendReminder
		DelayerSendReminder = nil
		defer func() { DelayerSendReminder = original }()
		if err := CreateSendReminderTask(ctx, "rem1"); err == nil {
			t.Error("expected error when DelayerSendReminder is nil, got nil")
		}
	})

	t.Run("enqueues_work", func(t *testing.T) {
		original := DelayerSendReminder
		DelayerSendReminder = delaying.VoidWithLog("sendReminder", func(ctx context.Context, reminderID string) error {
			return nil
		})
		defer func() { DelayerSendReminder = original }()
		if err := CreateSendReminderTask(ctx, "rem1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := QueueSendReminder(ctx, "rem1", time.Hour); err != nil {
			t.Errorf("unexpected error from QueueSendReminder: %v", err)
		}
	})
}
