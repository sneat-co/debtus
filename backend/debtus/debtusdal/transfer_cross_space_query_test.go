package debtusdal

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

// TestLoadTransfersByContactID_CrossSpaceIsolation documents how "space
// boundedness" actually works for transfer queries.
//
// Unlike contacts (DebtusSpaceContactDbo, stored under
// /spaces/{spaceID}/ext/debtus/contacts/{contactID} via
// dbo4spaceus.NewSpaceModuleItemKey) and the per-space DebtusSpaceDbo
// aggregate, transfers themselves are stored in a single ROOT collection
// (models4debtus.TransfersCollectionRef = dal.NewRootCollectionRef(...)), not
// nested under a space path. There is no `spaceID` field/filter on a
// transfer query at all (see transfer_dal.go: every query here filters by
// "BothUserIDs"/"BothCounterpartyIDs" array membership, never by spaceID).
//
// Isolation is therefore achieved indirectly: a query for "transfers
// involving contact X" only ever returns transfers that literally reference
// contactID X in BothCounterpartyIDs. Since contactus contact IDs are unique
// per (spaceID, contactID) storage key but globally unique as generated IDs,
// this does not leak between spaces in practice — but it also means there is
// no way to ask "give me every transfer that touches space S" directly; you
// have to know the space's user/contact IDs up front and query by those.
//
// This test seeds transfers that simulate the cross-space scenario: a lender
// in "space A" (contact ID prefixed spaceA-*) and a borrower contact tracked
// in "space B" (contact ID prefixed spaceB-*) share ONE transfer that
// legitimately references both; plus unrelated transfers that must not leak
// into either side's query results.
func TestLoadTransfersByContactID_CrossSpaceIsolation(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	const (
		spaceAContactID = "spaceA-lender-contact"
		spaceBContactID = "spaceB-borrower-contact"
		unrelatedSpace  = "spaceC-unrelated-contact"
	)

	seedTransfers(t, db, map[string]*models4debtus.TransferData{
		// The one legitimately cross-space transfer: references a contact
		// tracked in space A and a contact tracked in space B.
		"cross-space-transfer": {
			BothCounterpartyIDs: []string{spaceAContactID, spaceBContactID},
			BothUserIDs:         []string{"u-lender", "u-borrower"},
			DtCreated:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		// An unrelated transfer that touches neither space A nor space B.
		"unrelated-transfer": {
			BothCounterpartyIDs: []string{unrelatedSpace, "some-other-contact"},
			BothUserIDs:         []string{"u-other-1", "u-other-2"},
			DtCreated:           time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	})

	dal := NewTransferDal()

	t.Run("space_A_contact_sees_the_cross-space_transfer", func(t *testing.T) {
		transfers, _, err := dal.LoadTransfersByContactID(ctx, spaceAContactID, 0, 10)
		if err != nil {
			t.Fatalf("LoadTransfersByContactID() error: %v", err)
		}
		ids := transferEntryIDs(transfers)
		if !slices.Contains(ids, "cross-space-transfer") {
			t.Errorf("expected space A's query to include the cross-space transfer, got %v", ids)
		}
		if slices.Contains(ids, "unrelated-transfer") {
			t.Errorf("space A's query leaked an unrelated transfer: %v", ids)
		}
	})

	t.Run("space_B_contact_sees_the_same_cross-space_transfer", func(t *testing.T) {
		transfers, _, err := dal.LoadTransfersByContactID(ctx, spaceBContactID, 0, 10)
		if err != nil {
			t.Fatalf("LoadTransfersByContactID() error: %v", err)
		}
		ids := transferEntryIDs(transfers)
		if !slices.Contains(ids, "cross-space-transfer") {
			t.Errorf("expected space B's query to include the cross-space transfer, got %v", ids)
		}
		if slices.Contains(ids, "unrelated-transfer") {
			t.Errorf("space B's query leaked an unrelated transfer: %v", ids)
		}
	})

	t.Run("unrelated_space_does_not_see_the_cross-space_transfer", func(t *testing.T) {
		transfers, _, err := dal.LoadTransfersByContactID(ctx, unrelatedSpace, 0, 10)
		if err != nil {
			t.Fatalf("LoadTransfersByContactID() error: %v", err)
		}
		ids := transferEntryIDs(transfers)
		if slices.Contains(ids, "cross-space-transfer") {
			t.Errorf("unrelated space's query unexpectedly saw the cross-space transfer: %v", ids)
		}
		if !slices.Contains(ids, "unrelated-transfer") {
			t.Errorf("expected unrelated space's query to include its own transfer, got %v", ids)
		}
	})
}
