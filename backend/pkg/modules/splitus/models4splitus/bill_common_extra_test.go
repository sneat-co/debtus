package models4splitus

import (
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
)

func TestBillCommon_GetUserGroupID(t *testing.T) {
	bc := BillCommon{SpaceID: "space42"}
	if got := bc.GetUserGroupID(); got != "space42" {
		t.Errorf("expected space42, got %v", got)
	}
}

func TestBillCommon_AssignToGroup(t *testing.T) {
	t.Run("empty_groupID_error", func(t *testing.T) {
		bc := BillCommon{}
		if err := bc.AssignToGroup(""); err == nil {
			t.Error("expected error for empty groupID")
		}
	})

	t.Run("first_assign", func(t *testing.T) {
		bc := BillCommon{}
		if err := bc.AssignToGroup("g1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(bc.SpaceID) != "g1" {
			t.Errorf("expected SpaceID=g1, got %v", bc.SpaceID)
		}
	})

	t.Run("same_group_no_error", func(t *testing.T) {
		bc := BillCommon{SpaceID: "g1"}
		if err := bc.AssignToGroup("g1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("conflict_error", func(t *testing.T) {
		bc := BillCommon{SpaceID: "g1"}
		if err := bc.AssignToGroup("g2"); err == nil {
			t.Error("expected conflict error")
		}
	})
}

func TestBillCommon_IsOkToSplit(t *testing.T) {
	t.Run("too_few_members", func(t *testing.T) {
		bc := BillCommon{AmountTotal: 1000}
		if bc.IsOkToSplit() {
			t.Error("expected false for 0 members")
		}
		bc.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A"}},
		}
		if bc.IsOkToSplit() {
			t.Error("expected false for 1 member")
		}
	})

	t.Run("paid_less_than_total", func(t *testing.T) {
		bc := BillCommon{
			AmountTotal: 1000,
			Members: []*briefs4splitus.BillMemberBrief{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A"}, Paid: 500},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "B"}, Paid: 400},
			},
		}
		if bc.IsOkToSplit() {
			t.Error("expected false when paid < total")
		}
	})

	t.Run("paid_equals_total", func(t *testing.T) {
		bc := BillCommon{
			AmountTotal: 1000,
			Members: []*briefs4splitus.BillMemberBrief{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A"}, Paid: 600},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "B"}, Paid: 400},
			},
		}
		if !bc.IsOkToSplit() {
			t.Error("expected true when paid == total")
		}
	})
}

func TestBillCommon_TotalAmount(t *testing.T) {
	bc := BillCommon{Currency: "USD", AmountTotal: 1234}
	got := bc.TotalAmount()
	if got.Currency != money.CurrencyCode("USD") {
		t.Errorf("expected USD, got %v", got.Currency)
	}
	if got.Value != 1234 {
		t.Errorf("expected 1234, got %v", got.Value)
	}
}

func TestBillCommon_GetMembers(t *testing.T) {
	bc := BillCommon{
		Members: []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
		},
	}
	members := bc.GetMembers()
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].ID != "m1" || members[1].ID != "m2" {
		t.Errorf("unexpected members: %+v", members)
	}
}
