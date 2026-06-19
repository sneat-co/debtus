package debtusdal

import (
	"testing"
)

func TestNewReminderKey(t *testing.T) {
	const reminderID = "135"
	testStrKey(t, reminderID, NewReminderKey(reminderID))
}
