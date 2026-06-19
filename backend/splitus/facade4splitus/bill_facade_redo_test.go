package facade4splitus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/decimal"
	"github.com/strongo/strongoapp/person"
)

// ---- CreateBill: remaining validation branches ----

func TestCreateBill_PanicOnNilTx(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil tx")
		}
	}()
	_, _ = CreateBill(context.Background(), nil, spaceID, newMinimalBillEntity("u1"))
}

func TestCreateBill_EmptyCreatorUserID(t *testing.T) {
	billEntity := newMinimalBillEntity("")
	bill, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "CreatorUserID") {
		t.Errorf("expected CreatorUserID error, got: %v", err)
	}
	if bill.ID != "" {
		t.Error("bill.ID should be empty")
	}
}

// percentageBillMembers builds members for a percentage-split bill paid in
// full by the creator. Mutate the returned entity for invalid variations.
func newPercentageBill(percents ...decimal.Decimal64p2) *models4splitus.BillDbo {
	billEntity := new(models4splitus.BillDbo)
	billEntity.Status = models4splitus.BillStatusOutstanding
	billEntity.SplitMode = models4splitus.SplitModePercentage
	billEntity.CreatorUserID = "1"
	billEntity.AmountTotal = 100
	billEntity.Currency = "EUR"
	members := make([]*briefs4splitus.BillMemberBrief, len(percents))
	owesLeft := billEntity.AmountTotal
	for i, p := range percents {
		owes := decimal.Decimal64p2(int64(billEntity.AmountTotal) * int64(p) / 10000)
		if i == len(percents)-1 {
			owes = owesLeft
		}
		owesLeft -= owes
		members[i] = &briefs4splitus.BillMemberBrief{
			Percent: p,
			Owes:    owes,
			MemberBrief: briefs4splitus.MemberBrief{
				ID:   string(rune('1' + i)),
				Name: "Member " + string(rune('1'+i)),
			},
		}
		if i == 0 {
			members[i].UserID = "1"
			members[i].Paid = billEntity.AmountTotal
		} else {
			members[i].ContactByUser = briefs4splitus.MemberContactBriefsByUserID{
				"1": briefs4splitus.MemberContactBrief{ContactID: "c" + string(rune('1'+i))},
			}
		}
	}
	billEntity.Members = members
	return billEntity
}

func TestCreateBill_MemberNegativeOwes(t *testing.T) {
	billEntity := newPercentageBill(5000, 5000)
	billEntity.Members[1].Owes = -1
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "Owes is negative") {
		t.Errorf("expected negative Owes error, got: %v", err)
	}
}

func TestCreateBill_NonCreatorMemberWithoutContact(t *testing.T) {
	billEntity := newPercentageBill(5000, 5000)
	billEntity.Members[1].UserID = "3"
	billEntity.Members[1].ContactByUser = nil
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "ContactByUser) == 0") {
		t.Errorf("expected ContactByUser error, got: %v", err)
	}
}

func TestCreateBill_MemberContactWithEmptyContactID(t *testing.T) {
	billEntity := newPercentageBill(5000, 5000)
	billEntity.Members[1].ContactByUser = briefs4splitus.MemberContactBriefsByUserID{
		"1": briefs4splitus.MemberContactBrief{ContactID: ""},
	}
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "empty ContactID") {
		t.Errorf("expected empty ContactID error, got: %v", err)
	}
}

func TestCreateBill_DuplicateContactIDAcrossMembers(t *testing.T) {
	// Two userless members referencing the same contact: the duplicate is
	// detected and only one contact ID is collected.
	billEntity := newPercentageBill(5000, 2500, 2500)
	sameContact := briefs4splitus.MemberContactBriefsByUserID{
		"1": briefs4splitus.MemberContactBrief{ContactID: "dup1"},
	}
	billEntity.Members[1].ContactByUser = sameContact
	billEntity.Members[2].ContactByUser = sameContact
	bill, err := createBillViaTx(t, billEntity, "dup1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bill.Data.ContactIDs) != 1 || bill.Data.ContactIDs[0] != "dup1" {
		t.Errorf("expected single deduplicated contact ID, got: %v", bill.Data.ContactIDs)
	}
}

func TestCreateBill_GetContactsError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("get multi failed")
	billEntity := newPercentageBill(5000, 5000)
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, getMulti: func([]dal.Record) error { return wantErr }}
		_, err := CreateBill(ctx, failingTx, spaceID, billEntity)
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestCreateBill_EquallyMemberAdjustmentTooNegative(t *testing.T) {
	billEntity, err := createGoodBillSplitEqually(t)
	if err != nil {
		t.Fatal(err)
	}
	billEntity.Members[1].Adjustment = -(billEntity.AmountTotal + 100)
	_, err = createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "AdjustmentInCents < 0") {
		t.Errorf("expected too-negative adjustment error, got: %v", err)
	}
}

func TestCreateBill_EquallyUnequalShares(t *testing.T) {
	billEntity, err := createGoodBillSplitEqually(t)
	if err != nil {
		t.Fatal(err)
	}
	billEntity.Members[1].Shares = 3 // members[0].Shares == 0
	_, err = createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "Shares not equal") {
		t.Errorf("expected unequal shares error, got: %v", err)
	}
}

func TestCreateBill_EquallyTooManyDistinctAmounts(t *testing.T) {
	// owes 2.11 / 2.12 / 2.13 all deviate within 1 cent from the equal amount
	// 2.12 but give 3 distinct values > 2 + adjustmentsCount(0).
	billEntity := new(models4splitus.BillDbo)
	billEntity.Status = models4splitus.BillStatusOutstanding
	billEntity.SplitMode = models4splitus.SplitModeEqually
	billEntity.CreatorUserID = "1"
	billEntity.AmountTotal = 636
	billEntity.Currency = "EUR"
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{Owes: 211, MemberBrief: briefs4splitus.MemberBrief{ID: "1", UserID: "1", Name: "First"}, Paid: 636},
		{Owes: 212, MemberBrief: briefs4splitus.MemberBrief{ID: "2", Name: "Second", ContactByUser: briefs4splitus.MemberContactBriefsByUserID{"1": briefs4splitus.MemberContactBrief{ContactID: "2"}}}},
		{Owes: 213, MemberBrief: briefs4splitus.MemberBrief{ID: "3", Name: "Third", ContactByUser: briefs4splitus.MemberContactBriefsByUserID{"1": briefs4splitus.MemberContactBrief{ContactID: "4"}}}},
	}
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "amountsCountByValue") {
		t.Errorf("expected too many distinct amounts error, got: %v", err)
	}
}

func TestCreateBill_PercentagesDoNotSumTo100(t *testing.T) {
	billEntity := newPercentageBill(5000, 3000)
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "should be 100%") {
		t.Errorf("expected percentage sum error, got: %v", err)
	}
}

func TestCreateBill_PercentageOwesDoNotSumToTotal(t *testing.T) {
	billEntity := newPercentageBill(5000, 5000)
	billEntity.Members[0].Owes = 10
	billEntity.Members[1].Owes = 10
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "totalOwedByMembers != billEntity.AmountTotal") {
		t.Errorf("expected owed total mismatch error, got: %v", err)
	}
}

// exactAmountBill builds a bill split by exact amounts paid by the creator.
func exactAmountBill() *models4splitus.BillDbo {
	billEntity := new(models4splitus.BillDbo)
	billEntity.Status = models4splitus.BillStatusOutstanding
	billEntity.SplitMode = models4splitus.SplitModeExactAmount
	billEntity.CreatorUserID = "1"
	billEntity.AmountTotal = 100
	billEntity.Currency = "EUR"
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{Owes: 60, MemberBrief: briefs4splitus.MemberBrief{ID: "1", UserID: "1", Name: "First"}, Paid: 100},
		{Owes: 40, MemberBrief: briefs4splitus.MemberBrief{ID: "2", Name: "Second", ContactByUser: briefs4splitus.MemberContactBriefsByUserID{"1": briefs4splitus.MemberContactBrief{ContactID: "x2"}}}},
	}
	return billEntity
}

func TestCreateBill_ExactAmountWithAdjustment(t *testing.T) {
	billEntity := exactAmountBill()
	billEntity.Members[1].Adjustment = 5
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "Adjustment property not allowed") {
		t.Errorf("expected adjustment-not-allowed error, got: %v", err)
	}
}

func TestCreateBill_ExactAmountWithShares(t *testing.T) {
	billEntity := exactAmountBill()
	billEntity.Members[1].Shares = 2
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "Shares property not allowed") {
		t.Errorf("expected shares-not-allowed error, got: %v", err)
	}
}

func TestCreateBill_ExactAmountSuccess(t *testing.T) {
	billEntity := exactAmountBill()
	bill, err := createBillViaTx(t, billEntity, "x2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill.ID == "" {
		t.Error("expected non-empty bill ID")
	}
}

func TestCreateBill_ShareModeMemberWithoutShares(t *testing.T) {
	billEntity := new(models4splitus.BillDbo)
	billEntity.Status = models4splitus.BillStatusOutstanding
	billEntity.SplitMode = models4splitus.SplitModeShare
	billEntity.CreatorUserID = "1"
	billEntity.AmountTotal = 100
	billEntity.Currency = "EUR"
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{Owes: 50, MemberBrief: briefs4splitus.MemberBrief{ID: "1", UserID: "1", Name: "First", Shares: 0}, Paid: 100},
		{Owes: 50, MemberBrief: briefs4splitus.MemberBrief{ID: "2", Name: "Second", Shares: 1, ContactByUser: briefs4splitus.MemberContactBriefsByUserID{"1": briefs4splitus.MemberContactBrief{ContactID: "x2"}}}},
	}
	_, err := createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "missing Shares value") {
		t.Errorf("expected missing shares error, got: %v", err)
	}
}

func TestCreateBill_ShareModeBillSharesMismatch(t *testing.T) {
	billEntity, err := createGoodBillSplitByShare(t)
	if err != nil {
		t.Fatal(err)
	}
	billEntity.Shares = 99 // members total 6
	_, err = createBillViaTx(t, billEntity)
	if err == nil || !strings.Contains(err.Error(), "billEntity.Shares != totalSharesPerMembers") {
		t.Errorf("expected shares mismatch error, got: %v", err)
	}
}

func TestCreateBill_InsertBillEntityError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("insert failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
		_, err := CreateBill(ctx, failingTx, spaceID, newMinimalBillEntity("u1"))
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestCreateBill_BillHistoryInsertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("history insert failed")
	insertCount := 0
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error {
			insertCount++
			if insertCount > 1 { // first insert is the bill, second the history
				return wantErr
			}
			return nil
		}}
		_, err := CreateBill(ctx, failingTx, spaceID, newMinimalBillEntity("u1"))
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- AssignBillToGroup remaining branches ----

func TestAssignBillToGroup_SpaceGetError(t *testing.T) {
	// Bill has no members and the splitus space record does not exist.
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	billEntity := newMinimalBillEntity("u1")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u1")
		return err
	})
	if err == nil {
		t.Error("expected error when splitus space is missing")
	}
}

func seedSpaceWithMembers(t *testing.T, ctx context.Context, db dal.DB, members ...briefs4splitus.SpaceSplitMember) {
	t.Helper()
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	splitusSpace.Data.Members = members
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitus space: %v", err)
	}
}

func TestAssignBillToGroup_PaidSetFromCurrentUser(t *testing.T) {
	// Creator u1 is NOT a space member; current user u2 is, so the second
	// loop assigns Paid to u2's bill member.
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2"}},
	)
	billEntity := newMinimalBillEntity("u1")
	billEntity.Currency = ""

	var bill models4splitus.BillEntry
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, _, err = AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u2")
		return
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
	members := bill.Data.GetBillMembers()
	if len(members) != 1 || members[0].Paid != billEntity.AmountTotal {
		t.Errorf("expected single member paid in full, got: %+v", members)
	}
}

func TestAssignBillToGroup_UserNotMember_GetUserError(t *testing.T) {
	// Neither the creator nor the current user is a space member, and the
	// user record is missing, so dal4userus.GetUser fails.
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2"}},
	)
	billEntity := newMinimalBillEntity("u1")
	billEntity.Currency = ""

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u3")
		return err
	})
	if err == nil {
		t.Error("expected error when user record is missing")
	}
}

func TestAssignBillToGroup_UserNotMember_AddedFromUserRecord(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2"}},
	)
	// Seed the u3 user record.
	user := dbo4userus.NewUserEntry("u3")
	user.Data.Names = &person.NameFields{FirstName: "Third", LastName: "User"}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, user.Record)
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	billEntity := newMinimalBillEntity("u1")
	billEntity.Currency = ""

	var bill models4splitus.BillEntry
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, _, err = AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u3")
		return
	}); err != nil {
		t.Fatalf("AssignBillToGroup failed: %v", err)
	}
	members := bill.Data.GetBillMembers()
	if len(members) != 2 {
		t.Fatalf("expected 2 bill members, got %d", len(members))
	}
	var u3 *briefs4splitus.BillMemberBrief
	for _, m := range members {
		if m.UserID == "u3" {
			u3 = m
		}
	}
	if u3 == nil || u3.Paid != billEntity.AmountTotal {
		t.Errorf("expected u3 member paid in full, got: %+v", u3)
	}
}

func TestAssignBillToGroup_SetBillMembersError(t *testing.T) {
	// A space member without a name fails bill member validation.
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: ""}},
	)
	billEntity := newMinimalBillEntity("u1")
	billEntity.Currency = ""

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u1")
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "no name") {
		t.Errorf("expected member validation error, got: %v", err)
	}
}

func TestAssignBillToGroup_WithCurrency_ApplyBalanceError(t *testing.T) {
	// With a currency set, AssignBillToGroup calls ApplyBillBalanceDifference,
	// which is a production stub that always errors.
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}},
	)
	billEntity := newMinimalBillEntity("u1") // Currency = EUR

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, err := AssignBillToGroup(ctx, tx, models4splitus.BillEntry{Data: billEntity}, coretypes.SpaceID(spaceID), "u1")
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected ApplyBillBalanceDifference stub error, got: %v", err)
	}
}

// ---- AddBillMember ----
// BillCommon.AddOrGetMember has a production bug: the named return value
// billMembers is never assigned, so both the new-member and existing-member
// code paths panic. The statements in AddBillMember up to and including the
// bill.Data.AddOrGetMember call are covered with recover()-wrapped tests;
// everything after that call is unreachable and documented in TEST-COVERAGE.md.

func TestAddBillMember_SpaceGetError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))
	bill.Data.SpaceID = coretypes.SpaceID(spaceID) // space record not seeded

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, _, _, _, err := AddBillMember(ctx, tx, "u1", bill, "m1", "u2", "User2", 0)
		return err
	})
	if err == nil {
		t.Error("expected error when splitus space is missing")
	}
}

func TestAddBillMember_NewSpaceMember_PanicsInBillAddOrGetMember(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db) // empty space - the new member changes the group
	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))
	bill.Data.SpaceID = coretypes.SpaceID(spaceID)

	panicked := false
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, _, _, _, _ = AddBillMember(ctx, tx, "u1", bill, "m1", "u2", "User2", 0)
		return nil
	})
	if !panicked {
		t.Error("expected panic from BillCommon.AddOrGetMember production bug")
	}
}

func TestAddBillMember_ExistingSpaceMember_PanicsInBillAddOrGetMember(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	seedSpaceWithMembers(t, ctx, db,
		briefs4splitus.SpaceSplitMember{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Name: "User2"}},
	)
	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))
	bill.Data.SpaceID = coretypes.SpaceID(spaceID)

	panicked := false
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, _, _, _, _ = AddBillMember(ctx, tx, "u1", bill, "m2", "u2", "User2", 0)
		return nil
	})
	if !panicked {
		t.Error("expected panic from BillCommon.AddOrGetMember production bug")
	}
}

// ---- DeleteBill error branches ----

func TestDeleteBill_BillNotFound(t *testing.T) {
	ctx := context.Background()
	_ = sneattesting.SetupMemoryDB(t)
	if _, err := DeleteBill(ctx, "no-such-bill", "u1"); err == nil {
		t.Error("expected error for unknown bill")
	}
}

func TestDeleteBill_HistoryInsertError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, memDB, newMinimalBillEntity("u1"))

	wantErr := errors.New("history insert failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
	}})
	if _, err := DeleteBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestDeleteBill_SaveBillError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, memDB, newMinimalBillEntity("u1"))

	wantErr := errors.New("save bill failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
	}})
	if _, err := DeleteBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// deletedBillWithSpace seeds an already-deleted bill assigned to the test space.
func deletedBillWithSpace(t *testing.T, ctx context.Context, db dal.DB) models4splitus.BillEntry {
	t.Helper()
	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	billEntity.Status = models4splitus.BillStatusDeleted
	key := dal.NewKeyWithID(models4splitus.BillKind, "deleted-bill-1")
	rec := dal.NewRecordWithData(key, billEntity)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, rec)
	}); err != nil {
		t.Fatalf("failed to seed deleted bill: %v", err)
	}
	bill := models4splitus.BillEntry{Data: billEntity}
	bill.ID = "deleted-bill-1"
	bill.Key = key
	bill.Record = rec
	return bill
}

func TestDeleteBill_SpaceGetError(t *testing.T) {
	// Already-deleted bill with a spaceID whose space record does not exist:
	// the status branch is skipped and tx.Get on the space fails.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)
	if _, err := DeleteBill(ctx, bill.ID, "u1"); err == nil {
		t.Error("expected error when splitus space record is missing")
	}
}

func TestDeleteBill_SpaceSetError(t *testing.T) {
	// The deleted bill is still listed in the space's outstanding bills, so
	// RemoveBill reports a change and the subsequent tx.Set fails.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)

	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := memDB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		if _, err := splitusSpace.Data.AddBill(bill); err != nil {
			return err
		}
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitus space: %v", err)
	}

	wantErr := errors.New("space set failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
	}})
	if _, err := DeleteBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- RestoreBill error branches ----

func TestRestoreBill_BillNotFound(t *testing.T) {
	ctx := context.Background()
	_ = sneattesting.SetupMemoryDB(t)
	if _, err := RestoreBill(ctx, "no-such-bill", "u1"); err == nil {
		t.Error("expected error for unknown bill")
	}
}

func TestRestoreBill_HistoryInsertError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)

	wantErr := errors.New("history insert failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
	}})
	if _, err := RestoreBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestRestoreBill_SaveBillError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)

	wantErr := errors.New("save bill failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
	}})
	if _, err := RestoreBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestRestoreBill_SpaceGetError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)
	// No splitus space record seeded.
	if _, err := RestoreBill(ctx, bill.ID, "u1"); err == nil {
		t.Error("expected error when splitus space record is missing")
	}
}

func TestRestoreBill_SpaceSetError(t *testing.T) {
	// The space exists but does not contain the bill, so AddBill reports a
	// change and the space tx.Set fails. The bill's own Set is allowed
	// through by matching on the record key.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := deletedBillWithSpace(t, ctx, memDB)
	seedSpaceWithMembers(t, ctx, memDB)

	wantErr := errors.New("space set failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, set: func(rec dal.Record) error {
			if rec.Key().Parent() != nil { // the space module record has a parent key
				return wantErr
			}
			return nil
		}}
	}})
	if _, err := RestoreBill(ctx, bill.ID, "u1"); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- GetBillByID / InsertBillEntity / SaveBill ----

func TestGetBillByID_GetSneatDBError(t *testing.T) {
	wantErr := errors.New("no db")
	failSneatDB(t, wantErr)
	_, err := GetBillByID(context.Background(), nil, "b1")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestInsertBillEntity_InsertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("insert failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
		_, err := InsertBillEntity(ctx, failingTx, newMinimalBillEntity("u1"))
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSaveBill_SetError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	wantErr := errors.New("set failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
		return SaveBill(ctx, failingTx, bill)
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- GetBillMembersUserInfo ----

func TestGetBillMembersUserInfo_NoMembers(t *testing.T) {
	bill := models4splitus.BillEntry{Data: new(models4splitus.BillDbo)}
	infos, err := GetBillMembersUserInfo(context.Background(), bill, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected no member infos, got %d", len(infos))
	}
}

func TestGetBillMembersUserInfo_PanicsOnSuccessPath(t *testing.T) {
	// Production bug: billMembersUserInfo is never allocated, so the success
	// path panics with index out of range. Cover the statements before the
	// panic with recover().
	bill := models4splitus.BillEntry{Data: new(models4splitus.BillDbo)}
	bill.Data.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{
			ID:   "m1",
			Name: "User1",
			ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
				"7": briefs4splitus.MemberContactBrief{ContactID: "c1", ContactName: "User1"},
			},
		}},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic due to nil billMembersUserInfo slice")
		}
	}()
	_, _ = GetBillMembersUserInfo(context.Background(), bill, 7)
}

// ---- delayedUpdateBillDependencies: member delay error ----

func TestDelayedUpdateBillDependencies_DelayUserError(t *testing.T) {
	wantErr := errors.New("user delay error")
	origGroupWithBill := delayerUpdateGroupWithBill
	origUserWithBill := delayerUpdateUserWithBill
	delayerUpdateGroupWithBill = voidDelayer{id: "test"}
	delayerUpdateUserWithBill = errDelayer{id: "test", err: wantErr}
	defer func() {
		delayerUpdateGroupWithBill = origGroupWithBill
		delayerUpdateUserWithBill = origUserWithBill
	}()

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Name: "User1"}, Paid: 100, Owes: 100},
	}
	bill := createBillInDB(t, ctx, db, billEntity)

	err := delayedUpdateBillDependencies(ctx, bill.ID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- group_dal: delayedUpdateGroupWithBill error branches ----

func TestDelayedUpdateGroupWithBill_BillNotFound(t *testing.T) {
	ctx := context.Background()
	_ = sneattesting.SetupMemoryDB(t)
	if err := delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), "no-such-bill"); err == nil {
		t.Error("expected error for unknown bill")
	}
}

func TestDelayedUpdateGroupWithBill_SpaceNotFound(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))
	if err := delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), bill.ID); err == nil {
		t.Error("expected error when splitus space record is missing")
	}
}

func TestDelayedUpdateGroupWithBill_SaveSpaceError(t *testing.T) {
	restore := initTestDelayers(voidDelayer{id: "test"})
	defer restore()
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	bill := createBillInDB(t, ctx, memDB, newMinimalBillEntity("u1"))
	seedSpaceWithMembers(t, ctx, memDB)

	wantErr := errors.New("save space failed")
	overrideSneatDB(t, fakeDB{DB: memDB, wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
		return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
	}})
	if err := delayedUpdateGroupWithBill(ctx, coretypes.SpaceID(spaceID), bill.ID); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// ---- group_member_dal ----

func TestCreateGroupMember_Success(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	const newID int64 = 7
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		// dalgo2memory would assign an `int` ID which panics on the int64
		// type assertion, so emulate an adapter that assigns int64 IDs.
		idTx := txWrap{ReadwriteTransaction: tx, insert: func(rec dal.Record) error {
			rec.Key().ID = newID
			return nil
		}}
		d := NewGroupMemberDalGae()
		gm, err := d.CreateGroupMember(ctx, idTx, &models4splitus.GroupMemberData{Name: "gm"})
		if err != nil {
			return err
		}
		if gm.ID != newID {
			t.Errorf("expected ID %d, got %d", newID, gm.ID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateGroupMember_InsertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("insert failed")
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		failingTx := txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
		d := NewGroupMemberDalGae()
		_, err := d.CreateGroupMember(ctx, failingTx, &models4splitus.GroupMemberData{Name: "gm"})
		return err
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestGetGroupMemberByID_GetSneatDBError(t *testing.T) {
	wantErr := errors.New("no db")
	failSneatDB(t, wantErr)
	d := NewGroupMemberDalGae()
	_, err := d.GetGroupMemberByID(context.Background(), nil, 42)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}
