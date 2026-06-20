package facade4debtus

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func TestLinkReceiptUsers(t *testing.T) {
	ctx := context.Background()

	// newLinkerWithParties returns a linker whose changes already have non-nil
	// inviter/invited parties (LinkReceiptUsers assigns into changes.inviter.user
	// etc. and would nil-deref otherwise — its production caller supplies them).
	newLinkerWithParties := func() *ReceiptUsersLinker {
		changes := newReceiptDbChanges()
		changes.inviter = &userLinkingParty{}
		changes.invited = &userLinkingParty{}
		return NewReceiptUsersLinker(changes)
	}

	t.Run("invited_user_not_found_returns_error", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		_, err := newLinkerWithParties().LinkReceiptUsers(ctx, "r1", "u2")
		if !dal.IsNotFound(err) {
			t.Fatalf("expected not-found for missing invited user, got: %v", err)
		}
	})
}

// NOTE: a full happy-/deep-path test of LinkReceiptUsers is not feasible with
// the in-memory DB. After loading the invited user it runs a transaction that
// calls linkUsersByReceiptWithinTransaction, which dereferences
// changes.inviter.contact.Data (the inviter's contactus contact). LinkReceiptUsers
// does not populate that contact before the linker runs — its production call
// graph supplies a fully pre-loaded contact graph that the in-memory harness
// cannot reconstruct. The deep linker body itself is covered via the direct
// linkUsersByReceiptWithinTransaction test (see receipt_link_deep_test.go).
