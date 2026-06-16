package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func seedRecord(t *testing.T, ctx context.Context, db dal.DB, record dal.Record) {
	t.Helper()
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, record)
	}); err != nil {
		t.Fatalf("seed record: %v", err)
	}
}

// TestClaimInvite_StringID verifies that ClaimInvite persists an InviteClaim
// with a non-empty string ID and sets user.InvitedByUserID.
func TestClaimInvite_StringID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()
	setupDelayers(t)

	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("creator1").Record)
	seedRecord(t, ctx, db, models4debtus.NewInvite("HELLO", &models4debtus.InviteData{
		CreatedByUserID: "creator1",
		MaxClaimsCount:  1,
		Type:            models4debtus.InviteTypePersonal,
	}).Record)
	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("claimer1").Record)

	if err := NewInviteDal().ClaimInvite(ctx, "claimer1", "HELLO", "Telegram", "DebtusBot"); err != nil {
		t.Fatalf("ClaimInvite() error: %v", err)
	}

	// Verify the claimant now has InvitedByUserID set.
	loaded := dbo4userus.NewUserEntry("claimer1")
	if err := db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("load claimer: %v", err)
	}
	if loaded.Data.InvitedByUserID != "creator1" {
		t.Errorf("InvitedByUserID = %q, want creator1", loaded.Data.InvitedByUserID)
	}
}

// TestDelayedUpdateInviteClaimedCount_updatesInvite seeds a claim, runs the
// delayed handler, and verifies ClaimedCount is incremented and LastClaimIDs
// ([]string) contains the claim ID.
func TestDelayedUpdateInviteClaimedCount_updatesInvite(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	seedRecord(t, ctx, db, models4debtus.NewInvite("INV1", &models4debtus.InviteData{
		CreatedByUserID: "u1",
		MaxClaimsCount:  5,
	}).Record)
	seedRecord(t, ctx, db, models4debtus.NewInviteClaim("clm-abc",
		models4debtus.NewInviteClaimData("INV1", "u2", "Telegram", "bot")).Record)

	if err := delayedUpdateInviteClaimedCount(ctx, "clm-abc"); err != nil {
		t.Fatalf("delayedUpdateInviteClaimedCount: %v", err)
	}

	updated := models4debtus.NewInvite("INV1", nil)
	if err := db.Get(ctx, updated.Record); err != nil {
		t.Fatalf("load invite: %v", err)
	}
	if updated.Data.ClaimedCount != 1 {
		t.Errorf("ClaimedCount = %d, want 1", updated.Data.ClaimedCount)
	}
	if len(updated.Data.LastClaimIDs) != 1 || updated.Data.LastClaimIDs[0] != "clm-abc" {
		t.Errorf("LastClaimIDs = %v, want [clm-abc]", updated.Data.LastClaimIDs)
	}
}

// TestDelayedUpdateInviteClaimedCount_idempotent verifies that running the
// handler twice does not double-increment ClaimedCount.
func TestDelayedUpdateInviteClaimedCount_idempotent(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	seedRecord(t, ctx, db, models4debtus.NewInvite("INV2", &models4debtus.InviteData{CreatedByUserID: "u1"}).Record)
	seedRecord(t, ctx, db, models4debtus.NewInviteClaim("clm-dup",
		models4debtus.NewInviteClaimData("INV2", "u2", "Telegram", "bot")).Record)

	for i := 0; i < 2; i++ {
		if err := delayedUpdateInviteClaimedCount(ctx, "clm-dup"); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	updated := models4debtus.NewInvite("INV2", nil)
	if err := db.Get(ctx, updated.Record); err != nil {
		t.Fatalf("load invite: %v", err)
	}
	if updated.Data.ClaimedCount != 1 {
		t.Errorf("ClaimedCount = %d after 2 runs, want 1 (idempotent)", updated.Data.ClaimedCount)
	}
}
