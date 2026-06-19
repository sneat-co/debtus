package delayed4debtus

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"go.uber.org/mock/gomock"
)

// TestDiscardReminder_GetMultiError_withReturnTransferID covers the GetMulti error
// path inside the returnTransferID > "" branch (reminder_delays.go:174).
func TestDiscardReminder_GetMultiError_withReturnTransferID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_ = sneattesting.SetupMemoryDB(t)

	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	// returnTransferID is non-empty → GetMulti is called with 3 records and fails.
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(context.DeadlineExceeded)

	err := discardReminder(context.Background(), mockTx, "rem1", "t1", "rt1")
	if err == nil {
		t.Error("expected error from GetMulti, got nil")
	}
}

// TestDiscardReminder_TelegramGetUserError covers the GetUser error path
// (reminder_delays.go:206) when reminder.Locale is empty and the user record is missing.
func TestDiscardReminder_TelegramGetUserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const (
		reminderID = "rem-getuser-err"
		transferID = "t-getuser-err"
		userID     = "u-missing"
	)

	reminderDbo := &dbo4reminders.ReminderDbo{
		SentVia:      "telegram",
		BotID:        "bot1",
		MessageIntID: 55,
		ChatIntID:    321,
		UserID:       userID,
		Locale:       "", // empty → dal4userus.GetUser is attempted
		Status:       dbo4reminders.ReminderStatusSent,
	}
	transferDbo := &models4debtus.TransferData{
		CreatorUserID: userID,
		FromJson:      `{"userID":"` + userID + `"}`,
		ToJson:        `{"userID":"u2"}`,
	}

	db := sneattesting.SetupMemoryDB(t)

	ctx := context.Background()
	// Seed reminder and transfer but NOT the user → GetUser returns not-found.
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		r := dbo4reminders.NewReminder(reminderID, reminderDbo)
		if err := tx.Set(ctx, r.Record); err != nil {
			return err
		}
		tr := models4debtus.NewTransfer(transferID, transferDbo)
		return tx.Set(ctx, tr.Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, records []dal.Record) error {
			return db.GetMulti(ctx, records)
		},
	).AnyTimes()

	stubGetTelegramBotApi(t, fakeBotAPI(fakeRoundTripper{}), nil)

	err := discardReminder(ctx, mockTx, reminderID, transferID, "")
	if err == nil {
		t.Error("expected error from dal4userus.GetUser (user not seeded), got nil")
	}
}
