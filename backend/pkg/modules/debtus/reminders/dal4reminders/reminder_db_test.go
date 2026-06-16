package dal4reminders

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
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

func getReminder(t *testing.T, db dal.DB, id string) dbo4reminders.Reminder {
	t.Helper()
	reminder := dbo4reminders.NewReminder(id, nil)
	if err := db.Get(context.Background(), reminder.Record); err != nil {
		t.Fatalf("failed to get reminder: %v", err)
	}
	return reminder
}

func TestGetReminderByID_and_SaveReminder(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusCreated})

	reminder, err := GetReminderByID(ctx, db, "rem1")
	if err != nil {
		t.Fatalf("GetReminderByID: %v", err)
	}
	if reminder.Data.Status != dbo4reminders.ReminderStatusCreated {
		t.Errorf("unexpected status %q", reminder.Data.Status)
	}

	if err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		reminder.Data.Status = dbo4reminders.ReminderStatusSending
		return SaveReminder(ctx, tx, reminder)
	}); err != nil {
		t.Fatalf("SaveReminder: %v", err)
	}
	if got := getReminder(t, db, "rem1").Data.Status; got != dbo4reminders.ReminderStatusSending {
		t.Errorf("status after save = %q, want sending", got)
	}
}

func TestSetReminderStatus(t *testing.T) {
	ctx := context.Background()
	when := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	statusCases := []struct {
		status  string
		checkDt func(data *dbo4reminders.ReminderDbo) time.Time
	}{
		{dbo4reminders.ReminderStatusDiscarded, func(d *dbo4reminders.ReminderDbo) time.Time { return d.DtDiscarded }},
		{dbo4reminders.ReminderStatusSent, func(d *dbo4reminders.ReminderDbo) time.Time { return d.DtSent }},
		{dbo4reminders.ReminderStatusViewed, func(d *dbo4reminders.ReminderDbo) time.Time { return d.DtViewed }},
		{dbo4reminders.ReminderStatusUsed, func(d *dbo4reminders.ReminderDbo) time.Time { return d.DtUsed }},
	}
	for _, tc := range statusCases {
		t.Run(tc.status, func(t *testing.T) {
			db := sneattesting.SetupMemoryDB(t)
			seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusCreated})
			reminder, err := SetReminderStatus(ctx, "rem1", "", tc.status, when)
			if err != nil {
				t.Fatalf("SetReminderStatus: %v", err)
			}
			if reminder.Data.Status != tc.status {
				t.Errorf("status = %q, want %q", reminder.Data.Status, tc.status)
			}
			stored := getReminder(t, db, "rem1")
			if stored.Data.Status != tc.status {
				t.Errorf("stored status = %q, want %q", stored.Data.Status, tc.status)
			}
			if got := tc.checkDt(stored.Data); !got.Equal(when) {
				t.Errorf("status timestamp = %v, want %v", got, when)
			}
		})
	}

	t.Run("sending_does_not_set_timestamp", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusCreated})
		if _, err := SetReminderStatus(ctx, "rem1", "", dbo4reminders.ReminderStatusSending, when); err != nil {
			t.Fatalf("SetReminderStatus: %v", err)
		}
		if got := getReminder(t, db, "rem1").Data.Status; got != dbo4reminders.ReminderStatusSending {
			t.Errorf("stored status = %q, want sending", got)
		}
	})

	t.Run("unsupported_status", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusCreated})
		if _, err := SetReminderStatus(ctx, "rem1", "", "bogus", when); err == nil {
			t.Error("expected error for unsupported status")
		}
	})

	t.Run("no_change_when_same_status", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusSent})
		if _, err := SetReminderStatus(ctx, "rem1", "", dbo4reminders.ReminderStatusSent, when); err != nil {
			t.Fatalf("SetReminderStatus: %v", err)
		}
	})

	t.Run("missing_reminder", func(t *testing.T) {
		_ = sneattesting.SetupMemoryDB(t)
		if _, err := SetReminderStatus(ctx, "nosuch", "", dbo4reminders.ReminderStatusSent, when); err == nil {
			t.Error("expected error for missing reminder")
		}
	})
}

func TestSetReminderIsSent(t *testing.T) {
	ctx := context.Background()
	sentAt := time.Date(2026, 6, 10, 13, 0, 0, 0, time.UTC)

	t.Run("happy_path", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		dtNext := time.Date(2026, 6, 10, 12, 30, 0, 0, time.UTC)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
			Status: dbo4reminders.ReminderStatusSending,
			DtNext: dtNext,
		})
		if err := SetReminderIsSent(ctx, "rem1", sentAt, 123, "", "en-US", ""); err != nil {
			t.Fatalf("SetReminderIsSent: %v", err)
		}
		stored := getReminder(t, db, "rem1")
		if stored.Data.Status != dbo4reminders.ReminderStatusSent {
			t.Errorf("status = %q, want sent", stored.Data.Status)
		}
		if !stored.Data.DtSent.Equal(sentAt) {
			t.Errorf("DtSent = %v, want %v", stored.Data.DtSent, sentAt)
		}
		if !stored.Data.DtScheduled.Equal(dtNext) {
			t.Errorf("DtScheduled = %v, want %v", stored.Data.DtScheduled, dtNext)
		}
		if !stored.Data.DtNext.IsZero() {
			t.Errorf("DtNext = %v, want zero", stored.Data.DtNext)
		}
		if stored.Data.MessageIntID != 123 {
			t.Errorf("MessageIntID = %v, want 123", stored.Data.MessageIntID)
		}
	})

	t.Run("message_str_id", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusSending})
		if err := SetReminderIsSent(ctx, "rem1", sentAt, 0, "msg-1", "en-US", ""); err != nil {
			t.Fatalf("SetReminderIsSent: %v", err)
		}
		if got := getReminder(t, db, "rem1").Data.MessageStrID; got != "msg-1" {
			t.Errorf("MessageStrID = %q, want msg-1", got)
		}
	})

	t.Run("wrong_status_is_noop", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{Status: dbo4reminders.ReminderStatusCreated})
		if err := SetReminderIsSent(ctx, "rem1", sentAt, 123, "", "en-US", ""); err != nil {
			t.Fatalf("SetReminderIsSent: %v", err)
		}
		if got := getReminder(t, db, "rem1").Data.Status; got != dbo4reminders.ReminderStatusCreated {
			t.Errorf("status = %q, want created (no-op)", got)
		}
	})

	t.Run("missing_reminder_is_noop", func(t *testing.T) {
		_ = sneattesting.SetupMemoryDB(t)
		if err := SetReminderIsSent(ctx, "nosuch", sentAt, 123, "", "en-US", ""); err != nil {
			t.Errorf("expected nil for missing reminder, got %v", err)
		}
	})

	t.Run("invalid_message_ids", func(t *testing.T) {
		_ = sneattesting.SetupMemoryDB(t)
		if err := SetReminderIsSent(ctx, "rem1", sentAt, 0, "", "en-US", ""); err == nil {
			t.Error("expected validation error when both message IDs are empty")
		}
	})
}

func TestRescheduleReminder_NotImplemented(t *testing.T) {
	_, _, err := RescheduleReminder(context.Background(), "rem1", time.Hour)
	if err == nil {
		t.Error("expected not-implemented error")
	}
}

func TestGetDueReminderIDs_EmptyDB(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	ids, err := GetDueReminderIDs(context.Background(), db)
	if err != nil {
		t.Fatalf("GetDueReminderIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected no due reminders, got %v", ids)
	}
}
