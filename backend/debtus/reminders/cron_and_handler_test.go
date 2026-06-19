package reminders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-go-core/facade"
)

var origGetSneatDB = facade.GetSneatDB

func overrideGetSneatDBError(t *testing.T, err error) {
	t.Helper()
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return nil, err
	}
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })
}

func overrideGetDueReminderIDs(t *testing.T, ids []string, err error) {
	t.Helper()
	orig := getDueReminderIDs
	getDueReminderIDs = func(_ context.Context, _ dal.QueryExecutor) ([]string, error) {
		return ids, err
	}
	t.Cleanup(func() { getDueReminderIDs = orig })
}

func overrideCreateSendReminderTask(t *testing.T, err error) {
	t.Helper()
	orig := createSendReminderTask
	createSendReminderTask = func(_ context.Context, _ string) error {
		return err
	}
	t.Cleanup(func() { createSendReminderTask = orig })
}

func TestCronSendReminders_GetSneatDBError(t *testing.T) {
	overrideGetSneatDBError(t, errors.New("db error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/cron/send-reminders", nil)
	CronSendReminders(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestCronSendReminders_GetDueReminderIDsError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	overrideGetDueReminderIDs(t, nil, errors.New("query error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/cron/send-reminders", nil)
	CronSendReminders(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "query error") {
		t.Errorf("expected error in body, got: %q", w.Body.String())
	}
}

func TestCronSendReminders_CreateTaskError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	overrideGetDueReminderIDs(t, []string{"rem1"}, nil)
	overrideCreateSendReminderTask(t, errors.New("task error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/cron/send-reminders", nil)
	CronSendReminders(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestCronSendReminders_Success(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	overrideGetDueReminderIDs(t, []string{"rem1", "rem2"}, nil)
	overrideCreateSendReminderTask(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/cron/send-reminders", nil)
	CronSendReminders(context.Background(), w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestSendReminderHandler_ParseFormError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	w := httptest.NewRecorder()
	// %zz causes ParseForm to return an error
	r := httptest.NewRequest(http.MethodPost, "/task-queue/send-reminder?id=%zz", nil)
	SendReminderHandler(context.Background(), w, r)
	// Handler logs the error and returns without writing a status code (200 default)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSendReminder_GetSneatDBError(t *testing.T) {
	overrideGetSneatDBError(t, errors.New("db unavailable"))
	err := sendReminder(context.Background(), "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "db unavailable") {
		t.Errorf("expected db error, got: %v", err)
	}
}

func TestSendReminderHandler_NotFound_WritesStatus(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/task-queue/send-reminder",
		strings.NewReader("id=nosuch"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// sendReminder returns not-found error, handler should NOT write 500
	SendReminderHandler(context.Background(), w, r)
	if w.Code == http.StatusInternalServerError {
		t.Errorf("expected no 500 for not-found reminder, got %d", w.Code)
	}
}

func overrideGetTransferByID(t *testing.T, transfer models4debtus.TransferEntry, err error) {
	t.Helper()
	orig := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return transfer, err
	}
	t.Cleanup(func() { getTransferByID = orig })
}

func overrideDiscardReminder(t *testing.T, err error) {
	t.Helper()
	orig := discardReminder
	discardReminder = func(_ context.Context, _, _, _ string) error {
		return err
	}
	t.Cleanup(func() { discardReminder = orig })
}

func TestSendReminder_TransferLoadError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	wantErr := errors.New("network error")
	overrideGetTransferByID(t, models4debtus.TransferEntry{}, wantErr)
	err := sendReminder(context.Background(), "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("expected transfer load error, got: %v", err)
	}
}

func TestSendReminder_TransferNotOutstanding_DiscardSuccess(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	transfer := models4debtus.NewTransfer("txid", &models4debtus.TransferData{
		IsOutstanding: false,
	})
	overrideGetTransferByID(t, transfer, nil)
	overrideDiscardReminder(t, nil)
	err := sendReminder(context.Background(), "rem1")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestSendReminder_TransferNotOutstanding_DiscardError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	transfer := models4debtus.NewTransfer("txid", &models4debtus.TransferData{
		IsOutstanding: false,
	})
	overrideGetTransferByID(t, transfer, nil)
	overrideDiscardReminder(t, errors.New("discard failed"))
	err := sendReminder(context.Background(), "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendReminderHandler_NonNotFoundError_Writes500(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	overrideGetTransferByID(t, models4debtus.TransferEntry{}, errors.New("network error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/task-queue/send-reminder",
		strings.NewReader("id=rem1"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	SendReminderHandler(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-not-found error, got %d", w.Code)
	}
}
