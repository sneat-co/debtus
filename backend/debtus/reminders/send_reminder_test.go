package reminders

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func seedReminder(t *testing.T, db dal.DB, id string, data *dbo4reminders.ReminderDbo) {
	t.Helper()
	ctx := context.Background()
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, dbo4reminders.NewReminder(id, data).Record)
	}); err != nil {
		t.Fatalf("failed to seed reminder: %v", err)
	}
}

func TestSendReminder_EmptyID(t *testing.T) {
	if err := sendReminder(context.Background(), ""); err == nil {
		t.Error("expected error for empty reminder ID")
	}
}

func TestSendReminder_StatusNotCreated(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusSent})
	if err := sendReminder(context.Background(), "rem1"); err != nil {
		t.Errorf("expected nil for already sent reminder, got: %v", err)
	}
}

func TestSendReminder_TransferNotFound_MarksReminderInvalid(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "no-such-transfer",
		DtNext:   time.Now(),
	})
	if err := sendReminder(ctx, "rem1"); err != nil {
		t.Fatalf("expected nil (reminder marked invalid), got: %v", err)
	}
	reminder := dbo4reminders.NewReminder("rem1", nil)
	if err := db.Get(ctx, reminder.Record); err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if reminder.Data.Status != dbo4reminders.ReminderStatusInvalidNoTransfer {
		t.Errorf("status = %q, want %q", reminder.Data.Status, dbo4reminders.ReminderStatusInvalidNoTransfer)
	}
	if !reminder.Data.DtNext.IsZero() {
		t.Errorf("DtNext = %v, want zero", reminder.Data.DtNext)
	}
}

func TestSendReminderHandler(t *testing.T) {
	t.Run("no_id", func(t *testing.T) {
		_ = sneattesting.SetupMemoryDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/task-queue/send-reminder", strings.NewReader(""))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		SendReminderHandler(context.Background(), w, r)
		if w.Code != 200 {
			t.Errorf("expected 200 for missing id, got %d", w.Code)
		}
	})

	t.Run("missing_reminder", func(t *testing.T) {
		_ = sneattesting.SetupMemoryDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/task-queue/send-reminder", strings.NewReader("id=nosuch"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		SendReminderHandler(context.Background(), w, r)
		if w.Code != 200 {
			t.Errorf("expected 200 for not-found reminder, got %d", w.Code)
		}
	})
}

func TestCronSendReminders_NoDueReminders(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cron/send-reminders", nil)
	CronSendReminders(context.Background(), w, r)
	if w.Code != 200 {
		t.Errorf("expected 200 when no due reminders, got %d", w.Code)
	}
}
