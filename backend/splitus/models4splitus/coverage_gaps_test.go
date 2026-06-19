package models4splitus

import (
	"testing"
	"time"

	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
)

// --- NewBillEntry with non-nil BillCommon (covers line 37-38) ---

func TestNewBillEntry_NonNilBillCommon(t *testing.T) {
	bc := newValidBillCommon()
	entry := NewBillEntry("id1", &bc)
	if entry.Data == nil {
		t.Fatal("expected non-nil Data")
	}
	if entry.Data.AmountTotal != bc.AmountTotal {
		t.Errorf("expected AmountTotal=%v, got %v", bc.AmountTotal, entry.Data.AmountTotal)
	}
}

// --- BillDbo.Validate error paths (validateBalance error + BillCommon.Validate error) ---

func TestBillDbo_Validate_ValidateBalanceError(t *testing.T) {
	// validateBalance returns error (negative Owes) → Validate returns early
	bc := newValidBillCommon()
	setCreatedAt(&bc)
	entity := NewBillEntity(bc)
	entity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Owes: -1},
	}
	if err := entity.Validate(); err == nil {
		t.Error("expected error from validateBalance")
	}
}

func TestBillDbo_Validate_BillCommonValidateError(t *testing.T) {
	// validateBalance passes (empty members), BillCommon.Validate returns error
	bc := newValidBillCommon()
	setCreatedAt(&bc)
	bc.SplitMode = "" // will cause BillCommon.Validate to return error
	entity := NewBillEntity(bc)
	if err := entity.Validate(); err == nil {
		t.Error("expected error from BillCommon.Validate")
	}
}

// --- GetBalance panic branches ---

func TestBillDbo_GetBalance_NegativeOwesPanics(t *testing.T) {
	entity := NewBillEntity(newValidBillCommon())
	entity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A"}, Owes: -1},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative Owes")
		}
	}()
	entity.GetBalance()
}

func TestBillDbo_GetBalance_NegativePaidPanics(t *testing.T) {
	entity := NewBillEntity(newValidBillCommon())
	entity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A"}, Paid: -1, Owes: 0},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative Paid")
		}
	}()
	entity.GetBalance()
}

// --- setUserIDs duplicate dedup branch ---

func TestBillCommon_SetUserIDs_Deduplication(t *testing.T) {
	bc := newValidBillCommon()
	bc.SplitMode = SplitModeEqually
	bc.AmountTotal = 1000
	entity := NewBillEntity(bc)
	// Two members with same UserID would fail validateMembersForDuplicatesAndBasicChecks,
	// call setUserIDs directly instead
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", UserID: "u1"}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", UserID: "u1"}}, // same userID
	}
	entity.setUserIDs(members)
	if len(entity.UserIDs) != 1 || entity.UserIDs[0] != "u1" {
		t.Errorf("expected dedup to [u1], got %v", entity.UserIDs)
	}
}

// --- BillCommon.AddOrGetMember ---

func TestBillCommon_AddOrGetMember_Panics(t *testing.T) {
	// AddOrGetMember panics when the newly inserted member index != len(billMembers)-1
	// The function has complex panics — just exercise it for coverage via the isNew branch
	// by having an empty member list (safe path won't panic)
	bc := BillCommon{
		AmountTotal: 1000,
		Currency:    "EUR",
	}
	// With empty Members, the existing-member check always skips, isNew=true path runs
	// But the panic "index != len(billMembers)-1" compares index (0) to len(nil)-1 = -1
	// so it will panic. Wrap in recover to cover the line.
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		bc.AddOrGetMember("m1", "u1", "", "Alice")
	}()
	if !didPanic {
		t.Error("expected AddOrGetMember to panic with empty Members")
	}
}

// --- updateMemberOwesForEqualSplit empty members branch ---

func TestUpdateMemberOwesForEqualSplit_EmptyMembers(t *testing.T) {
	// covers the membersCount==0 early return; function must not panic and slice stays empty
	members := []*briefs4splitus.BillMemberBrief{}
	updateMemberOwesForEqualSplit(1000, "", members)
	if len(members) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(members))
	}
}

// --- updateMemberOwesForEqualSplit: adjustment makes adjustedTotal < 0 panic ---

func TestUpdateMemberOwesForEqualSplit_AdjustmentPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for adjustment > total")
		}
	}()
	members := []*briefs4splitus.BillMemberBrief{
		{Adjustment: 2000}, // adjustment > amountTotal=1000
	}
	updateMemberOwesForEqualSplit(1000, "", members)
}

// --- updateMemberOwesForEqualSplit: creator listed twice panic ---

func TestUpdateMemberOwesForEqualSplit_CreatorTwicePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for creator listed twice")
		}
	}()
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{UserID: "u1"}},
		{MemberBrief: briefs4splitus.MemberBrief{UserID: "u1"}},
	}
	updateMemberOwesForEqualSplit(1000, "u1", members)
}

// --- updateMemberOwesForSplitByShares: empty members ---

func TestUpdateMemberOwesForSplitByShares_EmptyMembers(t *testing.T) {
	// covers the membersCount==0 early return; function must not panic and slice stays empty
	members := []*briefs4splitus.BillMemberBrief{}
	updateMemberOwesForSplitByShares(1000, "", members)
	if len(members) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(members))
	}
}

// --- updateMemberOwesForSplitByShares: negative shares panic ---

func TestUpdateMemberOwesForSplitByShares_NegativeSharesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative Shares")
		}
	}()
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{Shares: -1}},
	}
	updateMemberOwesForSplitByShares(1000, "", members)
}

// --- updateMemberOwesForSplitByShares: with creatorUserID ---

func TestUpdateMemberOwesForSplitByShares_WithCreator(t *testing.T) {
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: "u1", Shares: 3}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", UserID: "u2", Shares: 2}},
	}
	updateMemberOwesForSplitByShares(1000, "u1", members)
	total := members[0].Owes + members[1].Owes
	if total != 1000 {
		t.Errorf("expected total=1000, got %v", total)
	}
}

// --- AddBill: SetOutstandingBills error path (hard to trigger, skip) ---
// --- AddBill: changed=true with SetOutstandingBills ---

// --- NewBillsHistory: panic path (err from NewKeyWithOptions) ---
// This can't be triggered without seam; covered by existing test.

// --- NewGroupKey: panic on error (very unlikely, skip) ---

// --- GetTelegramGroups: error from Unmarshal ---

func TestGroupDbo_GetTelegramGroups_InvalidJSON(t *testing.T) {
	g := &GroupDbo{
		TelegramGroupsJson:  `{invalid`,
		TelegramGroupsCount: 1,
	}
	_, err := g.GetTelegramGroups()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- SetTelegramGroups: panic on marshal error (impossible for valid types) ---
// --- SetTelegramGroups: count changes but JSON same (need separate count change) ---

func TestGroupDbo_SetTelegramGroups_CountChange(t *testing.T) {
	g := &GroupDbo{
		TelegramGroupsJson:  `[]`,
		TelegramGroupsCount: 5, // mismatch with empty JSON
	}
	groups := []briefs4splitus.GroupTgChatJson{}
	changed := g.SetTelegramGroups(groups)
	if !changed {
		t.Error("expected changed=true when count changes")
	}
}

// --- SplitusSpaceDbo.AddOrGetMember: existing member panic path ---

// --- AddOrGetMember (package-level): contactID match path ---

func TestAddOrGetMember_ContactIDMatch(t *testing.T) {
	members := []briefs4splitus.MemberBrief{
		{ID: "m1", Name: "Alice", UserID: "u1", ContactIDs: []string{"c1"}},
	}
	idx, member, isNew, _ := AddOrGetMember(members, "", "", "c1", "Alice")
	if isNew {
		t.Error("expected isNew=false for existing contact ID")
	}
	if idx != 0 {
		t.Errorf("expected idx=0, got %d", idx)
	}
	if member.ID != "m1" {
		t.Errorf("expected member ID=m1, got %v", member.ID)
	}
}

func TestAddOrGetMember_NewContactIDAdded(t *testing.T) {
	members := []briefs4splitus.MemberBrief{
		{ID: "m1", Name: "Alice", UserID: "u1"},
	}
	_, member, isNew, changed := AddOrGetMember(members, "", "u1", "c1", "Alice")
	if isNew {
		t.Error("expected isNew=false for existing userID")
	}
	if !changed {
		t.Error("expected changed=true when contactID is added")
	}
	if len(member.ContactIDs) == 0 || member.ContactIDs[0] != "c1" {
		t.Errorf("expected contactID c1 added, got %v", member.ContactIDs)
	}
}

func TestAddOrGetMember_ExistingContactID(t *testing.T) {
	members := []briefs4splitus.MemberBrief{
		{ID: "m1", Name: "Alice", UserID: "u1", ContactIDs: []string{"c1"}},
	}
	_, _, isNew, changed := AddOrGetMember(members, "", "u1", "c1", "Alice")
	if isNew {
		t.Error("expected isNew=false")
	}
	if changed {
		t.Error("expected changed=false for existing contactID")
	}
}

func TestAddOrGetMember_NoUserNoContact(t *testing.T) {
	// userID="" and contactID="" → skips lookup, creates new member
	members := []briefs4splitus.MemberBrief{
		{ID: "m1", Name: "Alice"},
	}
	_, _, isNew, _ := AddOrGetMember(members, "", "", "", "Bob")
	if !isNew {
		t.Error("expected isNew=true when no userID/contactID")
	}
}

// --- SplitusSpaceDbo.RemoveBill: balance zeroed out (delete branch) ---

func TestSplitusSpaceDbo_RemoveBill_ZeroesBalance(t *testing.T) {
	v := &SplitusSpaceDbo{
		Members: []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		},
	}
	bill := newTestBillEntry("b1", "Dinner", 1000, []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 1000, Owes: 500},
	})
	if _, err := v.AddBill(bill); err != nil {
		t.Fatalf("AddBill: %v", err)
	}
	if _, err := v.RemoveBill(bill); err != nil {
		t.Fatalf("RemoveBill: %v", err)
	}
	// balance[EUR] should be zero and deleted
	members := v.GetGroupMembers()
	if members[0].Balance != nil && members[0].Balance["EUR"] != 0 {
		t.Errorf("expected EUR balance to be zero/deleted, got %v", members[0].Balance)
	}
}

// --- BillScheduleEntity.Validate: covers the CreatorUserID panic path ---

func TestBillScheduleEntity_Validate_PanicOnMissingCreator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing CreatorUserID")
		}
	}()
	bc := newValidBillCommon()
	setCreatedAt(&bc)
	bc.CreatorUserID = ""
	entity := &BillScheduleEntity{BillCommon: bc}
	_ = entity.Validate()
}

// --- NewBillsHistory: covers the panic (err != nil) branch ---
// The only way to get err from NewKeyWithOptions is if options are invalid,
// but with WithRandomStringID it always succeeds. Not coverable without seam.

// --- NewGroupKey panic path ---
// dal.NewKeyWithOptions panics only on internal error. Not coverable without seam.

// cover the "else" branch in AddBill (changed=false → SetOutstandingBills not called)
func TestBillsHolder_AddBill_NoChangeDoesNotCallSet(t *testing.T) {
	v := &BillsHolder{}
	bill := newTestBillEntry("b1", "Dinner", 1000, nil)
	// First add
	if _, _, err := v.AddBill(bill); err != nil {
		t.Fatal(err)
	}
	// Second add: same bill → changed=false, SetOutstandingBills NOT called (line 66 not hit)
	isNew, changed, err := v.AddBill(bill)
	if err != nil || isNew || changed {
		t.Errorf("expected isNew=false changed=false err=nil, got isNew=%v changed=%v err=%v", isNew, changed, err)
	}
}

// cover the BillsHistoryDbo.Validate path: DtCreated non-zero (first branch skipped)
func TestBillsHistoryDbo_Validate_DtCreatedAlreadySet(t *testing.T) {
	entity := &BillsHistoryDbo{
		DtCreated: time.Now(),
		Action:    BillHistoryActionCreated,
		GroupIDs:  []string{"g1"},
	}
	if err := entity.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
