package models4splitus

import (
	"errors"
	"testing"
	"time"

	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
)

func newValidBillCommon() BillCommon {
	return BillCommon{
		CreatorUserID: "user1",
		SplitMode:     SplitModeEqually,
		Status:        BillStatusOutstanding,
		AmountTotal:   1000,
		Currency:      "EUR",
	}
}

func setCreatedAt(bc *BillCommon) {
	bc.CreatedAt = time.Now()
}

func TestNewBillEntity(t *testing.T) {
	bc := BillCommon{AmountTotal: 100, Currency: "USD"}
	entity := NewBillEntity(bc)
	if entity == nil {
		t.Fatal("expected non-nil entity")
	}
	if entity.AmountTotal != 100 {
		t.Errorf("expected AmountTotal=100, got %v", entity.AmountTotal)
	}
}

func TestNewBillEntry_NilBillCommon(t *testing.T) {
	entry := NewBillEntry("id1", nil)
	if entry.Data == nil {
		t.Fatal("expected non-nil Data")
	}
	// Data should be zero-value BillDbo
	if entry.Data.AmountTotal != 0 {
		t.Errorf("expected zero AmountTotal, got %v", entry.Data.AmountTotal)
	}
}

func TestBillDbo_Validate(t *testing.T) {
	t.Run("valid_bill", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		entity := NewBillEntity(bc)
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 600, Owes: 600},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}, Paid: 400, Owes: 400},
		}
		entity.AmountTotal = 1000
		if err := entity.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// check SponsorIDs / DebtorIDs are empty when paid == owes
		if len(entity.SponsorIDs) != 0 {
			t.Errorf("expected no sponsors, got %v", entity.SponsorIDs)
		}
	})

	t.Run("sponsor_and_debtor_classification", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		entity := NewBillEntity(bc)
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 1000, Owes: 500},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}, Paid: 0, Owes: 500},
		}
		entity.AmountTotal = 1000
		if err := entity.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(entity.SponsorIDs) != 1 || entity.SponsorIDs[0] != "m1" {
			t.Errorf("expected SponsorIDs=[m1], got %v", entity.SponsorIDs)
		}
		if len(entity.DebtorIDs) != 1 || entity.DebtorIDs[0] != "m2" {
			t.Errorf("expected DebtorIDs=[m2], got %v", entity.DebtorIDs)
		}
	})

	t.Run("negative_owes_error", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		entity := NewBillEntity(bc)
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Owes: -1},
		}
		err := entity.validateBalance()
		if err == nil {
			t.Fatal("expected error for negative owes")
		}
		if !errors.Is(err, ErrNegativeAmount) {
			t.Errorf("expected ErrNegativeAmount, got %v", err)
		}
	})

	t.Run("negative_paid_error", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: -1, Owes: 0},
		}
		err := entity.validateBalance()
		if err == nil {
			t.Fatal("expected error for negative paid")
		}
		if !errors.Is(err, ErrNegativeAmount) {
			t.Errorf("expected ErrNegativeAmount, got %v", err)
		}
	})

	t.Run("total_owed_mismatch", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		entity.AmountTotal = 1000
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Owes: 500, Paid: 500},
		}
		err := entity.validateBalance()
		if err == nil {
			t.Fatal("expected error for totalOwed mismatch")
		}
		if !errors.Is(err, ErrTotalOwedIsNotMatchingBillAmount) && !errors.Is(err, ErrBillTotalBalanceIsNotZero) {
			t.Errorf("got unexpected error: %v", err)
		}
	})

	t.Run("total_paid_exceeds_amount", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		entity.AmountTotal = 500
		entity.Members = []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Owes: 500, Paid: 600},
		}
		err := entity.validateBalance()
		if err == nil {
			t.Fatal("expected error for total paid exceeds amount")
		}
		if !errors.Is(err, ErrTotalPaidIsGreaterThenBillAmount) && !errors.Is(err, ErrBillTotalBalanceIsNotZero) {
			t.Errorf("got unexpected error: %v", err)
		}
	})

	t.Run("empty_members_no_error", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		if err := entity.validateBalance(); err != nil {
			t.Errorf("unexpected error with empty members: %v", err)
		}
	})
}

func TestBillDbo_GetBalance(t *testing.T) {
	bc := newValidBillCommon()
	entity := NewBillEntity(bc)
	entity.AmountTotal = 1000
	entity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 1000, Owes: 500},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}, Paid: 0, Owes: 500},
	}

	balance := entity.GetBalance()
	if len(balance) != 2 {
		t.Fatalf("expected 2 entries in balance, got %d", len(balance))
	}
	if balance["m1"].Paid != 1000 || balance["m1"].Owes != 500 {
		t.Errorf("unexpected m1 balance: %+v", balance["m1"])
	}
	if balance["m2"].Paid != 0 || balance["m2"].Owes != 500 {
		t.Errorf("unexpected m2 balance: %+v", balance["m2"])
	}
}

func TestBillDbo_GetBalance_EmptyMember(t *testing.T) {
	// member with both Owes=0 and Paid=0 should not be in balance
	bc := newValidBillCommon()
	entity := NewBillEntity(bc)
	entity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Paid: 0, Owes: 0},
	}
	balance := entity.GetBalance()
	if len(balance) != 0 {
		t.Errorf("expected empty balance for zero-paid/owes member, got %d entries", len(balance))
	}
}

func TestBillDbo_SetBillMembers(t *testing.T) {
	t.Run("equally_split", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = SplitModeEqually
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
		}
		if err := entity.SetBillMembers(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if entity.Members[0].Owes != 500 {
			t.Errorf("expected m1.Owes=500, got %v", entity.Members[0].Owes)
		}
	})

	t.Run("shares_split", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = SplitModeShare
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 3}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", Shares: 2}},
		}
		if err := entity.SetBillMembers(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		total := entity.Members[0].Owes + entity.Members[1].Owes
		if total != 1000 {
			t.Errorf("expected total owes=1000, got %v", total)
		}
	})

	t.Run("percentage_split", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = SplitModePercentage
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 70}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", Shares: 30}},
		}
		if err := entity.SetBillMembers(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		total := entity.Members[0].Owes + entity.Members[1].Owes
		if total != 1000 {
			t.Errorf("expected total owes=1000, got %v", total)
		}
	})

	t.Run("unknown_split_mode_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = "unknown-mode"
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		}
		if err := entity.SetBillMembers(members); err == nil {
			t.Error("expected error for unknown split mode")
		}
	})

	t.Run("duplicate_userid_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = SplitModeEqually
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", UserID: "user1"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", UserID: "user1"}},
		}
		if err := entity.SetBillMembers(members); err == nil {
			t.Error("expected error for duplicate userID")
		}
	})

	t.Run("sets_user_ids", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.SplitMode = SplitModeEqually
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", UserID: "user1"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", UserID: "user2"}},
		}
		if err := entity.SetBillMembers(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(entity.UserIDs) != 2 {
			t.Errorf("expected 2 UserIDs, got %v", entity.UserIDs)
		}
	})

	t.Run("dedup_user_ids_via_set_members", func(t *testing.T) {
		// setUserIDs should deduplicate
		bc := newValidBillCommon()
		bc.SplitMode = SplitModeEqually
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", UserID: "user1"}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
		}
		if err := entity.SetBillMembers(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(entity.UserIDs) != 1 || entity.UserIDs[0] != "user1" {
			t.Errorf("expected [user1], got %v", entity.UserIDs)
		}
	})
}

func TestValidateMembersForDuplicatesAndBasicChecks(t *testing.T) {
	t.Run("empty_name_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.AmountTotal = 1000
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: ""}},
		}
		err := entity.validateMembersForDuplicatesAndBasicChecks(members)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("owes_greater_than_total_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.AmountTotal = 100
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Owes: 200},
		}
		err := entity.validateMembersForDuplicatesAndBasicChecks(members)
		if err == nil {
			t.Error("expected error for owes > total")
		}
	})

	t.Run("adjustment_too_big_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.AmountTotal = 100
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Adjustment: 200},
		}
		err := entity.validateMembersForDuplicatesAndBasicChecks(members)
		if err == nil {
			t.Error("expected error for adjustment too big")
		}
	})

	t.Run("negative_adjustment_too_big_error", func(t *testing.T) {
		bc := newValidBillCommon()
		bc.AmountTotal = 100
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}, Adjustment: -200},
		}
		err := entity.validateMembersForDuplicatesAndBasicChecks(members)
		if err == nil {
			t.Error("expected error for negative adjustment too big")
		}
	})

	t.Run("assigns_member_id_when_empty", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{Name: "Alice"}},
		}
		if err := entity.validateMembersForDuplicatesAndBasicChecks(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if members[0].ID == "" {
			t.Error("expected ID to be assigned")
		}
	})

	t.Run("unequal_splits_sets_isEquallySplit_false", func(t *testing.T) {
		bc := newValidBillCommon()
		entity := NewBillEntity(bc)
		members := []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 2}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob", Shares: 3}},
		}
		if err := entity.validateMembersForDuplicatesAndBasicChecks(members); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestBillCommon_Validate(t *testing.T) {
	t.Run("missing_split_mode", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		bc.SplitMode = ""
		err := bc.Validate()
		if err == nil {
			t.Error("expected error for missing SplitMode")
		}
	})

	t.Run("missing_status", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		bc.Status = ""
		err := bc.Validate()
		if err == nil {
			t.Error("expected error for missing Status")
		}
	})

	t.Run("missing_created_at", func(t *testing.T) {
		bc := newValidBillCommon()
		// Do NOT set CreatedAt - leave as zero
		err := bc.Validate()
		if err == nil {
			t.Error("expected error for missing CreatedAt")
		}
	})

	t.Run("valid", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		err := bc.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing_creator_panics", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		bc.CreatorUserID = ""
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for missing CreatorUserID")
			}
		}()
		_ = bc.Validate()
	})
}

func TestFixTotal_Case_Minus1(t *testing.T) {
	// Decimal64p2 represents hundredths: 100 = $1.00
	// To hit case -1: amountTotal - total == -1, i.e. total = amountTotal + 1
	// amountTotal = 1000, members total = 1001
	// The loop in fixTotal never updates max, so idx ends up at the last
	// member whose Owes > 0 (both here), i.e. members[1].
	members := []*briefs4splitus.BillMemberBrief{
		{Owes: 601},
		{Owes: 400},
	}
	owesBefore := members[1].Owes
	fixTotal(1000, members)
	// The branch adds 1 to members[1]
	if members[1].Owes != owesBefore+1 {
		t.Errorf("expected members[1].Owes=%v, got %v", owesBefore+1, members[1].Owes)
	}
}

func TestFixTotal_PanicOnLargeRemainder(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for large remainder")
		}
	}()
	members := []*briefs4splitus.BillMemberBrief{
		{Owes: 100},
	}
	// amountTotal - total = 1000 - 100 = 900 => should panic
	fixTotal(1000, members)
}
