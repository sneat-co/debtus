package facade4splitus

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/strongoapp/with"
)

func newMinimalBillEntity(creatorUserID string) *models4splitus.BillDbo {
	billEntity := new(models4splitus.BillDbo)
	billEntity.Status = models4splitus.BillStatusOutstanding
	billEntity.SplitMode = models4splitus.SplitModeEqually
	billEntity.CreatorUserID = creatorUserID
	billEntity.AmountTotal = 100
	billEntity.Currency = "EUR"
	billEntity.Name = "Test bill"
	billEntity.CreatedFields = with.CreatedFields{
		CreatedAtField: with.CreatedAtField{CreatedAt: time.Now()},
		CreatedByField: with.CreatedByField{CreatedBy: creatorUserID},
	}
	return billEntity
}

// createBillViaTx runs CreateBill in a memory-DB transaction, optionally
// seeding standard contacts first, and returns the CreateBill result.
func createBillViaTx(t *testing.T, billEntity *models4splitus.BillDbo, contactIDs ...string) (bill models4splitus.BillEntry, createErr error) {
	t.Helper()
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	if len(contactIDs) > 0 {
		if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			for _, contactID := range contactIDs {
				contact := dal4contactus.NewContactEntry(coretypes.SpaceID(spaceID), contactID)
				if err := tx.Set(ctx, contact.Record); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.Fatalf("failed to seed contacts: %v", err)
		}
	}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		bill, createErr = CreateBill(ctx, tx, spaceID, billEntity)
		return nil // expected errors are asserted by the caller
	}); err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	return
}

func createBillInDB(t *testing.T, ctx context.Context, db dal.DB, billEntity *models4splitus.BillDbo) models4splitus.BillEntry {
	t.Helper()
	var bill models4splitus.BillEntry
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, err = CreateBill(ctx, tx, spaceID, billEntity)
		return err
	})
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}
	return bill
}

func TestCreateBill_InsertsBillAndHistory(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	if bill.ID == "" {
		t.Fatal("expected non-empty bill ID")
	}
	stored, err := GetBillByID(ctx, nil, bill.ID)
	if err != nil {
		t.Fatalf("failed to get bill by ID: %v", err)
	}
	if stored.Data.Name != "Test bill" {
		t.Errorf("unexpected bill name: %q", stored.Data.Name)
	}
	if stored.Data.Status != models4splitus.BillStatusOutstanding {
		t.Errorf("unexpected bill status: %q", stored.Data.Status)
	}
}

func TestDeleteBill_And_RestoreBill(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	bill := createBillInDB(t, ctx, db, newMinimalBillEntity("u1"))

	deleted, err := DeleteBill(ctx, bill.ID, "u1")
	if err != nil {
		t.Fatalf("DeleteBill failed: %v", err)
	}
	if deleted.Data.Status != models4splitus.BillStatusDeleted {
		t.Errorf("expected status deleted, got %q", deleted.Data.Status)
	}

	// Deleting a deleted bill again is a no-op for status draft/outstanding branch
	if _, err = DeleteBill(ctx, bill.ID, "u1"); err != nil {
		t.Fatalf("second DeleteBill failed: %v", err)
	}

	restored, err := RestoreBill(ctx, bill.ID, "u1")
	if err != nil {
		t.Fatalf("RestoreBill failed: %v", err)
	}
	if restored.Data.Status != models4splitus.BillStatusDraft {
		t.Errorf("expected status %q, got %q", models4splitus.BillStatusDraft, restored.Data.Status)
	}

	// Restoring a non-deleted bill returns an error
	if _, err = RestoreBill(ctx, bill.ID, "u1"); err == nil {
		t.Error("expected error restoring a non-deleted bill")
	}
}

// TestCreateBill_SourcesMemberUserIDFromStandardContact verifies that when a
// bill member references a contact (and has no UserID yet), CreateBill resolves
// the member's UserID from the STANDARD contactus contact's UserID.
func TestCreateBill_SourcesMemberUserIDFromStandardContact(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed a standard contact "c1" owned by user "resolved-user".
	stdContact := dal4contactus.NewContactEntry(coretypes.SpaceID(spaceID), "c1")
	stdContact.Data.UserID = "resolved-user"
	stdContact.Data.SetName("full", "Counterparty")
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, stdContact.Record)
	}); err != nil {
		t.Fatalf("failed to seed standard contact: %v", err)
	}

	billEntity := newMinimalBillEntity("u1")
	billEntity.SplitMode = models4splitus.SplitModeEqually
	if err := billEntity.SetBillMembers([]*briefs4splitus.BillMemberBrief{
		{
			MemberBrief: briefs4splitus.MemberBrief{
				ID:   "m1",
				Name: "Counterparty",
				ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
					"u1": briefs4splitus.MemberContactBrief{ContactID: "c1", ContactName: "Counterparty"},
				},
			},
			Owes: 100,
			Paid: 100,
		},
	}); err != nil {
		t.Fatalf("SetBillMembers failed: %v", err)
	}

	var bill models4splitus.BillEntry
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, err = CreateBill(ctx, tx, spaceID, billEntity)
		return err
	}); err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	members := bill.Data.GetBillMembers()
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if got := members[0].UserID; got != "resolved-user" {
		t.Errorf("member.UserID = %q, want %q (sourced from standard contact)", got, "resolved-user")
	}
}

func TestDeleteBill_RemovesBillFromSplitusSpace(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	billEntity := newMinimalBillEntity("u1")
	billEntity.SpaceID = coretypes.SpaceID(spaceID)
	billEntity.Members = []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First", UserID: "u1"}, Paid: 100, Owes: 50},
		{MemberBrief: briefs4splitus.MemberBrief{
			ID: "m2", Name: "Second", UserID: "u2",
			ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
				"u1": briefs4splitus.MemberContactBrief{ContactID: "c2", ContactName: "Second"},
			},
		}, Owes: 50},
	}

	bill := createBillInDB(t, ctx, db, billEntity)

	// Seed the splitus space with members and the outstanding bill.
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	splitusSpace.Data.Members = []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "First", UserID: "u1"}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Second", UserID: "u2"}},
	}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		if _, err := splitusSpace.Data.AddBill(bill); err != nil {
			return err
		}
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitus space: %v", err)
	}

	if _, err := DeleteBill(ctx, bill.ID, "u1"); err != nil {
		t.Fatalf("DeleteBill failed: %v", err)
	}

	reloaded := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := db.Get(ctx, reloaded.Record); err != nil {
		t.Fatalf("failed to reload splitus space: %v", err)
	}
	if _, ok := reloaded.Data.GetOutstandingBills()[bill.ID]; ok {
		t.Error("expected bill to be removed from outstanding bills")
	}
	for _, m := range reloaded.Data.GetGroupMembers() {
		if balance := m.Balance["EUR"]; balance != 0 {
			t.Errorf("expected member %s balance reverted to 0, got %v", m.ID, balance)
		}
	}
}
