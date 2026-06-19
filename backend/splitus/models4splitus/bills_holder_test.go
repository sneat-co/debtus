package models4splitus

import (
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/strongo/decimal"
)

func newTestBillEntry(id, name string, amount decimal.Decimal64p2, members []*briefs4splitus.BillMemberBrief) BillEntry {
	bill := NewBillEntry(id, nil)
	bill.Data.Name = name
	bill.Data.AmountTotal = amount
	bill.Data.Currency = "EUR"
	bill.Data.Members = members
	return bill
}

func TestBillsHolder_AddBill(t *testing.T) {
	t.Run("adds_new_bill_to_empty_holder", func(t *testing.T) {
		v := &BillsHolder{}
		bill := newTestBillEntry("b1", "Dinner", 100, nil)
		isNew, changed, err := v.AddBill(bill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isNew {
			t.Error("expected isNew=true when adding a new bill")
		}
		if !changed {
			t.Error("expected changed=true when adding a new bill")
		}
		brief, ok := v.GetOutstandingBills()["b1"]
		if !ok {
			t.Fatal("expected bill b1 in outstanding bills")
		}
		if brief.Name != "Dinner" || brief.Total != 100 || brief.Currency != "EUR" {
			t.Errorf("unexpected brief: %+v", brief)
		}
	})

	t.Run("same_bill_twice_is_not_a_change", func(t *testing.T) {
		v := &BillsHolder{}
		bill := newTestBillEntry("b1", "Dinner", 100, nil)
		if _, _, err := v.AddBill(bill); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		isNew, changed, err := v.AddBill(bill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isNew {
			t.Error("expected isNew=false when adding the same bill twice")
		}
		if changed {
			t.Error("expected changed=false when adding the same bill twice")
		}
	})

	t.Run("updates_existing_bill_brief", func(t *testing.T) {
		v := &BillsHolder{}
		bill := newTestBillEntry("b1", "Dinner", 100, nil)
		if _, _, err := v.AddBill(bill); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bill = newTestBillEntry("b1", "Lunch", 80, []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First"}},
		})
		isNew, changed, err := v.AddBill(bill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isNew {
			t.Error("expected isNew=false when updating an existing bill brief")
		}
		if !changed {
			t.Error("expected changed=true when bill brief fields changed")
		}
		brief := v.GetOutstandingBills()["b1"]
		if brief.Name != "Lunch" {
			t.Errorf("expected updated name Lunch, got %q", brief.Name)
		}
		if brief.Total != 80 {
			t.Errorf("expected updated total 80, got %v", brief.Total)
		}
		if brief.MembersCount != 1 {
			t.Errorf("expected updated members count 1, got %v", brief.MembersCount)
		}
	})
}

func TestSplitusSpaceDbo_AddBill(t *testing.T) {
	t.Run("applies_member_balances_for_new_bill", func(t *testing.T) {
		v := &SplitusSpaceDbo{
			Members: []briefs4splitus.SpaceSplitMember{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First"}},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Second"}},
			},
		}
		bill := newTestBillEntry("b1", "Dinner", 100, []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First"}, Paid: 100, Owes: 50},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Second"}, Owes: 50},
		})
		changed, err := v.AddBill(bill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		members := v.GetGroupMembers()
		if balance := members[0].Balance[money.CurrencyCode("EUR")]; balance != 50 {
			t.Errorf("expected m1 balance 50, got %v", balance)
		}
		if balance := members[1].Balance[money.CurrencyCode("EUR")]; balance != -50 {
			t.Errorf("expected m2 balance -50, got %v", balance)
		}
	})

	t.Run("no_change_no_balance_update", func(t *testing.T) {
		v := &SplitusSpaceDbo{
			Members: []briefs4splitus.SpaceSplitMember{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First"}},
			},
		}
		bill := newTestBillEntry("b1", "Dinner", 100, []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First"}, Paid: 100, Owes: 100},
		})
		if _, err := v.AddBill(bill); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		changed, err := v.AddBill(bill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false on second AddBill with same data")
		}
	})
}
