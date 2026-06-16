package models4debtus

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
)

// Constructors that wrap record.NewDataWithID (or assign record data directly)
// must nil-guard their data argument, as dalgo panics on nil record data and
// callers like the DAL layer construct records with nil data before tx.Get().

func TestNewReceiptWithoutID_NilData(t *testing.T) {
	receipt := NewReceiptWithoutID(nil) // must not panic
	if receipt.Data == nil {
		t.Fatal("NewReceiptWithoutID(nil).Data == nil")
	}
}

func TestNewInviteClaim_NilData(t *testing.T) {
	claim := NewInviteClaim("42", nil) // must not panic
	if claim.Data == nil {
		t.Fatal("NewInviteClaim(\"42\", nil).Data == nil")
	}

	// Mirror the real usage in inviteclaim_delay.go: construct with nil data,
	// then tx.Get() and read claim.Data fields.
	ctx := context.Background()
	db := dalgo2memory.NewDB()
	seeded := NewInviteClaim("42", &InviteClaimData{InviteCode: "test-invite-code"})
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, seeded.Record)
	}); err != nil {
		t.Fatalf("failed to seed invite claim: %v", err)
	}
	if err := db.Get(ctx, claim.Record); err != nil {
		t.Fatalf("failed to get invite claim: %v", err)
	}
	if claim.Data.InviteCode != "test-invite-code" {
		t.Errorf("claim.Data.InviteCode = %q, want %q", claim.Data.InviteCode, "test-invite-code")
	}
}

func TestNewFeedback_NilData(t *testing.T) {
	feedback := NewFeedback("7", nil) // must not panic
	if feedback.FeedbackData == nil {
		t.Fatal("NewFeedback(7, nil).FeedbackData == nil")
	}

	// Mirror the real usage in feedback_dal.go GetFeedbackByID: construct with
	// nil data, then tx.Get() and read feedback.FeedbackData fields.
	ctx := context.Background()
	db := dalgo2memory.NewDB()
	seeded := NewFeedback("7", &FeedbackData{Rate: "5", UserStrID: "u1"})
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, seeded.Record)
	}); err != nil {
		t.Fatalf("failed to seed feedback: %v", err)
	}
	if err := db.Get(ctx, feedback.Record); err != nil {
		t.Fatalf("failed to get feedback: %v", err)
	}
	if feedback.Rate != "5" {
		t.Errorf("feedback.Rate = %q, want %q", feedback.Rate, "5")
	}
}

func TestNewInviteClaimWithoutID_NilData(t *testing.T) {
	claim := NewInviteClaimWithoutID(nil) // must not panic
	if claim.Data == nil {
		t.Fatal("NewInviteClaimWithoutID(nil).Data == nil")
	}
}

func TestNewRewardWithIncompleteKey_NilData(t *testing.T) {
	reward := NewRewardWithIncompleteKey(nil) // must not panic
	if reward.Data == nil {
		t.Fatal("NewRewardWithIncompleteKey(nil).Data == nil")
	}
}

// TestConstructorsNilDataAudit verifies every other models4debtus record
// constructor that accepts a data pointer is safe to call with nil data.
func TestConstructorsNilDataAudit(t *testing.T) {
	for _, tt := range []struct {
		name    string
		dataNil func() bool
	}{
		{"NewReceipt", func() bool { return NewReceipt("r1", nil).Data == nil }},
		{"NewTransfer", func() bool { return NewTransfer("t1", nil).Data == nil }},
		{"NewTransferWithIncompleteKey", func() bool { return NewTransferWithIncompleteKey(nil).Data == nil }},
		{"NewInvite", func() bool { return NewInvite("i1", nil).Data == nil }},
		{"NewTwilioSms", func() bool { return NewTwilioSms("s1", nil).Data == nil }},
		{"NewDebtusSpaceContactEntry", func() bool { return NewDebtusSpaceContactEntry("space1", "c1", nil).Data == nil }},
		{"NewReward", func() bool { return NewReward("rw1", nil).Data == nil }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dataNil() { // must not panic
				t.Errorf("%s with nil data returned entry with nil Data", tt.name)
			}
		})
	}

	t.Run("NewReferrerEntry", func(t *testing.T) {
		// NewReferrerEntry intentionally panics on nil: it is only used to
		// insert new records, so nil data is a programming error.
		defer func() {
			if r := recover(); r == nil {
				t.Error("NewReferrerEntry(nil) expected to panic")
			}
		}()
		NewReferrerEntry(nil)
	})
}
