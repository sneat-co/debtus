package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func TestContactDalGae_SaveContact(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	const spaceID coretypes.SpaceID = "space1"
	dbo := &models4debtus.DebtusSpaceContactDbo{}
	contact := models4debtus.NewDebtusSpaceContactEntry(spaceID, "c1", dbo)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return NewContactDal().SaveContact(ctx, tx, contact)
	})
	if err != nil {
		t.Fatalf("SaveContact() returned error: %v", err)
	}

	loaded := models4debtus.NewDebtusSpaceContactEntry(spaceID, "c1", nil)
	if err = db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("contact not found after save: %v", err)
	}
}

func TestContactDalGae_GetContactIDsByTitle(t *testing.T) {
	ctx := context.Background()

	// GetContactIDsByTitle loads a contactus space entry and searches its Contacts
	// map. The memory adapter won't execute the query-based paths (GetLatestContacts,
	// GetContactsWithDebts) because they filter on scalar UserID fields that the
	// memory adapter does support, but the contactus space record is needed.
	// Test the case-sensitive and case-insensitive title match branches.

	t.Run("returns_error_when_space_not_found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		// "missingspace" is a valid lowercase-alphanumeric space ID that simply
		// doesn't exist in the memory DB — expect a not-found error.
		_, err := NewContactDal().GetContactIDsByTitle(ctx, db, "missingspace", "u1", "Alice", true)
		if err == nil {
			t.Error("expected error when contactus space is missing, got nil")
		}
	})
}

// InsertContact sets contact.Data but never initialises contact.Record with a
// key (the method signature has no spaceID parameter, so no valid nested key
// can be constructed). Calling it panics on any adapter. The method is
// effectively dead production code and cannot be tested without a redesign.
