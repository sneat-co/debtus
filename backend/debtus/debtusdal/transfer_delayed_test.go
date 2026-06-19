package debtusdal

import (
	"context"
	"testing"

	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/strongoapp/person"
)

func TestDelayedUpdateTransfersWithCounterparty(t *testing.T) {
	ctx := context.Background()

	t.Run("noop_when_creatorCounterpartyID_empty", func(t *testing.T) {
		if err := delayedUpdateTransfersWithCounterparty(ctx, "space1", "", "ccp"); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("noop_when_counterpartyCounterpartyID_empty", func(t *testing.T) {
		if err := delayedUpdateTransfersWithCounterparty(ctx, "space1", "ccp", ""); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("enqueues_per_transfer_with_missing_counterparty", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		// Transfer with creator counterparty "cp1" and a missing counterparty (""),
		// matching the first query (BothCounterpartyIDs contains cp1 AND "").
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": {BothCounterpartyIDs: []string{"cp1", ""}},
		})
		if err := delayedUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "ccp1"); err != nil {
			t.Errorf("delayedUpdateTransfersWithCounterparty() returned error: %v", err)
		}
	})

	t.Run("else_branch_when_no_missing_counterparty", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		// Transfer where both counterparties present — first query returns nothing,
		// so the else branch runs the second (2-counterparty) query.
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": {BothCounterpartyIDs: []string{"cp1", "ccp1"}},
		})
		if err := delayedUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "ccp1"); err != nil {
			t.Errorf("delayedUpdateTransfersWithCounterparty() returned error: %v", err)
		}
	})

	t.Run("else_branch_no_transfers_at_all", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		if err := delayedUpdateTransfersWithCounterparty(ctx, "space1", "cpX", "ccpX"); err != nil {
			t.Errorf("delayedUpdateTransfersWithCounterparty() returned error: %v", err)
		}
	})
}

func TestDelayedUpdateTransfersWithCreatorName(t *testing.T) {
	ctx := context.Background()

	t.Run("not_found_user_returns_nil", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		// User does not exist -> GetUser returns not-found -> err set to nil.
		if err := delayedUpdateTransfersWithCreatorName(ctx, "no-such-user"); err != nil {
			t.Errorf("expected nil for missing user, got %v", err)
		}
	})

	t.Run("no_matching_transfers_returns_nil", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		user := dbo4userus.NewUserEntry("u1")
		user.Data.Names = &person.NameFields{FirstName: "Alice"}
		seedRecord(t, ctx, db, user.Record)
		// Transfer belongs to other users only — the reader yields no records for
		// u1, so the loop exits on ErrNoMoreRecords without spawning a goroutine.
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": {BothUserIDs: []string{"u8", "u9"}},
		})
		if err := delayedUpdateTransfersWithCreatorName(ctx, "u1"); err != nil {
			t.Errorf("delayedUpdateTransfersWithCreatorName() returned error: %v", err)
		}
	})
}
