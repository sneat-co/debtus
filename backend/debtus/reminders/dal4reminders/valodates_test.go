package dal4reminders

import (
	"strings"
	"testing"
	"time"
)

func Test_ValidateSetReminderIsSentMessageIDs(t *testing.T) {
	var err error
	now := time.Now()
	if err = ValidateSetReminderIsSentMessageIDs(0, "", now); err == nil {
		t.Error("Should fail: _validateSetReminderIsSentMessageIDs(0, '')")
	}
	if err = ValidateSetReminderIsSentMessageIDs(1, "not empty", now); err == nil {
		t.Error("Should fail: _validateSetReminderIsSentMessageIDs(1, 'not empty')")
	}
	if err = ValidateSetReminderIsSentMessageIDs(1, "", time.Time{}); err == nil {
		t.Error("Should fail as sentAt is zero")
		if !strings.Contains(err.Error(), "sentAt.IsZero()") {
			t.Error("Error message does not contain 'sentAt.IsZero()'")
		}
	}
}
