package dal4reminders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/dal-go/dalgo/record"
	"github.com/dal-go/dalgo/recordset"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"go.uber.org/mock/gomock"
)

// fakeQueryExecutor implements dal.QueryExecutor and returns a configured
// error from ExecuteQueryToRecordsReader.
type fakeQueryExecutor struct {
	dal.DB
	execErr error
	reader  dal.RecordsReader
}

func (f fakeQueryExecutor) ExecuteQueryToRecordsReader(_ context.Context, _ dal.Query) (dal.RecordsReader, error) {
	if f.execErr != nil {
		return nil, f.execErr
	}
	return f.reader, nil
}

func (f fakeQueryExecutor) ExecuteQueryToRecordsetReader(ctx context.Context, query dal.Query, options ...recordset.Option) (dal.RecordsetReader, error) {
	return f.DB.ExecuteQueryToRecordsetReader(ctx, query, options...)
}

// fakeRecordsReader returns an error from Next (to test SelectAllIDs error path).
type errorRecordsReader struct {
	nextErr error
}

func (r *errorRecordsReader) Cursor() (string, error)   { return "", nil }
func (r *errorRecordsReader) Close() error              { return nil }
func (r *errorRecordsReader) Next() (dal.Record, error) { return nil, r.nextErr }

// origGetSneatDB holds the original facade.GetSneatDB for restoration.
var origGetSneatDB = facade.GetSneatDB

func TestGetDueReminderIDs_ExecuteQueryError(t *testing.T) {
	wantErr := errors.New("query executor error")
	fakeDB := fakeQueryExecutor{
		DB:      dalgo2memory.NewDB(),
		execErr: wantErr,
	}
	_, err := GetDueReminderIDs(context.Background(), fakeDB)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped query error, got: %v", err)
	}
}

func TestGetDueReminderIDs_SelectAllIDsError(t *testing.T) {
	wantErr := errors.New("reader error")
	fakeDB := fakeQueryExecutor{
		DB:     dalgo2memory.NewDB(),
		reader: &errorRecordsReader{nextErr: wantErr},
	}
	_, err := GetDueReminderIDs(context.Background(), fakeDB)
	if err == nil {
		t.Fatal("expected error from SelectAllIDs, got nil")
	}
}

// mockDBWithTx creates a mock DB that calls the given worker function inside
// RunReadwriteTransaction.
func mockDBWithTx(t *testing.T) (*mock_dal.MockDB, *mock_dal.MockReadwriteTransaction) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		}).AnyTimes()
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return mockDB, nil
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origGetSneatDB
	})
	return mockDB, mockTx
}

// TestSetReminderIsSent_DBGetError covers the branch where Get returns a
// non-NotFound error inside the transaction.
func TestSetReminderIsSent_DBGetError(t *testing.T) {
	_, mockTx := mockDBWithTx(t)
	wantErr := errors.New("db get error")
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(wantErr).AnyTimes()

	sentAt := time.Now()
	err := SetReminderIsSent(context.Background(), "rem1", sentAt, 123, "", "en-US", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped DB error, got: %v", err)
	}
}

// nilDataReminder constructs a Reminder with a nil Data pointer, exercising the
// reminder.Data == nil branch in SetReminderIsSentInTransaction.
func nilDataReminder(id string) dbo4reminders.Reminder {
	key := dal.NewKeyWithID(dbo4reminders.ReminderKind, id)
	return dbo4reminders.Reminder{
		RecordWithID: record.WithID[string]{
			ID:     id,
			Key:    key,
			Record: dal.NewRecordWithData(key, new(dbo4reminders.ReminderDbo)),
		},
		Data: nil,
	}
}

// TestSetReminderIsSentInTransaction_NilData_NotFound covers the nil Data path
// where the re-fetched reminder is not found (returns nil).
func TestSetReminderIsSentInTransaction_NilData_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	// Simulate not-found
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(dal.ErrRecordNotFound).AnyTimes()

	reminder := nilDataReminder("rem1")
	sentAt := time.Now()
	err := SetReminderIsSentInTransaction(context.Background(), mockTx, reminder, sentAt, 123, "", "en-US", "")
	if err != nil {
		t.Errorf("expected nil for not-found reminder, got: %v", err)
	}
}

// TestSetReminderIsSentInTransaction_NilData_DBError covers the nil Data path
// where the re-fetch returns a non-NotFound error.
func TestSetReminderIsSentInTransaction_NilData_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	wantErr := errors.New("db fetch error")
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(wantErr).AnyTimes()

	reminder := nilDataReminder("rem1")
	sentAt := time.Now()
	err := SetReminderIsSentInTransaction(context.Background(), mockTx, reminder, sentAt, 123, "", "en-US", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped DB error, got: %v", err)
	}
}

// TestSetReminderIsSentInTransaction_TxSetError covers the tx.Set error path.
func TestSetReminderIsSentInTransaction_TxSetError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	// Seed a reminder with status=sending
	ctx := context.Background()
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
			Status: dbo4reminders.ReminderStatusSending,
		}).Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Use a mock tx that reads from memory DB but fails on Set.
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return db.Get(ctx, rec)
	}).AnyTimes()
	setErr := errors.New("set error")
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(setErr).AnyTimes()

	// Reminder with nil Data so GetReminderByID is called first.
	reminder := nilDataReminder("rem1")
	sentAt := time.Now()
	err := SetReminderIsSentInTransaction(ctx, mockTx, reminder, sentAt, 123, "", "en-US", "")
	if err == nil {
		t.Fatal("expected error from tx.Set, got nil")
	}
	if !errors.Is(err, setErr) {
		t.Errorf("expected wrapped set error, got: %v", err)
	}
}

// TestSetReminderStatus_TxSetError covers the tx.Set error path.
func TestSetReminderStatus_TxSetError(t *testing.T) {
	ctx := context.Background()
	when := time.Now()

	// We need a DB where the reminder GET works (returns a found reminder),
	// but the tx.Set fails.
	db := sneattesting.SetupMemoryDB(t)

	// Seed a reminder
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
			Status: dbo4reminders.ReminderStatusCreated,
		}).Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Create a mock DB that wraps the memory DB for Get but fails on Set
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, record dal.Record) error {
		return db.Get(ctx, record)
	}).AnyTimes()
	setErr := errors.New("set error")
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(setErr).AnyTimes()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return mockDB, nil
	}
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	_, err := SetReminderStatus(ctx, "rem1", "", dbo4reminders.ReminderStatusSent, when)
	if err == nil {
		t.Fatal("expected error from tx.Set, got nil")
	}
	if !errors.Is(err, setErr) {
		t.Errorf("expected wrapped set error, got: %v", err)
	}
}
