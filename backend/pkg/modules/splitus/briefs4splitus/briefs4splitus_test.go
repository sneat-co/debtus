package briefs4splitus

import (
	"strings"
	"testing"
)

func TestBillBalanceDifference_Reverse(t *testing.T) {
	diff := BillBalanceDifference{"m1": 100, "m2": -200}
	rev := diff.Reverse()
	if rev["m1"] != -100 {
		t.Errorf("m1: got %v want -100", rev["m1"])
	}
	if rev["m2"] != 200 {
		t.Errorf("m2: got %v want 200", rev["m2"])
	}
}

func TestMemberBrief_GetID(t *testing.T) {
	m := MemberBrief{ID: "abc"}
	if got := m.GetID(); got != "abc" {
		t.Errorf("GetID: got %q want %q", got, "abc")
	}
}

func TestMemberBrief_GetName(t *testing.T) {
	m := MemberBrief{Name: "Alice"}
	if got := m.GetName(); got != "Alice" {
		t.Errorf("GetName: got %q want %q", got, "Alice")
	}
}

func TestMemberBrief_GetShares(t *testing.T) {
	m := MemberBrief{Shares: 3}
	if got := m.GetShares(); got != 3 {
		t.Errorf("GetShares: got %d want 3", got)
	}
}

func TestSpaceSplitMember_String(t *testing.T) {
	m := &SpaceSplitMember{MemberBrief: MemberBrief{ID: "x", Name: "Bob"}}
	s := m.String()
	if !strings.Contains(s, "Bob") {
		t.Errorf("String: got %q, expected to contain 'Bob'", s)
	}
}

func TestBillMemberBrief_Balance(t *testing.T) {
	m := BillMemberBrief{}
	m.Paid = 1000
	m.Owes = 400
	if got := m.Balance(); got != 600 {
		t.Errorf("Balance: got %v want 600", got)
	}
}

func TestBillMemberBrief_String(t *testing.T) {
	m := &BillMemberBrief{}
	m.ID = "id1"
	m.Name = "Carol"
	s := m.String()
	if !strings.Contains(s, "Carol") {
		t.Errorf("String: got %q, expected to contain 'Carol'", s)
	}
}

func TestBillBalanceDifference_Reverse_empty(t *testing.T) {
	diff := BillBalanceDifference{}
	rev := diff.Reverse()
	if len(rev) != 0 {
		t.Errorf("expected empty reverse, got %v", rev)
	}
}

// TestBillBalanceByMember_BillBalanceDifference_largePrevious covers the branch
// where len(previous) > len(current)+1, triggering the capacity resize.
func TestBillBalanceByMember_BillBalanceDifference_largePrevious(t *testing.T) {
	// previous has 3 members, current has 0 → len(previous) > len(current)+1
	previous := BillBalanceByMember{
		"m1": BillMemberBalance{Paid: 100, Owes: 0},
		"m2": BillMemberBalance{Paid: 200, Owes: 0},
		"m3": BillMemberBalance{Paid: 300, Owes: 0},
	}
	current := BillBalanceByMember{}
	diff := current.BillBalanceDifference(previous)
	if len(diff) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(diff), diff)
	}
}
