package models4splitus

import (
	"errors"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
)

func newTestSpace() *SplitusSpaceDbo {
	return &SplitusSpaceDbo{
		Members: []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", UserID: "u1"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", UserID: "u2"}},
		},
	}
}

func TestSplitusSpaceDbo_AddOrGetMember(t *testing.T) {
	t.Run("add_new_member", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		isNew, changed, _, member, _ := v.AddOrGetMember("u1", "", "Alice")
		if !isNew {
			t.Error("expected isNew=true")
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if member.UserID != "u1" {
			t.Errorf("expected UserID=u1, got %v", member.UserID)
		}
	})

	t.Run("get_existing_member", func(t *testing.T) {
		v := newTestSpace()
		isNew, _, _, member, _ := v.AddOrGetMember("u1", "", "Alice")
		if isNew {
			t.Error("expected isNew=false for existing member")
		}
		if member.ID != "m1" {
			t.Errorf("expected member ID=m1, got %v", member.ID)
		}
	})

	t.Run("empty_userID_panics", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty userID")
			}
		}()
		v.AddOrGetMember("", "", "Alice")
	})

	t.Run("empty_name_panics", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty name")
			}
		}()
		v.AddOrGetMember("u1", "", "")
	})
}

func TestSplitusSpaceDbo_RemoveBill(t *testing.T) {
	t.Run("bill_not_in_outstanding_no_change", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		bill := newTestBillEntry("b1", "Dinner", 1000, nil)
		changed, err := v.RemoveBill(bill)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false for bill not in outstanding")
		}
	})

	t.Run("removes_existing_bill", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		bill := newTestBillEntry("b1", "Dinner", 1000, nil)
		if _, err := v.AddBill(bill); err != nil {
			t.Fatalf("AddBill failed: %v", err)
		}
		changed, err := v.RemoveBill(bill)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true when removing existing bill")
		}
		if _, ok := v.GetOutstandingBills()["b1"]; ok {
			t.Error("expected bill to be removed from outstanding")
		}
	})

	t.Run("reverses_member_balance", func(t *testing.T) {
		v := &SplitusSpaceDbo{
			Members: []briefs4splitus.SpaceSplitMember{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
			},
		}
		bill := newTestBillEntry("b1", "Dinner", 1000, []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 1000, Owes: 500},
		})
		if _, err := v.AddBill(bill); err != nil {
			t.Fatalf("AddBill failed: %v", err)
		}
		if _, err := v.RemoveBill(bill); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestSplitusSpaceDbo_GetGroupMemberByID(t *testing.T) {
	v := newTestSpace()

	t.Run("empty_id_error", func(t *testing.T) {
		_, err := v.GetGroupMemberByID("")
		if err == nil {
			t.Error("expected error for empty id")
		}
		if !errors.Is(err, dal.ErrRecordNotFound) {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("unknown_id_error", func(t *testing.T) {
		_, err := v.GetGroupMemberByID("unknown")
		if err == nil {
			t.Error("expected error for unknown id")
		}
		if !errors.Is(err, dal.ErrRecordNotFound) {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("known_id_returns_member", func(t *testing.T) {
		m, err := v.GetGroupMemberByID("m1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if m.ID != "m1" {
			t.Errorf("expected ID=m1, got %v", m.ID)
		}
	})
}

func TestSplitusSpaceDbo_GetGroupMemberByUserID(t *testing.T) {
	v := newTestSpace()

	t.Run("empty_userID_error", func(t *testing.T) {
		_, err := v.GetGroupMemberByUserID("")
		if err == nil {
			t.Error("expected error for empty userID")
		}
		if !errors.Is(err, dal.ErrRecordNotFound) {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("unknown_userID_error", func(t *testing.T) {
		_, err := v.GetGroupMemberByUserID("unknown")
		if err == nil {
			t.Error("expected error for unknown userID")
		}
	})

	t.Run("known_userID_returns_member", func(t *testing.T) {
		m, err := v.GetGroupMemberByUserID("u2")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if m.ID != "m2" {
			t.Errorf("expected ID=m2, got %v", m.ID)
		}
	})
}

func TestSplitusSpaceDbo_GetMembers(t *testing.T) {
	v := newTestSpace()
	members := v.GetMembers()
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].ID != "m1" || members[1].ID != "m2" {
		t.Errorf("unexpected members: %+v", members)
	}
}

func TestSplitusSpaceDbo_GetSplitMode(t *testing.T) {
	t.Run("no_members_returns_equally", func(t *testing.T) {
		v := &SplitusSpaceDbo{}
		if v.GetSplitMode() != SplitModeEqually {
			t.Errorf("expected SplitModeEqually for empty members")
		}
	})

	t.Run("equal_shares_returns_equally", func(t *testing.T) {
		v := &SplitusSpaceDbo{
			Members: []briefs4splitus.SpaceSplitMember{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A", Shares: 10}},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "B", Shares: 10}},
			},
		}
		if v.GetSplitMode() != SplitModeEqually {
			t.Errorf("expected SplitModeEqually for equal shares")
		}
	})

	t.Run("unequal_shares_returns_share", func(t *testing.T) {
		v := &SplitusSpaceDbo{
			Members: []briefs4splitus.SpaceSplitMember{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A", Shares: 3}},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "B", Shares: 7}},
			},
		}
		if v.GetSplitMode() != SplitModeShare {
			t.Errorf("expected SplitModeShare for unequal shares")
		}
	})
}

func TestSplitusSpaceDbo_TotalShares(t *testing.T) {
	v := &SplitusSpaceDbo{
		Members: []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "A", Shares: 3}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "B", Shares: 7}},
		},
	}
	if total := v.TotalShares(); total != 10 {
		t.Errorf("expected TotalShares=10, got %v", total)
	}
}

func TestSplitusSpaceDbo_UserIsMember(t *testing.T) {
	v := newTestSpace()
	if !v.UserIsMember("u1") {
		t.Error("expected u1 to be a member")
	}
	if v.UserIsMember("unknown") {
		t.Error("expected unknown to not be a member")
	}
}

func TestBillsHolder_ApplyBillBalanceDifference(t *testing.T) {
	t.Run("empty_currency_panics", func(t *testing.T) {
		v := &BillsHolder{}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty currency")
			}
		}()
		_, _ = v.ApplyBillBalanceDifference("", briefs4splitus.BillBalanceDifference{})
	})

	t.Run("padded_currency_panics", func(t *testing.T) {
		v := &BillsHolder{}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for padded currency")
			}
		}()
		_, _ = v.ApplyBillBalanceDifference(" EUR", briefs4splitus.BillBalanceDifference{})
	})

	t.Run("returns_not_implemented_error", func(t *testing.T) {
		v := &BillsHolder{}
		_, err := v.ApplyBillBalanceDifference("EUR", briefs4splitus.BillBalanceDifference{})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("expected 'not implemented' error, got: %v", err)
		}
	})
}

func TestBillsHolder_GetOutstandingBalance(t *testing.T) {
	v := &BillsHolder{}
	bill1 := newTestBillEntry("b1", "Dinner", 1000, nil)
	bill1.Data.Currency = "EUR"
	bill2 := newTestBillEntry("b2", "Lunch", 500, nil)
	bill2.Data.Currency = "USD"

	if _, _, err := v.AddBill(bill1); err != nil {
		t.Fatalf("AddBill failed: %v", err)
	}
	if _, _, err := v.AddBill(bill2); err != nil {
		t.Fatalf("AddBill failed: %v", err)
	}

	// Set UserBalance on the outstanding bills
	bills := v.GetOutstandingBills()
	b1 := bills["b1"]
	b1.UserBalance = 300
	bills["b1"] = b1
	b2 := bills["b2"]
	b2.UserBalance = 200
	bills["b2"] = b2
	_ = v.SetOutstandingBills(bills)

	balance := v.GetOutstandingBalance()
	if balance["EUR"] != 300 {
		t.Errorf("expected EUR balance=300, got %v", balance["EUR"])
	}
	if balance["USD"] != 200 {
		t.Errorf("expected USD balance=200, got %v", balance["USD"])
	}
}
