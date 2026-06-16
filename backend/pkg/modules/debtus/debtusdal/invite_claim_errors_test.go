package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestClaimInvite_errorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("invite_not_found", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		RegisterDal()
		setupDelayers(t)
		// No invite seeded -> tx.Get(invite) returns not-found.
		if err := NewInviteDal().ClaimInvite(ctx, "u1", "NOPE", "Telegram", "DebtusBot"); err == nil {
			t.Error("expected error when invite is missing, got nil")
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		RegisterDal()
		setupDelayers(t)
		// Invite exists but the claiming user does not -> tx.Get(user) returns not-found.
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewInvite("HELLO2", &models4debtus.InviteData{CreatedByUserID: "creator"}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		if err := NewInviteDal().ClaimInvite(ctx, "no-user", "HELLO2", "Telegram", "DebtusBot"); err == nil {
			t.Error("expected error when claiming user is missing, got nil")
		}
	})
}
