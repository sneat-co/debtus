package models4splitus

import (
	"testing"

	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
)

// --- bill_owes_calculations.go:55 — remainder > 1 panic in equal split ---
// This requires amountTotal and membersCount such that remainder > 1 after integer division.
// With adjustment this is possible: adjustedTotal=3, membersCount=2 → perMember=1, remainder=1 (OK)
// Actually with Decimal64p2 units, remainder is in hundredths.
// adjustedTotal=3 (0.03), membersCount=2 → perMember=1, remainder=1 (not > 1)
// Need remainder > 1: adjustedTotal=5, membersCount=2 → perMember=2, remainder=1 (still ≤ 1)
// adjustedTotal=4, membersCount=3 → perMember=1, remainder=1 ≤ 1
// This panic appears unreachable via valid inputs. Document as gap.

// --- bill_owes_calculations.go:115-122 — remainder loop in splitByShares ---
// With specific amountTotal and totalShares that produce large remainder:
// perShareBy100 initial: float64(amountTotal)/float64(totalShares)*100
// For amountTotal=7, totalShares=3: perShareBy100 = 7/3*100 = 233.33
// getRemainder = 7 - int(233.33)*3/100 = 7 - 6.99 → depends on int truncation
// decimal64p2 int: 7 - (233*3/100) = 7 - 6 = 1  (remainder=1, OK, no loop)
// Need larger mismatch. Try amountTotal=100 (i.e. $1.00), totalShares=7:
// perShareBy100 = 100/7*100 = 1428.57
// getRemainder = 100 - (1428*7/100) = 100 - 99 = 1 (no loop)
// Seems the loop is rarely entered. Try totalShares=3, amountTotal=100:
// perShareBy100 = 100/3*100 = 3333.33
// getRemainder = 100 - (3333*3/100) = 100 - 99 = 1 (no loop)
// This loop is unreachable through normal inputs with int truncation keeping remainder ≤ 1
// Document as gap.

// --- SplitusSpaceDbo.AddOrGetMember panic paths ---

func TestSplitusSpaceDbo_AddOrGetMember_ExistingMemberIDMismatchPanics(t *testing.T) {
	// line 54: existing member returned but member.ID != m.ID
	// This requires index to point to a member in groupMembers but with different ID than m.ID
	// m comes from AddOrGetMember on the MemberBrief slice; groupMembers is separately stored.
	// After SplitusSpaceDbo.AddOrGetMember runs AddOrGetMember on members (MemberBrief projection),
	// if the member found has ID="m1" but groupMembers[index].ID != "m1", it panics.
	// This can't happen in practice unless Members data is inconsistent.
	// Document as gap — requires inconsistent state that normal code never produces.
	t.Skip("panic requires inconsistent internal state; not coverable without seam")
}

func TestSplitusSpaceDbo_AddOrGetMember_EmptyMemberIDAfterAdd_Panics(t *testing.T) {
	// line 57: member.ID == "" panic — happens when AddOrGetMember returns member with empty ID
	// The package-level AddOrGetMember always generates a random ID, so this is unreachable.
	t.Skip("unreachable: AddOrGetMember always assigns a non-empty ID")
}

// --- bill_common.go:71-78 — BillCommon.AddOrGetMember existing-member path ---
// Lines 72-78 are the else branch: "member = billMembers[index]" where billMembers is nil.
// The function always panics on the isNew=true path (index != len(nil)-1 = -1)
// AND on the else path (billMembers[index] where billMembers is nil → panic).
// Neither is reachable without fixing the production code.
// Document as gap.

// --- bill_history.go:57 — BillSettlements panic on bad JSON ---

func TestBillsHistoryDbo_BillSettlements_BadJSONPanics(t *testing.T) {
	entity := &BillsHistoryDbo{
		BillsSettlementJson:  `{not valid json`,
		BillsSettlementCount: 1,
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid JSON in BillSettlements")
		}
	}()
	entity.BillSettlements()
}

// --- bill_history.go:64 — SetBillSettlements panic on marshal error ---
// json.Marshal on []BillSettlementJson never fails (all fields are string/decimal).
// This panic is unreachable. Document as gap.

// --- bill_history.go:147 — NewBillsHistory panic on err from dal.NewKeyWithOptions ---
// dal.NewKeyWithOptions with WithRandomStringID never returns an error.
// This panic is unreachable. Document as gap.

// --- group.go:38 — NewGroupKey panic from dal.NewKeyWithOptions ---
// Same as above — unreachable. Document as gap.

// --- group.go:77 — SetTelegramGroups: count same but JSON different ---
// When JSON changes but count stays the same, only the JSON branch fires.
// Cover by testing a case where the count matches but JSON is different:

func TestGroupDbo_SetTelegramGroups_JSONChangesCountSame(t *testing.T) {
	g := &GroupDbo{}
	groups1 := []briefs4splitus.GroupTgChatJson{{ChatID: 1}}
	g.SetTelegramGroups(groups1) // sets both JSON and count

	// Replace with different content but same length:
	groups2 := []briefs4splitus.GroupTgChatJson{{ChatID: 2}}
	changed := g.SetTelegramGroups(groups2)
	if !changed {
		t.Error("expected changed=true when JSON content changes but count stays same")
	}
	if g.TelegramGroupsCount != 1 {
		t.Errorf("expected count=1, got %d", g.TelegramGroupsCount)
	}
}

// --- splitus_space_dbo.go:102 — random ID collision in AddOrGetMember ---
// The collision loop (continue randomID) requires generated ID to match existing member ID.
// This is a random event with very low probability. Not reliably testable. Document as gap.

// --- splitus_space_dbo.go:108 — exhausted 100 attempts for random ID ---
// Requires 100 collisions in a row — practically impossible. Document as gap.

// --- splitus_space_dbo.go:150 — RemoveBill SetOutstandingBills error path ---
// SetOutstandingBills on BillsHolder just assigns v.OutstandingBills and returns nil.
// This error path is unreachable without overriding the method. Document as gap.

// --- splitus_space_dbo.go:160 — RemoveBill: nil balance nil case ---
// Line 160: groupMember.Balance == nil path in RemoveBill
// To hit this, we need a groupMember with nil Balance but matching bill member with non-zero Balance()

func TestSplitusSpaceDbo_RemoveBill_NilMemberBalance(t *testing.T) {
	v := &SplitusSpaceDbo{
		Members: []briefs4splitus.SpaceSplitMember{
			// Balance is nil by default
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		},
	}
	// Add a bill with a member that has non-zero balance (Paid != Owes)
	bill := newTestBillEntry("b1", "Dinner", 1000, []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 1000, Owes: 500},
	})

	// Directly add to outstanding bills without going through AddBill
	// (to avoid AddBill setting balance, leaving Balance nil)
	v.OutstandingBills = map[string]briefs4splitus.BillBrief{
		"b1": {Name: "Dinner", Total: 1000},
	}

	changed, err := v.RemoveBill(bill)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
}

// --- bills_holder.go:66 — SetOutstandingBills returns error path ---
// The default SetOutstandingBills implementation never errors.
// This line is unreachable without overriding the method. Document as gap.

// --- updateMemberOwesForSplitByShares remainder loop ---
// Cover the remainder > 1 loop by calling with values that produce it.
// With very small totalShares and large amountTotal:
// amountTotal=10000 (100.00), shares=[1,1,1] → totalShares=3, perShareBy100=10000/3*100=333333.33
// getRemainder = 10000 - (333333*3/100) = 10000 - 9999 = 1 (no loop)
// The loop requires remainder > 1 or remainder < -1 AFTER initial calculation.
// Try amountTotal=7 (0.07), totalShares=3:
// perShareBy100 = 7/3*100 = 233.33
// getRemainder = 7 - (233*3/100) = 7 - 6 = 1 (no loop)
// Try using Shares values that are very unbalanced:
// amountTotal=10, shares=[1]: totalShares=1, perShareBy100=10/1*100=1000
// getRemainder = 10 - (1000*1/100) = 10 - 10 = 0 (no loop)
// This loop appears unreachable via integer arithmetic with 2-decimal values. Document as gap.
