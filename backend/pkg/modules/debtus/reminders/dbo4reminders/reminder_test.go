package dbo4reminders

import (
	"testing"
	"time"
)

func TestNewReminder(t *testing.T) {
	t.Run("nil_entity", func(t *testing.T) {
		reminder := NewReminder("rem1", nil)
		if reminder.ID != "rem1" {
			t.Errorf("reminder.ID = %q, want rem1", reminder.ID)
		}
		if reminder.Key == nil {
			t.Fatal("reminder.Key is nil")
		}
		if reminder.Key.Collection() != ReminderKind {
			t.Errorf("key collection = %q, want %q", reminder.Key.Collection(), ReminderKind)
		}
		if reminder.Data == nil {
			t.Error("reminder.Data is nil — expected nil-guarded data")
		}
		if reminder.Record == nil {
			t.Error("reminder.Record is nil")
		}
	})

	t.Run("with_entity", func(t *testing.T) {
		data := &ReminderDbo{Status: ReminderStatusCreated}
		reminder := NewReminder("rem2", data)
		if reminder.Data != data {
			t.Error("reminder.Data is not the provided entity")
		}
	})
}

func TestNewReminderViaTelegram(t *testing.T) {
	next := time.Now().Add(time.Hour)
	r := NewReminderViaTelegram("bot1", 42, "u1", "t1", true, next)
	if err := r.Validate(); err != nil {
		t.Errorf("expected valid reminder, got: %v", err)
	}
	if r.Status != ReminderStatusCreated || r.SentVia != "telegram" || r.BotID != "bot1" ||
		r.ChatIntID != 42 || r.UserID != "u1" || r.TargetID != "t1" || !r.IsAutomatic || !r.DtNext.Equal(next) {
		t.Errorf("unexpected reminder: %+v", r)
	}
}

func TestReminderDbo_Validate(t *testing.T) {
	valid := func() *ReminderDbo {
		return NewReminderViaTelegram("bot1", 42, "u1", "t1", false, time.Now().Add(time.Hour))
	}
	cases := []struct {
		name   string
		mutate func(r *ReminderDbo)
	}{
		{"unknown_status", func(r *ReminderDbo) { r.Status = "bogus" }},
		{"target_id_spaces", func(r *ReminderDbo) { r.TargetID = "t1 " }},
		{"target_kind_spaces", func(r *ReminderDbo) { r.TargetKind = "transfer " }},
		{"empty_sent_via", func(r *ReminderDbo) { r.SentVia = "" }},
		{"zero_dt_created", func(r *ReminderDbo) { r.DtCreated = time.Time{} }},
		{"sent_before_created", func(r *ReminderDbo) { r.DtSent = r.DtCreated.Add(-time.Hour) }},
		{"viewed_before_sent", func(r *ReminderDbo) {
			r.DtSent = r.DtCreated.Add(time.Hour)
			r.DtViewed = r.DtSent.Add(-time.Minute)
		}},
		{"chat_id_without_bot_id", func(r *ReminderDbo) { r.BotID = "" }},
		{"bot_id_without_chat_id", func(r *ReminderDbo) { r.ChatIntID = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := valid()
			tc.mutate(r)
			if err := r.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestReminderDbo_ScheduleNextReminder(t *testing.T) {
	r := NewReminderViaTelegram("bot1", 42, "u1", "t1", false, time.Now())
	r.DtSent = time.Now()
	r.MessageIntID = 7
	next := time.Now().Add(24 * time.Hour)
	scheduled := r.ScheduleNextReminder(5, next)
	if !r.IsRescheduled {
		t.Error("expected original reminder to be marked rescheduled")
	}
	if scheduled.ParentReminderID != 5 {
		t.Errorf("ParentReminderID = %d, want 5", scheduled.ParentReminderID)
	}
	if scheduled.Status != ReminderStatusCreated {
		t.Errorf("Status = %q, want created", scheduled.Status)
	}
	if !scheduled.DtNext.Equal(next) {
		t.Errorf("DtNext = %v, want %v", scheduled.DtNext, next)
	}
	if !scheduled.DtSent.IsZero() || scheduled.MessageIntID != 0 {
		t.Error("expected sent fields to be reset on scheduled reminder")
	}
}
