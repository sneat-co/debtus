package reminders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

// fakeSent implements emails.Sent
type fakeSent struct{ id string }

func (f fakeSent) MessageID() string { return f.id }

// fakeEmailClient implements emails.Client
type fakeEmailClient struct {
	err  error
	sent emails.Sent
}

func (c *fakeEmailClient) Send(_ context.Context, _ emails.Email) (emails.Sent, error) {
	return c.sent, c.err
}

// overrideSendReminderToUser replaces sendReminderToUserFn with a stub.
func overrideSendReminderToUser(t *testing.T, err error) {
	t.Helper()
	orig := sendReminderToUserFn
	sendReminderToUserFn = func(_ context.Context, _ string, _ models4debtus.TransferEntry) error {
		return err
	}
	t.Cleanup(func() { sendReminderToUserFn = orig })
}

// newValidTransfer creates a TransferEntry with minimal valid JSON so Counterparty() doesn't panic.
func newValidTransfer(id string) models4debtus.TransferEntry {
	return models4debtus.NewTransfer(id, &models4debtus.TransferData{
		CreatorUserID: "u1",
		IsOutstanding: true,
		FromJson:      `{"userID":"u1","contactName":"Alice"}`,
		ToJson:        `{"userID":"u2","contactName":"Bob"}`,
	})
}

// newValidUser creates a UserEntry with non-nil Names so user.Data.Names.UserName doesn't panic.
func newValidUser(id string) dbo4userus.UserEntry {
	dbo := &dbo4userus.UserDbo{Email: "user@example.com"}
	dbo.Names = &person.NameFields{UserName: "Test User"}
	return dbo4userus.NewUserEntryWithDbo(id, dbo)
}

func TestSendReminder_OutstandingTransfer_SendSuccess(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	transfer := newValidTransfer("txid")
	overrideGetTransferByID(t, transfer, nil)
	overrideSendReminderToUser(t, nil)
	err := sendReminder(context.Background(), "rem1")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestSendReminder_OutstandingTransfer_SendError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "txid",
	})
	transfer := newValidTransfer("txid")
	overrideGetTransferByID(t, transfer, nil)
	overrideSendReminderToUser(t, errors.New("send failed"))
	// sendReminder logs the error but still returns nil
	err := sendReminder(context.Background(), "rem1")
	if err != nil {
		t.Errorf("expected nil (error is logged not returned), got: %v", err)
	}
}

func TestSendReminderByEmail_GetEmailClientError(t *testing.T) {
	origClient := emailing.GetEmailClient
	wantErr := errors.New("no email client")
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return nil, wantErr
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	transfer := newValidTransfer("txid")
	user := newValidUser("u1")
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
	})
	err := sendReminderByEmail(context.Background(), reminder, "user@example.com", transfer, user)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected email client error, got: %v", err)
	}
}

func TestSendReminderByEmail_SendSuccess(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)

	origClient := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{sent: fakeSent{id: "msg1"}}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	transfer := newValidTransfer("txid")
	user := newValidUser("u1")
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusSending,
		UserID: "u1",
	})
	// SetReminderIsSent will fail (no record seeded in DB), so DelaySetReminderIsSent is called.
	// The delayer was registered via InitDelaying in 1_test.go init().
	err := sendReminderByEmail(context.Background(), reminder, "user@example.com", transfer, user)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendReminderByEmail_SendError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)

	origClient := emailing.GetEmailClient
	sendErr := errors.New("smtp failure")
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{err: sendErr}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	transfer := newValidTransfer("txid")
	user := newValidUser("u1")
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusSending,
		UserID: "u1",
	})
	err := sendReminderByEmail(context.Background(), reminder, "user@example.com", transfer, user)
	// send error gets stored in errDetails; final err check returns it wrapped
	if err == nil {
		t.Fatal("expected error for smtp failure, got nil")
	}
}

// TestSendReminder_TransferNotFound_TxError covers the branch where RunReadwriteTransaction
// fails after a not-found transfer error (line 96: "failed to update reminder").
func TestSendReminder_TransferNotFound_TxError(t *testing.T) {
	ctx := context.Background()

	// Set up a mock DB: Get succeeds (returns a seeded reminder), but RunReadwriteTransaction fails.
	ctrl := gomock.NewController(t)
	wantTxErr := errors.New("tx error")

	memDB := sneattesting.SetupMemoryDB(t)
	// Seed reminder via memory DB
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "no-such-transfer",
	})

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return(wantTxErr).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	// getTransferByID returns a not-found error to trigger the branch
	orig := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return models4debtus.TransferEntry{}, dal.ErrRecordNotFound
	}
	t.Cleanup(func() { getTransferByID = orig })

	err := sendReminder(ctx, "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantTxErr) {
		t.Errorf("expected wrapped tx error, got: %v", err)
	}
}

// TestSendReminder_TransferNotFound_TxGetError covers the branch where GetReminderByID
// fails inside the "mark invalid" transaction (line 85-86 in taskqueu_handler.go).
func TestSendReminder_TransferNotFound_TxGetError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	wantGetErr := errors.New("get error in tx")

	memDB := sneattesting.SetupMemoryDB(t)
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "no-such-transfer",
	})

	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(wantGetErr).AnyTimes()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	orig := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return models4debtus.TransferEntry{}, dal.ErrRecordNotFound
	}
	t.Cleanup(func() { getTransferByID = orig })

	err := sendReminder(ctx, "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantGetErr) {
		t.Errorf("expected wrapped get error, got: %v", err)
	}
}

// TestSendReminder_TransferNotFound_TxSetError covers the branch where SaveReminder
// fails inside the "mark invalid" transaction (line 91-92 in taskqueu_handler.go).
func TestSendReminder_TransferNotFound_TxSetError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	wantSetErr := errors.New("set error in tx")

	memDB := sneattesting.SetupMemoryDB(t)
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusCreated,
		TargetID: "no-such-transfer",
	})

	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(wantSetErr).AnyTimes()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	orig := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return models4debtus.TransferEntry{}, dal.ErrRecordNotFound
	}
	t.Cleanup(func() { getTransferByID = orig })

	err := sendReminder(ctx, "rem1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantSetErr) {
		t.Errorf("expected wrapped set error, got: %v", err)
	}
}

func TestDelaySetChatIsForbidden(t *testing.T) {
	// delaySetChatIsForbidden is set in init() via InitDelaying(delaying.MustRegisterFunc)
	// using delaying.VoidWithLog which always succeeds.
	err := DelaySetChatIsForbidden(context.Background(), "bot1", 42, time.Now())
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestSetChatIsForbidden_LogsBeforePanic(t *testing.T) {
	// SetChatIsForbidden panics with "TODO: Implement...".
	// The lines before the panic (logus.Debugf and _ = botID) are covered here.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got nil")
		}
	}()
	_ = SetChatIsForbidden(context.Background(), "bot1", 42, time.Now())
}
