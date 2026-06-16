package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestInviteDalGae_GetInvite(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_invite_when_exists", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewInvite("CODE1", &models4debtus.InviteData{CreatedByUserID: "u1"}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}

		// With explicit tx
		err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			invite, err := NewInviteDal().GetInvite(ctx, tx, "CODE1")
			if err != nil {
				return err
			}
			if invite.ID != "CODE1" {
				t.Errorf("invite.ID = %q, want CODE1", invite.ID)
			}
			if invite.Data.CreatedByUserID != "u1" {
				t.Errorf("invite.CreatedByUserID = %q, want u1", invite.Data.CreatedByUserID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("GetInvite(with tx) returned error: %v", err)
		}
	})

	t.Run("uses_facade_db_when_tx_nil", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewInvite("CODE2", &models4debtus.InviteData{CreatedByUserID: "u2"}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		invite, err := NewInviteDal().GetInvite(ctx, nil, "CODE2")
		if err != nil {
			t.Fatalf("GetInvite(nil tx) returned error: %v", err)
		}
		if invite.Data.CreatedByUserID != "u2" {
			t.Errorf("invite.CreatedByUserID = %q, want u2", invite.Data.CreatedByUserID)
		}
	})

	t.Run("returns_not_found_for_missing_invite", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		_, err := NewInviteDal().GetInvite(ctx, db, "NOSUCH")
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}
