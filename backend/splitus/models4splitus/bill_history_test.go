package models4splitus

import (
	"testing"
	"time"

	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/strongo/decimal"
)

func TestBillsHistoryDbo_BillSettlements(t *testing.T) {
	entity := &BillsHistoryDbo{}
	settlements := []briefs4splitus.BillSettlementJson{
		{BillID: "b1", GroupID: "g1", Amount: 500},
		{BillID: "b2", GroupID: "g1", Amount: 300},
	}
	entity.SetBillSettlements("g1", settlements)

	got := entity.BillSettlements()
	if len(got) != 2 {
		t.Fatalf("expected 2 settlements, got %d", len(got))
	}
	if got[0].BillID != "b1" || got[1].BillID != "b2" {
		t.Errorf("unexpected settlements: %+v", got)
	}
}

func TestBillsHistoryDbo_SetBillSettlements(t *testing.T) {
	entity := &BillsHistoryDbo{}
	settlements := []briefs4splitus.BillSettlementJson{
		{BillID: "b1", GroupID: "g1", Amount: 500},
		{BillID: "b2", GroupID: "g2", Amount: 300},
	}
	entity.SetBillSettlements("g1", settlements)

	if entity.BillsSettlementCount != 2 {
		t.Errorf("expected BillsSettlementCount=2, got %d", entity.BillsSettlementCount)
	}
	if len(entity.BillIDs) != 2 || entity.BillIDs[0] != "b1" || entity.BillIDs[1] != "b2" {
		t.Errorf("unexpected BillIDs: %v", entity.BillIDs)
	}
	if len(entity.GroupIDs) != 2 {
		t.Errorf("expected 2 GroupIDs, got %v", entity.GroupIDs)
	}
	if entity.BillsSettlementJson == "" {
		t.Error("expected non-empty BillsSettlementJson")
	}
}

func TestBillsHistoryDbo_SetBillSettlements_DuplicateGroup(t *testing.T) {
	entity := &BillsHistoryDbo{}
	settlements := []briefs4splitus.BillSettlementJson{
		{BillID: "b1", GroupID: "g1", Amount: 500},
		{BillID: "b2", GroupID: "g1", Amount: 300}, // same group
	}
	entity.SetBillSettlements("g1", settlements)
	if len(entity.GroupIDs) != 1 || entity.GroupIDs[0] != "g1" {
		t.Errorf("expected deduplicated GroupIDs=[g1], got %v", entity.GroupIDs)
	}
}

func TestBillsHistoryDbo_Validate(t *testing.T) {
	t.Run("empty_action_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{GroupIDs: []string{"g1"}}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for empty Action")
		}
	})

	t.Run("settled_without_json_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:   BillHistoryActionSettled,
			GroupIDs: []string{"g1"},
		}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for settled without JSON")
		}
	})

	t.Run("empty_group_ids_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{Action: BillHistoryActionCreated}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for empty GroupIDs")
		}
	})

	t.Run("mismatched_settlement_count_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:               BillHistoryActionSettled,
			GroupIDs:             []string{"g1"},
			BillIDs:              []string{"b1"},
			BillsSettlementJson:  `[{"bill":"b1","amount":500}]`,
			BillsSettlementCount: 99, // mismatch
		}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for mismatched BillsSettlementCount")
		}
	})

	t.Run("total_amount_mismatch_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:               BillHistoryActionSettled,
			GroupIDs:             []string{"g1"},
			BillIDs:              []string{"b1"},
			BillsSettlementJson:  `[{"bill":"b1","amount":500}]`,
			BillsSettlementCount: 1,
			TotalAmountAfter:     decimal.Decimal64p2(999), // mismatch with 500
		}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for TotalAmountAfter mismatch")
		}
	})

	t.Run("non_empty_count_without_json_error", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:               BillHistoryActionCreated,
			GroupIDs:             []string{"g1"},
			BillsSettlementCount: 1, // non-zero count but no JSON
		}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for non-zero count without JSON")
		}
	})

	t.Run("valid_created", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:   BillHistoryActionCreated,
			GroupIDs: []string{"g1"},
			BillIDs:  []string{"b1"},
		}
		if err := entity.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// DtCreated should be set automatically
		if entity.DtCreated.IsZero() {
			t.Error("expected DtCreated to be set")
		}
	})

	t.Run("valid_settled", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:           BillHistoryActionSettled,
			GroupIDs:         []string{"g1"},
			BillIDs:          []string{"b1"},
			TotalAmountAfter: decimal.Decimal64p2(500),
			DtCreated:        time.Now(),
		}
		entity.SetBillSettlements("g1", []briefs4splitus.BillSettlementJson{
			{BillID: "b1", GroupID: "g1", Amount: 500},
		})
		if err := entity.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("bill_id_mismatch_in_settlements", func(t *testing.T) {
		entity := &BillsHistoryDbo{
			Action:               BillHistoryActionSettled,
			GroupIDs:             []string{"g1"},
			BillIDs:              []string{"wrong-id"}, // mismatched
			BillsSettlementJson:  `[{"bill":"b1","amount":500}]`,
			BillsSettlementCount: 1,
			TotalAmountAfter:     500,
		}
		// The Validate will set BillIDs[0] mismatch check
		if err := entity.Validate(); err == nil {
			t.Error("expected error for BillID mismatch")
		}
	})
}

func TestNewBillsHistory(t *testing.T) {
	data := &BillsHistoryDbo{
		Action:   BillHistoryActionCreated,
		GroupIDs: []string{"g1"},
	}
	bh := NewBillsHistory(data)
	if bh.Key == nil {
		t.Error("expected non-nil Key")
	}
	if bh.Data != data {
		t.Error("expected Data to match input")
	}
	if bh.Key.ID == nil {
		t.Error("expected non-nil Key.ID")
	}
}

func TestNewBillHistoryBillCreated(t *testing.T) {
	bill := newTestBillEntry("b1", "Dinner", 1000, nil)
	bill.Data.CreatorUserID = "user1"
	bill.Data.SpaceID = "s1"

	t.Run("nil_splitusSpaceDbo", func(t *testing.T) {
		bh := NewBillHistoryBillCreated(bill, nil)
		if bh.Data.SplitMembersAfter != nil {
			t.Error("expected nil SplitMembersAfter for nil splitusSpaceDbo")
		}
		if bh.Data.Action != BillHistoryActionCreated {
			t.Errorf("expected action=created, got %v", bh.Data.Action)
		}
	})

	t.Run("non_nil_splitusSpaceDbo", func(t *testing.T) {
		space := &SplitusSpaceDbo{}
		space.Members = []briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		}
		bh := NewBillHistoryBillCreated(bill, space)
		if len(bh.Data.SplitMembersAfter) != 1 {
			t.Errorf("expected 1 SplitMemberAfter, got %d", len(bh.Data.SplitMembersAfter))
		}
	})
}

func TestNewBillHistoryMemberAdded(t *testing.T) {
	bill := newTestBillEntry("b1", "Dinner", 1000, nil)
	bill.Data.SpaceID = "s1"

	before := []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
	}
	after := []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
	}

	bh := NewBillHistoryMemberAdded("user1", bill, 800, before, after)
	if bh.Data.Action != BillHistoryActionMemberAdded {
		t.Errorf("expected action=member-added, got %v", bh.Data.Action)
	}
	if len(bh.Data.SplitMembersBefore) != 1 {
		t.Errorf("expected 1 member before, got %d", len(bh.Data.SplitMembersBefore))
	}
	if len(bh.Data.SplitMembersAfter) != 2 {
		t.Errorf("expected 2 members after, got %d", len(bh.Data.SplitMembersAfter))
	}
	if bh.Data.BillIDs[0] != "b1" {
		t.Errorf("expected BillIDs=[b1], got %v", bh.Data.BillIDs)
	}
}

func TestNewBillHistoryBillDeleted(t *testing.T) {
	bill := newTestBillEntry("b1", "Dinner", 1000, nil)
	bill.Data.SpaceID = "s1"
	bill.Data.Status = BillStatusOutstanding

	bh := NewBillHistoryBillDeleted("user1", bill)
	if bh.Data.Action != BillHistoryActionDeleted {
		t.Errorf("expected action=deleted, got %v", bh.Data.Action)
	}
	if bh.Data.StatusNew != BillStatusDeleted {
		t.Errorf("expected StatusNew=deleted, got %v", bh.Data.StatusNew)
	}
}

func TestNewBillHistoryBillRestored(t *testing.T) {
	bill := newTestBillEntry("b1", "Dinner", 1000, nil)
	bill.Data.SpaceID = "s1"
	bill.Data.Status = BillStatusOutstanding

	bh := NewBillHistoryBillRestored("user1", bill)
	if bh.Data.Action != BillHistoryActionRestored {
		t.Errorf("expected action=restored, got %v", bh.Data.Action)
	}
	if bh.Data.StatusOld != BillStatusDeleted {
		t.Errorf("expected StatusOld=deleted, got %v", bh.Data.StatusOld)
	}
}
