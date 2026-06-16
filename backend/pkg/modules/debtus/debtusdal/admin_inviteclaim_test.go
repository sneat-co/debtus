package debtusdal

import (
	"context"
	"testing"

	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestAdminDalGae_LatestUsers_returns_not_implemented(t *testing.T) {
	ctx := context.Background()
	_, err := NewAdminDal().LatestUsers(ctx)
	if err == nil {
		t.Error("LatestUsers() should return an error (not implemented), got nil")
	}
}

func TestDelayUpdateInviteClaimedCount_noop_when_claim_missing(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	// Set up dal4debtus.Default.Invite so delayedUpdateInviteClaimedCount can call GetInvite.
	RegisterDal()

	// Claim ID 999 does not exist — should log and return nil (not retry).
	if err := delayedUpdateInviteClaimedCount(ctx, "999"); err != nil {
		t.Errorf("expected nil when claim is missing, got: %v", err)
	}
}
