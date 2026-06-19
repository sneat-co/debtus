package dal4reminders

import (
	"errors"
	"time"
)

func ValidateSetReminderIsSentMessageIDs(messageIntID int64, messageStrID string, sentAt time.Time) error {
	if messageIntID != 0 && messageStrID != "" {
		return errors.New("messageIntID != 0 && messageStrID != ''")
	} else if messageIntID == 0 && messageStrID == "" {
		return errors.New("messageIntID == 0 && messageStrID == ''")
	}
	if sentAt.IsZero() {
		return errors.New("sentAt.IsZero()")
	}
	return nil
}
