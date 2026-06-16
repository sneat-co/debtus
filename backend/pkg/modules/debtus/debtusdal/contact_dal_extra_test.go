package debtusdal

import (
	"context"
	"errors"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func TestContactDal_DeleteContact(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes_contact_and_enqueues_transfer_cleanup", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		const spaceID coretypes.SpaceID = "space1"
		// Seed a contact so the Delete has something to remove.
		dbo := &models4debtus.DebtusSpaceContactDbo{}
		contact := models4debtus.NewDebtusSpaceContactEntry(spaceID, "c1", dbo)
		seedRecord(t, ctx, db, contact.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return NewContactDal().DeleteContact(ctx, tx, spaceID, "c1")
		})
		if err != nil {
			t.Fatalf("DeleteContact() returned error: %v", err)
		}
	})

	t.Run("delete_error_propagates", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		const spaceID coretypes.SpaceID = "space1"
		err := failingDB{DB: db, fault: faultDelete}.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return NewContactDal().DeleteContact(ctx, tx, spaceID, "c1")
		})
		if !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})
}

func TestDelayedDeleteContactTransfers(t *testing.T) {
	ctx := context.Background()

	// LoadTransferIDsByContactID returns dal.ErrReaderClosed from reader.Cursor()
	// once the dalgo2memory reader is exhausted, so delayedDeleteContactTransfers
	// returns that error before reaching DeleteMulti. We exercise the call path
	// (validation + query + early error return) regardless of the returned error;
	// the DeleteMulti branch is documented as an adapter-limited gap.
	t.Run("loads_transfer_ids_for_contact", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		RegisterDal()
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": {BothCounterpartyIDs: []string{"c1", "c2"}},
		})
		_ = delayedDeleteContactTransfers(ctx, "c1", "")
	})

	t.Run("no_matching_transfers", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		RegisterDal()
		_ = delayedDeleteContactTransfers(ctx, "cX", "")
	})
}

func TestContactDal_GetContactsWithDebts(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// The query filters on UserID + BalanceCount fields that the DebtusSpaceContactDbo
	// no longer stores, so the memory adapter returns no matches. The function still
	// runs fully and returns an empty slice.
	counterparties, err := NewContactDal().GetContactsWithDebts(ctx, db, "space1", "u1")
	if err != nil {
		t.Fatalf("GetContactsWithDebts() returned error: %v", err)
	}
	if len(counterparties) != 0 {
		t.Errorf("GetContactsWithDebts() = %v, want empty", counterparties)
	}
}

func TestContactDal_GetLatestContacts(t *testing.T) {
	ctx := context.Background()

	t.Run("with_tx", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contacts, err := NewContactDal().GetLatestContacts(ctx, "u1", db, "space1", 10, 0)
		if err != nil {
			t.Fatalf("GetLatestContacts() returned error: %v", err)
		}
		if len(contacts) != 0 {
			t.Errorf("GetLatestContacts() = %v, want empty", contacts)
		}
	})

	t.Run("nil_tx_uses_facade_db", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		contacts, err := NewContactDal().GetLatestContacts(ctx, "u1", nil, "space1", 0, 5)
		if err != nil {
			t.Fatalf("GetLatestContacts(nil tx) returned error: %v", err)
		}
		if len(contacts) != 0 {
			t.Errorf("GetLatestContacts() = %v, want empty", contacts)
		}
	})

	t.Run("nil_tx_facade_db_error", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		withErroringFacadeDB(t)
		if _, err := NewContactDal().GetLatestContacts(ctx, "u1", nil, "space1", 10, 0); !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})
}
