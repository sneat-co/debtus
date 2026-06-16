package facade4splitus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/update"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/decimal"
)

// ---- CreateBill: SplitMode == "" (reachable only via the isValidBillSplit seam) ----

func TestCreateBill_EmptySplitMode(t *testing.T) {
	orig := isValidBillSplit
	isValidBillSplit = func(models4splitus.SplitMode) bool { return true }
	t.Cleanup(func() { isValidBillSplit = orig })

	billEntity := newMinimalBillEntity("u1")
	billEntity.SplitMode = ""
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "Missing required property SplitMode") {
		t.Errorf("expected missing SplitMode error, got: %v", err)
	}
}

// ---- InsertBillEntity: key generation error ----

func TestInsertBillEntity_NewKeyError(t *testing.T) {
	wantErr := errors.New("key generation failed")
	orig := newBillKey
	newBillKey = func() (*dal.Key, error) { return nil, wantErr }
	t.Cleanup(func() { newBillKey = orig })

	_, err := InsertBillEntity(context.Background(), nil, newMinimalBillEntity("u1"))
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- DeleteBill: RemoveBill error ----

func TestDeleteBill_RemoveBillError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)
	seedSpaceWithMembers(t, ctx, memDB)

	wantErr := errors.New("remove bill failed")
	orig := splitusSpaceRemoveBill
	splitusSpaceRemoveBill = func(*models4splitus.SplitusSpaceDbo, models4splitus.BillEntry) (bool, error) {
		return false, wantErr
	}
	t.Cleanup(func() { splitusSpaceRemoveBill = orig })

	if _, err := DeleteBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- RestoreBill: AddBill error ----

func TestRestoreBill_AddBillError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)
	seedSpaceWithMembers(t, ctx, memDB)

	wantErr := errors.New("add bill failed")
	orig := splitusSpaceAddBill
	splitusSpaceAddBill = func(*models4splitus.SplitusSpaceDbo, models4splitus.BillEntry) (bool, error) {
		return false, wantErr
	}
	t.Cleanup(func() { splitusSpaceAddBill = orig })

	if _, err := RestoreBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- delayedUpdateGroupWithBill: AddBill error ----

func TestDelayedUpdateGroupWithBill_AddBillError(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, memDB, newMinimalBillEntity("u1"))
	seedSpaceWithMembers(t, ctx, memDB)

	wantErr := errors.New("add bill failed")
	orig := splitusSpaceAddBill
	splitusSpaceAddBill = func(*models4splitus.SplitusSpaceDbo, models4splitus.BillEntry) (bool, error) {
		return false, wantErr
	}
	t.Cleanup(func() { splitusSpaceAddBill = orig })

	if err := delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), bill.ID); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- Settle2members: SetGroupMembers returning no updates ----

func TestSettle2members_SetGroupMembersNoUpdates(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))

	orig := setSpaceGroupMembers
	setSpaceGroupMembers = func(*models4splitus.SplitusSpaceDbo, []briefs4splitus.SpaceSplitMember) []update.Update {
		return nil
	}
	t.Cleanup(func() { setSpaceGroupMembers = orig })

	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil || !strings.Contains(err.Error(), "members not changed") {
		t.Errorf("expected members-not-changed error, got: %v", err)
	}
}

// ---- delayedUpdateUserWithBill: SetOutstandingBills error / nil bill data ----

func TestDelayedUpdateUserWithBill_SetOutstandingBillsError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, fakeDB{DB: memDB, get: serveBillGet(updBillDbo(models4splitus.BillStatusOutstanding, updUserID))})
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		"other-bill": {Name: "Other"},
	})

	wantErr := errors.New("set outstanding bills failed")
	orig := setUserOutstandingBills
	setUserOutstandingBills = func(*models4splitus.SplitusUserDbo, map[string]briefs4splitus.BillBrief) error {
		return wantErr
	}
	t.Cleanup(func() { setUserOutstandingBills = orig })

	err := delayedUpdateUserWithBill(ctx, updBillID, updUserID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestDelayedUpdateUserWithBill_NilBillData(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, memDB)
	seedUserExt(t, ctx, memDB, nil)

	orig := getBillEntryByID
	getBillEntryByID = func(_ context.Context, _ dal.ReadSession, billID string) (models4splitus.BillEntry, error) {
		return models4splitus.BillEntry{}, nil // nil Data with nil error
	}
	t.Cleanup(func() { getBillEntryByID = orig })

	err := delayedUpdateUserWithBill(ctx, updBillID, updUserID)
	if err == nil || !strings.Contains(err.Error(), "bill.BillDbo == nil") {
		t.Errorf("expected nil bill data error, got: %v", err)
	}
}

// ---- AssignBillToGroup: currency branch with successful balance application ----

func TestAssignBillToGroup_WithCurrency_ApplySuccess(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}},
	)
	billEntity := newMinimalBillEntity("u1") // Currency EUR

	orig := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { applyBillBalanceDifference = orig })

	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u1")
		return err
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
}

func TestAssignBillToGroup_WithCurrency_SpaceSetError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}},
	)
	billEntity := newMinimalBillEntity("u1")

	orig := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { applyBillBalanceDifference = orig })

	wantErr := errors.New("space set failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
		_, _, err := AssignBillToGroup(ctx, failingTx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u1")
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- AddBillMember beyond bill.Data.AddOrGetMember (via the billAddOrGetMember seam) ----

// seedAddBillMemberFixtures stores a bill (one member m1/u1 paid in full) and a
// splitus space (member m2/u2) so AddBillMember's space lookup succeeds.
func seedAddBillMemberFixtures(t *testing.T, ctx context.Context, db dal.DB) models4splitus.BillEntry {
	t.Helper()
	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 100},
	}
	key := dal.NewKeyWithID(models4splitus.BillKind, "abm-bill-1")
	rec := dal.NewRecordWithData(key, billEntity)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, rec)
	}); err != nil {
		t.Fatalf("failed to seed bill: %v", err)
	}
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2"}},
	)
	bill := models4splitus.BillEntry{Data: billEntity}
	bill.ID = "abm-bill-1"
	bill.Key = key
	bill.Record = rec
	return bill
}

// overrideBillAddOrGetMember replaces the billAddOrGetMember seam to return
// fixed values, restoring the original on cleanup.
func overrideBillAddOrGetMember(t *testing.T, changed bool, index int, member *briefs4splitus.BillMemberBrief, members []*briefs4splitus.BillMemberBrief) {
	t.Helper()
	orig := billAddOrGetMember
	billAddOrGetMember = func(*models4splitus.BillDbo, string, string, string, string) (bool, bool, int, *briefs4splitus.BillMemberBrief, []*briefs4splitus.BillMemberBrief) {
		return false, changed, index, member, members
	}
	t.Cleanup(func() { billAddOrGetMember = orig })
}

// twoBillMembers returns m1 (creator, paid in full) and m2 (added member).
func twoBillMembers() (*briefs4splitus.BillMemberBrief, []*briefs4splitus.BillMemberBrief) {
	m1 := &briefs4splitus.BillMemberBrief{
		MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 50,
	}
	m2 := &briefs4splitus.BillMemberBrief{
		MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2",
			ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
				"u1": briefs4splitus.MemberContactBrief{ContactID: "c2"},
			}},
		Owes: 50,
	}
	return m2, []*briefs4splitus.BillMemberBrief{m1, m2}
}

func TestAddBillMember_NotChanged(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	overrideBillAddOrGetMember(t, false, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, changed, isJoined, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", 0)
		if changed || isJoined {
			t.Errorf("expected changed=false isJoined=false, got %v %v", changed, isJoined)
		}
		return err
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddBillMember_PaidAlreadySet(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	m2.Paid = 30
	overrideBillAddOrGetMember(t, false, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, changed, _, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		if changed {
			t.Error("expected changed=false when paid is already set")
		}
		return err
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddBillMember_PaidEqualsTotal_Success(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	origApply := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { applyBillBalanceDifference = origApply })

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, changed, isJoined, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(100))
		if err != nil {
			return err
		}
		if !changed || !isJoined {
			t.Errorf("expected changed and isJoined, got %v %v", changed, isJoined)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddBillMember_PaidExceedsTotal(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers() // m1 already paid 100
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "exceeds bill amount") {
		t.Errorf("expected paid-exceeds error, got: %v", err)
	}
}

func TestAddBillMember_PartialPaid_Success(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	members[0].Paid = 0 // so paidTotal stays within the bill amount
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	origApply := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return false, nil // no group change: skip the space save
	}
	t.Cleanup(func() { applyBillBalanceDifference = origApply })

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, isJoined, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		if err != nil {
			return err
		}
		if !isJoined {
			t.Error("expected isJoined")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddBillMember_SetBillMembersError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	m2.Name = "" // fails bill member validation
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", 0)
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "no name") {
		t.Errorf("expected member validation error, got: %v", err)
	}
}

func TestAddBillMember_SaveBillError(t *testing.T) {
	// SaveBill fails because the delayer errors after the bill Set succeeds.
	wantErr := errors.New("delay failed")
	origUsersWithBill := delayerUpdateUsersWithBill
	delayerUpdateUsersWithBill = errDelayer{id: "test", err: wantErr}
	defer func() { delayerUpdateUsersWithBill = origUsersWithBill }()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", 0)
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestAddBillMember_ApplyBalanceError(t *testing.T) {
	// The real ApplyBillBalanceDifference stub always errors; the balance
	// changed because m1's paid was zeroed by SetBillMembers recalculation.
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	members[0].Paid = 0
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "failed to apply bill difference") {
		t.Errorf("expected apply-balance error, got: %v", err)
	}
}

func TestAddBillMember_SpaceSetErrorAfterApply(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	members[0].Paid = 0
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	origApply := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { applyBillBalanceDifference = origApply })

	wantErr := errors.New("space set failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, set: func(rec dal.Record) error {
			if rec.Key().Parent() != nil { // fail only the space module record
				return wantErr
			}
			return nil
		}}
		_, _, _, _, err := AddBillMember(ctx, failingTx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestAddBillMember_HistoryInsertError(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := seedAddBillMemberFixtures(t, ctx, db)
	m2, members := twoBillMembers()
	members[0].Paid = 0
	overrideBillAddOrGetMember(t, true, 1, m2, members)

	origApply := applyBillBalanceDifference
	applyBillBalanceDifference = func(*models4splitus.SplitusSpaceDbo, money.CurrencyCode, briefs4splitus.BillBalanceDifference) (bool, error) {
		return false, nil
	}
	t.Cleanup(func() { applyBillBalanceDifference = origApply })

	wantErr := errors.New("history insert failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
		_, _, _, _, err := AddBillMember(ctx, failingTx, "u1", bill, "m2", "u2", "User2", decimal.Decimal64p2(30))
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}
