package reminders

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dal4reminders"
	"github.com/strongo/logus"
)

// getDueReminderIDs is a seam so tests can inject a fake implementation.
var getDueReminderIDs = func(ctx context.Context, db dal.QueryExecutor) ([]string, error) {
	return dal4reminders.GetDueReminderIDs(ctx, db)
}

// createSendReminderTask is a seam so tests can inject a fake implementation.
var createSendReminderTask = func(ctx context.Context, reminderID string) error {
	return debtusdal.CreateSendReminderTask(ctx, reminderID)
}

func CronSendReminders(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	db, err := facade.GetSneatDB(ctx)
	if err != nil {
		logus.Errorf(ctx, "Failed to get database: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reminderIDs, err := getDueReminderIDs(ctx, db)
	if err != nil {
		logus.Errorf(ctx, "Failed to get due reminders: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if len(reminderIDs) == 0 {
		logus.Debugf(ctx, "No reminders to send")
		return
	}

	logus.Debugf(ctx, "Loaded %d reminder(s)", len(reminderIDs))

	for _, reminderID := range reminderIDs {
		if err = createSendReminderTask(ctx, reminderID); err != nil {
			logus.Errorf(ctx, "Failed to queue send-reminder task for reminder %s: %v", reminderID, err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "failed to queue send-reminder task for reminder %s: %v", reminderID, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
