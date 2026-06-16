package facade4splitusbot

import (
	"context"
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/internal/testutil"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"go.uber.org/mock/gomock"
)

// mockGetSneatDB replaces facade.GetSneatDB with a mock that calls the
// transaction function exactly once with a fresh MockReadwriteTransaction.
func mockGetSneatDB(t *testing.T) *mock_dal.MockReadwriteTransaction {
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
	return mockTx
}

// origGetSneatDB holds the original (panic) implementation so we can restore it.
var origGetSneatDB = facade.GetSneatDB

// ---------- CreateGroup ----------

func TestCreateGroup(t *testing.T) {
	mockTx := mockGetSneatDB(t)
	_ = mockTx // tx func returns error before any tx calls

	ctx := context.Background()
	_, _, err := CreateGroup(ctx, &models4splitus.GroupDbo{}, "tg", nil, nil)
	if err == nil {
		t.Fatal("expected an error from CreateGroup stub")
	}
	if !errors.Is(err, err) { // just ensure non-nil
		t.Fatal("unexpected nil error")
	}
}

// ---------- AddUsersToTheGroupAndOutstandingBills ----------

func TestAddUsersToTheGroupAndOutstandingBills_emptyUsers(t *testing.T) {
	ctx := context.Background()
	// Should return an error immediately, no DB access.
	_, _, err := AddUsersToTheGroupAndOutstandingBills(ctx, "space1", nil)
	if err == nil {
		t.Fatal("expected error for empty newUsers")
	}
}

func TestAddUsersToTheGroupAndOutstandingBills_withUsers_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// The tx will call tx.Get on the splitusSpace record (not found → empty space, no change).
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(dal.ErrRecordNotFound)
		return dal.ErrRecordNotFound
	})
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	newUsers := []NewUser{{Name: "Alice"}}
	_, _, err := AddUsersToTheGroupAndOutstandingBills(ctx, "space1", newUsers)
	if err != nil && !errors.Is(err, dal.ErrRecordNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddUsersToTheGroupAndOutstandingBills_memberAdded(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origGroupDelayer := facade4splitus.DelayerUpdateGroupUsers
	facade4splitus.DelayerUpdateGroupUsers = stub
	defer func() { facade4splitus.DelayerUpdateGroupUsers = origGroupDelayer }()

	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// tx.Get succeeds with no existing members, so newUser is added (isChanged=true).
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(nil) // record exists, empty members
		return nil
	})
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	// NewUser needs a non-nil PlatformUserData with a non-empty AppUserID so
	// AddOrGetMember doesn't panic.
	userData := &botsfwmodels.PlatformUserBaseDbo{}
	userData.AppUserID = "user-alice"
	newUsers := []NewUser{{Name: "Alice", PlatformUserData: userData}}
	_, _, err := AddUsersToTheGroupAndOutstandingBills(ctx, "space1", newUsers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- DelayUpdateGroupUsers ----------

func TestDelayUpdateGroupUsers(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origDelayer := facade4splitus.DelayerUpdateGroupUsers
	facade4splitus.DelayerUpdateGroupUsers = stub
	defer func() { facade4splitus.DelayerUpdateGroupUsers = origDelayer }()

	ctx := context.Background()
	err := DelayUpdateGroupUsers(ctx, "group1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.Calls()) != 1 {
		t.Fatalf("expected 1 enqueued call, got %d", len(stub.Calls()))
	}
}

func TestDelayUpdateGroupUsers_emptyGroupID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty groupID")
		}
	}()
	_ = DelayUpdateGroupUsers(context.Background(), "")
}

func TestAddUsersToTheGroupAndOutstandingBills_delayError(t *testing.T) {
	// Cover the `return err` branch when DelayUpdateGroupUsers returns an error.
	stub := &testutil.StubDelayer{Err: errors.New("delay error")}
	origGroupDelayer := facade4splitus.DelayerUpdateGroupUsers
	facade4splitus.DelayerUpdateGroupUsers = stub
	defer func() { facade4splitus.DelayerUpdateGroupUsers = origGroupDelayer }()

	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(nil)
		return nil
	})
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	userData := &botsfwmodels.PlatformUserBaseDbo{}
	userData.AppUserID = "user-bob"
	newUsers := []NewUser{{Name: "Bob", PlatformUserData: userData}}
	_, _, err := AddUsersToTheGroupAndOutstandingBills(ctx, "space1", newUsers)
	if err == nil {
		t.Fatal("expected error from delay")
	}
}

// ---------- delayedUpdateGroupUsers ----------

func Test_delayedUpdateGroupUsers_emptySpaceID(t *testing.T) {
	ctx := context.Background()
	err := delayedUpdateGroupUsers(ctx, "")
	if err != nil {
		t.Fatalf("expected nil for empty spaceID, got %v", err)
	}
}

func Test_delayedUpdateGroupUsers_withSpaceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// tx.Get returns not-found so no members to iterate.
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(dal.ErrRecordNotFound)
		return dal.ErrRecordNotFound
	})
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	err := delayedUpdateGroupUsers(ctx, "space1")
	if err != nil && !errors.Is(err, dal.ErrRecordNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- delayedUpdateUserWithGroups ----------

func Test_delayedUpdateUserWithGroups(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// tx.GetMulti is called with an empty slice (groupIDs2add is nil).
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	err := delayedUpdateUserWithGroups(ctx, "user1", nil, nil)
	if err == nil {
		t.Fatal("expected 'not implemented' error")
	}
}

// ---------- UpdateUserWithGroups ----------

func TestUpdateUserWithGroups(t *testing.T) {
	ctx := context.Background()
	err := UpdateUserWithGroups(ctx, nil, dbo4userus.UserEntry{}, nil, nil)
	if err == nil {
		t.Fatal("expected 'not implemented' error")
	}
}

// ---------- DelayUpdateContactWithGroups ----------

func TestDelayUpdateContactWithGroups(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origDelayer := facade4splitus.DelayerUpdateContactWithGroups
	facade4splitus.DelayerUpdateContactWithGroups = stub
	defer func() { facade4splitus.DelayerUpdateContactWithGroups = origDelayer }()

	ctx := context.Background()
	err := DelayUpdateContactWithGroups(ctx, "contact1", []string{"g1"}, []string{"g2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.Calls()) != 1 {
		t.Fatalf("expected 1 enqueued call, got %d", len(stub.Calls()))
	}
}

// ---------- delayedUpdateContactWithGroup ----------

func Test_delayedUpdateContactWithGroup(t *testing.T) {
	mockTx := mockGetSneatDB(t)
	_ = mockTx // UpdateContactWithGroups is called inside tx; it returns error immediately, no tx calls.

	ctx := context.Background()
	err := delayedUpdateContactWithGroup(ctx, "contact1", []string{"g1"}, nil)
	if err == nil {
		t.Fatal("expected 'UpdateContactWithGroups not implemented' error")
	}
}

// ---------- UpdateContactWithGroups ----------

func TestUpdateContactWithGroups(t *testing.T) {
	ctx := context.Background()
	err := UpdateContactWithGroups(ctx, "contact1", []string{"g1"}, nil)
	if err == nil {
		t.Fatal("expected 'UpdateContactWithGroups not implemented' error")
	}
}

// ---------- delayUpdateUserWithGroups (delay_update_user_with_group.go) ----------

func Test_delayUpdateUserWithGroups(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origDelayer := DelayerUpdateUserWithGroups
	DelayerUpdateUserWithGroups = stub
	defer func() { DelayerUpdateUserWithGroups = origDelayer }()

	ctx := context.Background()
	err := delayUpdateUserWithGroups(ctx, "user1", []string{"g1"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.Calls()) != 1 {
		t.Fatalf("expected 1 enqueued call, got %d", len(stub.Calls()))
	}
}

// ---------- AddUsersToTheGroupAndOutstandingBills — tx.Set error path ----------

func TestAddUsersToTheGroupAndOutstandingBills_setError(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origGroupDelayer := facade4splitus.DelayerUpdateGroupUsers
	facade4splitus.DelayerUpdateGroupUsers = stub
	defer func() { facade4splitus.DelayerUpdateGroupUsers = origGroupDelayer }()

	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(nil)
		return nil
	})
	setErr := errors.New("set error")
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(setErr)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	userData := &botsfwmodels.PlatformUserBaseDbo{}
	userData.AppUserID = "user-carol"
	newUsers := []NewUser{{Name: "Carol", PlatformUserData: userData}}
	_, _, err := AddUsersToTheGroupAndOutstandingBills(ctx, "space1", newUsers)
	if !errors.Is(err, setErr) {
		t.Fatalf("expected set error, got %v", err)
	}
}

// ---------- delayedUpdateGroupUsers — with members ----------

func Test_delayedUpdateGroupUsers_withMembers(t *testing.T) {
	stub := &testutil.StubDelayer{}
	origDelayer := DelayerUpdateUserWithGroups
	DelayerUpdateUserWithGroups = stub
	defer func() { DelayerUpdateUserWithGroups = origDelayer }()

	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// tx.Get succeeds and populates the SplitusSpaceDbo with one member having a UserID.
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(nil)
		// Populate members via the data pointer.
		data := r.Data().(*models4splitus.SplitusSpaceDbo)
		data.Members = []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{UserID: "user-x"}},
		}
		return nil
	})
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	// delayUpdateUserWithGroups will be called; stub records it and returns nil.
	err := delayedUpdateGroupUsers(ctx, "space1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_delayedUpdateGroupUsers_delayUserError(t *testing.T) {
	// Cover the `return err` branch when delayUpdateUserWithGroups returns an error.
	delayErr := errors.New("delay user error")
	stub := &testutil.StubDelayer{Err: delayErr}
	origDelayer := DelayerUpdateUserWithGroups
	DelayerUpdateUserWithGroups = stub
	defer func() { DelayerUpdateUserWithGroups = origDelayer }()

	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r dal.Record) error {
		r.SetError(nil)
		data := r.Data().(*models4splitus.SplitusSpaceDbo)
		data.Members = []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{UserID: "user-y"}},
		}
		return nil
	})
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	err := delayedUpdateGroupUsers(ctx, "space1")
	if !errors.Is(err, delayErr) {
		t.Fatalf("expected delay error, got %v", err)
	}
}

// ---------- delayedUpdateUserWithGroups — with groupIDs2add ----------

func Test_delayedUpdateUserWithGroups_withGroups(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	// tx.GetMulti is called with the group records slice.
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, records []dal.Record) error {
			// Mark records as existing so group.Record.Error() == nil path is taken.
			for _, r := range records {
				r.SetError(nil)
			}
			return nil
		})
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	// Returns "not implemented" after iterating over groups2add (all with no record error).
	err := delayedUpdateUserWithGroups(ctx, "user1", []string{"g1"}, nil)
	if err == nil {
		t.Fatal("expected 'not implemented' error")
	}
}

func Test_delayedUpdateUserWithGroups_getMultiError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockDB := mock_dal.NewMockDB(ctrl)
	getMultiErr := errors.New("getmulti error")
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(getMultiErr)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		})
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	defer func() { facade.GetSneatDB = origGetSneatDB }()

	ctx := context.Background()
	err := delayedUpdateUserWithGroups(ctx, "user1", []string{"g1"}, nil)
	if !errors.Is(err, getMultiErr) {
		t.Fatalf("expected getmulti error, got %v", err)
	}
}

// NOTE: the `group.Record.Error()` branch inside delayedUpdateUserWithGroups is
// structurally unreachable: the production code passes a nil splitusSpaceRecords
// slice to tx.GetMulti (bug), so groups2add[i].Record is never fetched and its
// Error() always returns nil. Documented in TEST-COVERAGE.md as a gap.
