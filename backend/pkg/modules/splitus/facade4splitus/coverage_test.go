package facade4splitus

import (
	"context"
	"errors"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/decimal"
	"github.com/strongo/delaying"
)

// ---- errDelayer / voidDelayer test doubles ----

// errDelayer is a delaying.Delayer that always returns the given error.
type errDelayer struct {
	id  string
	err error
}

func (e errDelayer) ID() string          { return e.id }
func (e errDelayer) Implementation() any { return nil }
func (e errDelayer) EnqueueWork(_ context.Context, _ delaying.Params, _ ...any) error {
	return e.err
}
func (e errDelayer) EnqueueWorkMulti(_ context.Context, _ delaying.Params, _ ...[]any) error {
	return e.err
}

// voidDelayer is a delaying.Delayer that always succeeds (no-op).
type voidDelayer struct{ id string }

func (v voidDelayer) ID() string          { return v.id }
func (v voidDelayer) Implementation() any { return nil }
func (v voidDelayer) EnqueueWork(_ context.Context, _ delaying.Params, _ ...any) error {
	return nil
}
func (v voidDelayer) EnqueueWorkMulti(_ context.Context, _ delaying.Params, _ ...[]any) error {
	return nil
}

// initTestDelayers sets all the package-level delayer vars to the given delayer
// and returns a cleanup func that restores the originals.
func initTestDelayers(d delaying.Delayer) func() {
	origGroupWithBill := delayerUpdateGroupWithBill
	origBillDeps := delayerUpdateBillDependencies
	origUsersWithBill := delayerUpdateUsersWithBill
	origUserWithBill := delayerUpdateUserWithBill

	delayerUpdateGroupWithBill = d
	delayerUpdateBillDependencies = d
	delayerUpdateUsersWithBill = d
	delayerUpdateUserWithBill = d

	return func() {
		delayerUpdateGroupWithBill = origGroupWithBill
		delayerUpdateBillDependencies = origBillDeps
		delayerUpdateUsersWithBill = origUsersWithBill
		delayerUpdateUserWithBill = origUserWithBill
	}
}

// ---- SplitMemberTotal.Balance ----

func TestSplitMemberTotal_Balance(t *testing.T) {
	smt := SplitMemberTotal{briefs4splitus.BillMemberBalance{Paid: 100, Owes: 60}}
	if got := smt.Balance(); got != 40 {
		t.Errorf("Balance() = %v, want 40", got)
	}
}

// ---- DelayUpdateGroupWithBill ----

func TestDelayUpdateGroupWithBill_Success(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	if err := DelayUpdateGroupWithBill(ctx, "g1", "b1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayUpdateGroupWithBill_Error(t *testing.T) {
	wantErr := errors.New("enqueue error")
	restore := initTestDelayers(errDelayer{id: "test", err: wantErr})
	defer restore()

	ctx := context.Background()
	err := DelayUpdateGroupWithBill(ctx, "g1", "b1")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- DelayUpdateBillDependencies ----

func TestDelayUpdateBillDependencies_Success(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	if err := DelayUpdateBillDependencies(ctx, "b1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayUpdateBillDependencies_Error(t *testing.T) {
	wantErr := errors.New("enqueue error")
	restore := initTestDelayers(errDelayer{id: "test", err: wantErr})
	defer restore()

	ctx := context.Background()
	err := DelayUpdateBillDependencies(ctx, "b1")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- DelayUpdateSpaceWithBill ----

func TestDelayUpdateSpaceWithBill_Success(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	if err := DelayUpdateSpaceWithBill(ctx, "u1", "b1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayUpdateSpaceWithBill_Error(t *testing.T) {
	wantErr := errors.New("enqueue error")
	restore := initTestDelayers(errDelayer{id: "test", err: wantErr})
	defer restore()

	ctx := context.Background()
	err := DelayUpdateSpaceWithBill(ctx, "u1", "b1")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- delayedUpdateBillDependencies ----

func TestDelayedUpdateBillDependencies_BillNotFound(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	_ = db

	// Bill does not exist — should return nil (warning logged, err cleared)
	err := delayedUpdateBillDependencies(ctx, "nonexistent-bill-id")
	if err != nil {
		t.Errorf("expected nil for not-found bill, got: %v", err)
	}
}

func TestDelayedUpdateBillDependencies_BillWithGroup(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 50},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2",
			ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
				"u1": briefs4splitus.MemberContactBrief{ContactID: "c2"},
			}}, Owes: 50},
	}

	bill := createBillInDB(t, ctx, db, billEntity)

	err := delayedUpdateBillDependencies(ctx, bill.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- delayedUpdateUsersWithBill ----

func TestDelayedUpdateUsersWithBill(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	_ = db

	err := delayedUpdateUsersWithBill(ctx, "b1", []string{"u1", "u2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Note: delayedUpdateUserWithBill spawns a goroutine that calls GetBillByID with
// nil tx (requiring facade.GetSneatDB) concurrently with RunUserExtWorker which holds
// the dalgo2memory RWMutex. The goroutine deadlocks trying to acquire the same lock,
// becomes orphaned when t.Cleanup restores facade.GetSneatDB, and panics in subsequent
// tests. These paths are documented in TEST-COVERAGE.md as gaps.

// ---- SaveBill - DelayUpdateUsersWithBill error path ----

func TestSaveBill_DelayError(t *testing.T) {
	wantErr := errors.New("enqueue error")
	// Only set the usersWithBill delayer to error; others stay void.
	origUsersWithBill := delayerUpdateUsersWithBill
	delayerUpdateUsersWithBill = errDelayer{id: "test", err: wantErr}
	defer func() { delayerUpdateUsersWithBill = origUsersWithBill }()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return SaveBill(ctx, tx, bill)
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- GetBillMembersUserInfo ----
// GetBillMembersUserInfo has a production bug: billMembersUserInfo is never
// pre-allocated so `billMembersUserInfo[i] = ...` always panics on success path.
// The error-return path (missing contact) is the only reachable branch.

func TestGetBillMembersUserInfo_MissingContact(t *testing.T) {
	ctx := context.Background()
	bill := models4splitus.BillEntry{}
	bill.Data = &models4splitus.BillDbo{
		BillCommon: models4splitus.BillCommon{
			CreatorUserID: "1",
			SplitMode:     models4splitus.SplitModeEqually,
			AmountTotal:   100,
		},
	}
	members := []*briefs4splitus.BillMemberBrief{
		{
			MemberBrief: briefs4splitus.MemberBrief{
				ID:   "m1",
				Name: "User1",
				// No ContactByUser entry for user "2"
			},
			Owes: 100,
		},
	}
	if err := bill.Data.SetBillMembers(members); err != nil {
		t.Fatal(err)
	}

	_, err := GetBillMembersUserInfo(ctx, bill, 2)
	if err == nil {
		t.Error("expected error for missing contact")
	}
}

// ---- NewGroupMemberDalGae ----

func TestNewGroupMemberDalGae(t *testing.T) {
	d := NewGroupMemberDalGae()
	_ = d
}

// ---- GroupMemberDalGae.CreateGroupMember / GetGroupMemberByID ----
// CreateGroupMember and GetGroupMemberByID cannot be tested against dalgo2memory
// because dalgo2memory assigns integer auto-IDs as `int` while the production code
// type-asserts the key ID to `int64`, causing a panic. Documented in TEST-COVERAGE.md.

func TestGetGroupMemberByID_WithTx(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed a group member record directly (bypassing CreateGroupMember to avoid the
	// int vs int64 type-assertion panic in dalgo2memory).
	const memberID int64 = 42
	gm := models4splitus.NewGroupMember(memberID, &models4splitus.GroupMemberData{Name: "test"})
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, gm.Record)
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	// GetGroupMemberByID with a non-nil tx
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		d := NewGroupMemberDalGae()
		got, err := d.GetGroupMemberByID(ctx, tx, memberID)
		if err != nil {
			return err
		}
		if got.ID != memberID {
			t.Errorf("expected ID %d, got %d", memberID, got.ID)
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
}

func TestGetGroupMemberByID_WithNilTx(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	const memberID int64 = 99
	gm := models4splitus.NewGroupMember(memberID, &models4splitus.GroupMemberData{Name: "nil-tx-test"})
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, gm.Record)
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	// GetGroupMemberByID with nil tx → uses facade.GetSneatDB
	d := NewGroupMemberDalGae()
	got, err := d.GetGroupMemberByID(ctx, nil, memberID)
	if err != nil {
		t.Fatalf("GetGroupMemberByID with nil tx failed: %v", err)
	}
	if got.ID != memberID {
		t.Errorf("expected ID %d, got %d", memberID, got.ID)
	}
}

// ---- AssignBillToGroup ----

func seedEmptySplitusSpace(t *testing.T, ctx context.Context, db dal.DB) {
	t.Helper()
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed empty splitusSpace: %v", err)
	}
}

// TestAssignBillToGroup_EmptyMembers_NoGroupMembers: bill has no members and the
// space has no members either — AssignBillToGroup reads splitusSpace (finds empty)
// and returns without modifying members.
func TestAssignBillToGroup_EmptyMembers_NoGroupMembers(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedEmptySplitusSpace(t, ctx, db)

	billEntity := newMinimalBillEntity("u1")
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		_, _, err = AssignBillToGroup(ctx, tx, models4splitus.BillEntry{
			Data: billEntity,
		}, coretypes.SpaceID(spaceID), "u1")
		return
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
}

// TestAssignBillToGroup_NonEmptyMembers: bill already has members so AssignBillToGroup
// skips the splitusSpace.Get entirely.
func TestAssignBillToGroup_NonEmptyMembers(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 100},
	}
	bill := createBillInDB(t, ctx, db, billEntity)

	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, bill, coretypes.SpaceID(spaceID), "u1")
		return err
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
}

// ---- AddBillMember ----

func TestAddBillMember_PaidNegative(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntry := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", billEntry, "m1", "u2", "User2", decimal.Decimal64p2(-1))
		return err
	})
	if err == nil {
		t.Error("expected error for negative paid")
	}
}

func TestAddBillMember_EmptyBillID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	emptyBill := models4splitus.BillEntry{}
	emptyBill.Data = new(models4splitus.BillDbo)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", emptyBill, "m1", "u2", "User2", 0)
		return err
	})
	if err == nil {
		t.Error("expected error for empty bill ID")
	}
}

func TestAddBillMember_PanicNilTx(t *testing.T) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil tx")
		}
	}()
	bill := models4splitus.BillEntry{Data: new(models4splitus.BillDbo)}
	bill.ID = "b1"
	_, _, _, _, _ = AddBillMember(ctx, nil, "u1", bill, "m1", "u2", "User2", 0)
}

// ---- DeleteBill errors ----

func TestDeleteBill_SettledBill(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.Status = models4splitus.BillStatusSettled
	bill := createBillInDB(t, ctx, db, billEntity)

	_, err := DeleteBill(ctx, bill.ID, "u1")
	if !errors.Is(err, ErrSettledBillsCanNotBeDeleted) {
		t.Errorf("expected ErrSettledBillsCanNotBeDeleted, got: %v", err)
	}
}

// ---- RestoreBill - multi-member outstanding path ----

func TestRestoreBill_WithSpaceID_MultiMember(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	// m1 is the sole payer (Paid == AmountTotal=100); m2 owes 50.
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 50},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2",
			ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
				"u1": briefs4splitus.MemberContactBrief{ContactID: "c2"},
			}}, Owes: 50},
	}
	bill := createBillInDB(t, ctx, db, billEntity)

	// DeleteBill reads splitusSpace when bill has a spaceID — seed it.
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitusSpace: %v", err)
	}

	// Delete first
	deleted, err := DeleteBill(ctx, bill.ID, "u1")
	if err != nil {
		t.Fatalf("DeleteBill failed: %v", err)
	}
	if deleted.Data.Status != models4splitus.BillStatusDeleted {
		t.Fatalf("expected deleted status")
	}

	// Restore - multi-member bill should become outstanding
	restored, err := RestoreBill(ctx, bill.ID, "u1")
	if err != nil {
		t.Fatalf("RestoreBill failed: %v", err)
	}
	if restored.Data.Status != models4splitus.BillStatusOutstanding {
		t.Errorf("expected outstanding, got %q", restored.Data.Status)
	}
}

// ---- InsertBillEntity nil panic ----

func TestInsertBillEntity_NilPanic(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil billEntity")
		}
	}()

	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _ = InsertBillEntity(ctx, tx, nil)
		return nil
	})
}

// ---- GetBillByID with nil tx ----

func TestGetBillByID_NilTx(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	retrieved, err := GetBillByID(ctx, nil, bill.ID)
	if err != nil {
		t.Fatalf("GetBillByID with nil tx failed: %v", err)
	}
	if retrieved.ID != bill.ID {
		t.Errorf("expected ID %q, got %q", bill.ID, retrieved.ID)
	}
}

// ---- AddBillMember with splitusSpace in db ----
// NOTE: AddBillMember calls BillCommon.AddOrGetMember which has a production bug:
// when adding a new member it panics at `index != len(billMembers)-1` because
// `billMembers` (named return) is never assigned and has len 0. Any test that
// adds a truly new member will panic. Documented in TEST-COVERAGE.md as a gap.

// ---- delayedUpdateGroupWithBill ----

func TestDelayedUpdateGroupWithBill(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	bill := createBillInDB(t, ctx, db, billEntity)

	// Create the splitusSpace
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitusSpace: %v", err)
	}

	// First call - changed=true path
	err := delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), bill.ID)
	if err != nil {
		t.Fatalf("delayedUpdateGroupWithBill failed: %v", err)
	}

	// Second call — bill already in space, changed=false path
	err = delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), bill.ID)
	if err != nil {
		t.Fatalf("second delayedUpdateGroupWithBill failed: %v", err)
	}
}

// ---- AddBillMember paid paths ----
// AddBillMember calls BillCommon.AddOrGetMember which has a production bug:
// `billMembers` (named return) is never assigned, so any call that adds a *new*
// member panics at `index != len(billMembers)-1`. Tests for the paid==total and
// paid>total paths would both require a new member → untestable. Documented in
// TEST-COVERAGE.md.

// ---- AssignBillToGroup with existing group members ----
// NOTE: The currency branch (`bill.Data.Currency != ""`) inside AssignBillToGroup
// calls splitusSpace.Data.ApplyBillBalanceDifference which is a production stub
// returning "not implemented yet". Covered only the no-currency path here.
// Documented in TEST-COVERAGE.md.

func TestAssignBillToGroup_EmptyMembers_WithGroupMembers(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed a splitusSpace with members
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	splitusSpace.Data.Members = []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}},
	}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitusSpace: %v", err)
	}

	billEntity := newMinimalBillEntity("u1")
	// Clear currency to avoid the ApplyBillBalanceDifference stub (not implemented yet).
	billEntity.Currency = money.CurrencyCode("")
	billEntity.AmountTotal = 100

	inBill := models4splitus.BillEntry{Data: billEntity}

	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, inBill, coretypes.SpaceID(spaceID), "u1")
		return err
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
}

// TestAssignBillToGroup_AlreadyAssignedToAnotherGroup covers the error path where
// the bill is already assigned to a different group.
func TestAssignBillToGroup_AlreadyAssignedToAnotherGroup(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID("otherSpace") // pre-assigned to different space
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 100},
	}
	bill := createBillInDB(t, ctx, db, billEntity)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, bill, coretypes.SpaceID(spaceID), "u1")
		return err
	})
	if err == nil {
		t.Error("expected error when assigning bill already assigned to another group")
	}
}

// ---- InsertBillEntity error paths ----

func TestInsertBillEntity_EmptyCreatorUserID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := &models4splitus.BillDbo{}
	// CreatorUserID is empty — should return error
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, err := InsertBillEntity(ctx, tx, billEntity)
		return err
	})
	if err == nil {
		t.Error("expected error for empty CreatorUserID")
	}
}

func TestInsertBillEntity_ZeroAmount(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := &models4splitus.BillDbo{}
	billEntity.CreatorUserID = "u1"
	// AmountTotal is 0 — should return error
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, err := InsertBillEntity(ctx, tx, billEntity)
		return err
	})
	if err == nil {
		t.Error("expected error for zero AmountTotal")
	}
}

func TestInsertBillEntity_Success(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	var bill models4splitus.BillEntry
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, err = InsertBillEntity(ctx, tx, billEntity)
		return
	})
	if err != nil {
		t.Fatalf("InsertBillEntity failed: %v", err)
	}
	if bill.ID == "" {
		t.Error("expected non-empty bill ID")
	}
}

// ---- GetBillByID facade.GetSneatDB uninitialized path ----
// When facade.GetSneatDB is not initialized it panics — this path is reachable
// only in misconfigured environments. Documented in TEST-COVERAGE.md as a gap.

// ---- delayedUpdateBillDependencies: DelayUpdateGroupWithBill error path ----

func TestDelayedUpdateBillDependencies_DelayGroupError(t *testing.T) {
	wantErr := errors.New("group delay error")
	// Only make groupWithBill delayer fail; others are void.
	origGroupWithBill := delayerUpdateGroupWithBill
	delayerUpdateGroupWithBill = errDelayer{id: "test", err: wantErr}
	defer func() { delayerUpdateGroupWithBill = origGroupWithBill }()

	// Set others to void so they don't interfere.
	origBillDeps := delayerUpdateBillDependencies
	origUsersWithBill := delayerUpdateUsersWithBill
	origUserWithBill := delayerUpdateUserWithBill
	delayerUpdateBillDependencies = voidDelayer{id: "test"}
	delayerUpdateUsersWithBill = voidDelayer{id: "test"}
	delayerUpdateUserWithBill = voidDelayer{id: "test"}
	defer func() {
		delayerUpdateBillDependencies = origBillDeps
		delayerUpdateUsersWithBill = origUsersWithBill
		delayerUpdateUserWithBill = origUserWithBill
	}()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Create a bill with a spaceID so the userGroupID is set.
	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	bill := createBillInDB(t, ctx, db, billEntity)

	err := delayedUpdateBillDependencies(ctx, bill.ID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}
