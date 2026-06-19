package facade4debtus

// The previous tests in this file targeted the deleted GAE ContactDalGae API
// (NewContactDalGae, CreateContactWithinTransaction, CreateContact with int
// IDs) and were rewritten below against the current dalgo-based facade using
// an in-memory DB.
//
// CreateContact is not covered: createContactDbChanges.user is never
// populated by CreateContact (commented out since the migration), so the
// zero-contacts branch always fails with "appUser.ContactID == 0" and its
// error path dereferences contact.Data which is nil.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dbo4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
)

// TestCreateContactWithinTransaction_consistencyAsserts verifies that the
// former panic-based consistency asserts in createContactWithinTransaction now
// return errors instead of crashing the process.
func TestCreateContactWithinTransaction_consistencyAsserts(t *testing.T) {
	ctx := context.Background()

	// helper: run the function inside a real read-write transaction
	run := func(t *testing.T, changes *createContactDbChanges, counterpartyUserID string) error {
		t.Helper()
		db := sneattesting.SetupMemoryDB(t)
		var got error
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			got = createContactWithinTransaction(ctx, tx, changes, testSpaceID, counterpartyUserID, dto4contactus.ContactDetails{})
			// Always return nil so the tx wrapper does not swallow got.
			return nil
		}, dal.TxWithCrossGroup())
		return got
	}

	newChanges := func() *createContactDbChanges {
		user := dbo4userus.NewUserEntry("u1")
		return &createContactDbChanges{
			user:        user,
			debtusSpace: models4debtus.NewDebtusSpaceEntry(testSpaceID),
		}
	}

	t.Run("appUser_equals_counterpartyUserID_returns_error", func(t *testing.T) {
		changes := newChanges()
		err := run(t, changes, "u1") // same as appUser.ID
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "appUser.ContactID == counterpartyUserID") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("counterparty_contact_data_non_nil_but_empty_id_returns_error", func(t *testing.T) {
		changes := newChanges()
		// Set a non-nil Data but leave ID empty — triggers the second assert.
		changes.counterparty.Contact.Data = new(dbo4contactus.ContactDbo)
		err := run(t, changes, "u2")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "counterpartyContact.DebtusSpaceContactDbo != nil") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func seedRecords(t *testing.T, db dal.DB, records ...dal.Record) {
	t.Helper()
	err := db.RunReadwriteTransaction(context.Background(), func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.SetMulti(ctx, records)
	})
	if err != nil {
		t.Fatalf("failed to seed records: %v", err)
	}
}

const testSpaceID coretypes.SpaceID = "s1"

func TestGetDebtusSpaceContactByID(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seeded := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		seedRecords(t, db, seeded.Record)

		contact, err := GetDebtusSpaceContactByID(ctx, nil, testSpaceID, "c1")
		if err != nil {
			t.Fatalf("GetDebtusSpaceContactByID() returned error: %v", err)
		}
		if contact.Data.Status != models4debtus.DebtusContactStatusActive {
			t.Errorf("contact.Data.Status = %v, want %v", contact.Data.Status, models4debtus.DebtusContactStatusActive)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		if _, err := GetDebtusSpaceContactByID(ctx, nil, testSpaceID, "missing"); !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}

func TestGetDebtusSpaceContactsByIDs(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	c1 := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{Status: models4debtus.DebtusContactStatusActive})
	c2 := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c2", &models4debtus.DebtusSpaceContactDbo{Status: models4debtus.DebtusContactStatusArchived})
	seedRecords(t, db, c1.Record, c2.Record)

	contacts, err := GetDebtusSpaceContactsByIDs(ctx, nil, testSpaceID, []string{"c1", "c2"})
	if err != nil {
		t.Fatalf("GetDebtusSpaceContactsByIDs() returned error: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("len(contacts) = %d, want 2", len(contacts))
	}
	if contacts[0].Data.Status != models4debtus.DebtusContactStatusActive {
		t.Errorf("contacts[0].Status = %v, want active", contacts[0].Data.Status)
	}
	if contacts[1].Data.Status != models4debtus.DebtusContactStatusArchived {
		t.Errorf("contacts[1].Status = %v, want archived", contacts[1].Data.Status)
	}
}

func TestSaveContact(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
		Status: models4debtus.DebtusContactStatusActive,
	})
	if err := SaveContact(ctx, contact); err != nil {
		t.Fatalf("SaveContact() returned error: %v", err)
	}
	saved := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", nil)
	if err := db.Get(ctx, saved.Record); err != nil {
		t.Fatalf("failed to read saved contact: %v", err)
	}
	if saved.Data.Status != models4debtus.DebtusContactStatusActive {
		t.Errorf("saved.Data.Status = %v, want active", saved.Data.Status)
	}
}

func TestUpdateContact(t *testing.T) {
	ctx := context.Background()

	newContactDbo := func() *models4debtus.DebtusSpaceContactDbo {
		return &models4debtus.DebtusSpaceContactDbo{Status: models4debtus.DebtusContactStatusActive}
	}

	t.Run("updates_fields_and_debtus_space", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", newContactDbo())
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		seedRecords(t, db, contact.Record, debtusSpace.Record)

		updated, err := UpdateContact(ctx, testSpaceID, "c1", map[string]string{
			"Username":     "jbrown",
			"FirstName":    "Jack",
			"LastName":     "Brown",
			"ScreenName":   "JB",
			"EmailAddress": "jack@example.com",
			"PhoneNumber":  "1234567890",
		})
		if err != nil {
			t.Fatalf("UpdateContact() returned error: %v", err)
		}
		data := updated.Data
		if data.UserName != "jbrown" || data.FirstName != "Jack" || data.LastName != "Brown" || data.ScreenName != "JB" {
			t.Errorf("unexpected names: %+v", data.ContactDetails)
		}
		if data.EmailAddressOriginal != "jack@example.com" {
			t.Errorf("EmailAddressOriginal = %q", data.EmailAddressOriginal)
		}
		if data.PhoneNumber != 1234567890 {
			t.Errorf("PhoneNumber = %d, want 1234567890", data.PhoneNumber)
		}
		// Verify persisted
		saved := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", nil)
		if err = db.Get(ctx, saved.Record); err != nil {
			t.Fatalf("failed to read saved contact: %v", err)
		}
		if saved.Data.FirstName != "Jack" {
			t.Errorf("saved.Data.FirstName = %q, want Jack", saved.Data.FirstName)
		}
		savedSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		if err = db.Get(ctx, savedSpace.Record); err != nil {
			t.Fatalf("failed to read saved debtus space: %v", err)
		}
		if savedSpace.Data.Contacts["c1"] == nil {
			t.Error("debtus space should have a brief for the updated contact")
		}
	})

	t.Run("no_change_does_not_require_debtus_space", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		dbo := newContactDbo()
		dbo.FirstName = "Jack"
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", dbo)
		seedRecords(t, db, contact.Record) // note: no debtus space record seeded
		if _, err := UpdateContact(ctx, testSpaceID, "c1", map[string]string{"FirstName": "Jack"}); err != nil {
			t.Fatalf("UpdateContact() returned error: %v", err)
		}
	})

	t.Run("unknown_field_is_ignored", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", newContactDbo())
		seedRecords(t, db, contact.Record)
		if _, err := UpdateContact(ctx, testSpaceID, "c1", map[string]string{"NoSuchField": "x"}); err != nil {
			t.Fatalf("UpdateContact() returned error: %v", err)
		}
	})

	t.Run("invalid_phone_number", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", newContactDbo())
		seedRecords(t, db, contact.Record)
		if _, err := UpdateContact(ctx, testSpaceID, "c1", map[string]string{"PhoneNumber": "not-a-number"}); err == nil {
			t.Error("expected error for invalid phone number")
		}
	})
}

func TestDeleteContact(t *testing.T) {
	ctx := context.Background()
	userCtx := facade.NewUserContext("u1")

	t.Run("missing_contact_is_no_op", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		if err := DeleteContact(ctx, userCtx, testSpaceID, "missing"); err != nil {
			t.Errorf("expected nil error for missing contact, got: %v", err)
		}
	})

	t.Run("contact_with_counterparty_user_is_not_deletable", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})
		stdContact := dal4contactus.NewContactEntryWithData(testSpaceID, "c1", &dbo4contactus.ContactDbo{
			ContactBase: briefs4contactus.ContactBase{
				ContactBrief: briefs4contactus.ContactBrief{
					WithUserID: dbmodels.WithUserID{UserID: "u2"},
				},
			},
		})
		seedRecords(t, db, contact.Record, stdContact.Record)
		if err := DeleteContact(ctx, userCtx, testSpaceID, "c1"); !errors.Is(err, ErrContactIsNotDeletable) {
			t.Errorf("expected ErrContactIsNotDeletable, got: %v", err)
		}
	})

	t.Run("deletes_contact_with_zero_balance", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		debtusSpace.Data.Contacts = map[string]*models4debtus.DebtusContactBrief{
			"c1": {Status: models4debtus.DebtusContactStatusActive},
		}
		seedRecords(t, db, contact.Record, debtusSpace.Record)

		if err := DeleteContact(ctx, userCtx, testSpaceID, "c1"); err != nil {
			t.Fatalf("DeleteContact() returned error: %v", err)
		}
		deleted := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", nil)
		if err := db.Get(ctx, deleted.Record); !dal.IsNotFound(err) {
			t.Errorf("contact record should be deleted, got err: %v", err)
		}
	})

	t.Run("DeleteContactTx_works_within_callers_transaction", func(t *testing.T) {
		// Mirrors the usage in maintainance/merge-contacts.go that passes an
		// existing transaction.
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		seedRecords(t, db, contact.Record, debtusSpace.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return DeleteContactTx(ctx, userCtx, tx, testSpaceID, "c1")
		}, dal.TxWithCrossGroup())
		if err != nil {
			t.Fatalf("DeleteContactTx() returned error: %v", err)
		}
		deleted := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", nil)
		if err = db.Get(ctx, deleted.Record); !dal.IsNotFound(err) {
			t.Errorf("contact record should be deleted, got err: %v", err)
		}
	})

	t.Run("balance_mismatch_with_space_brief", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		debtusSpace.Data.Contacts = map[string]*models4debtus.DebtusContactBrief{
			"c1": {Status: models4debtus.DebtusContactStatusActive, Balance: money.Balance{"EUR": 100}},
		}
		seedRecords(t, db, contact.Record, debtusSpace.Record)

		err := DeleteContact(ctx, userCtx, testSpaceID, "c1")
		if err == nil || !strings.Contains(err.Error(), "data integrity") {
			t.Errorf("expected data integrity error, got: %v", err)
		}
	})

	t.Run("non_zero_balance_is_not_implemented", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status:   models4debtus.DebtusContactStatusActive,
			Balanced: money.Balanced{Balance: money.Balance{"EUR": 100}},
		})
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		debtusSpace.Data.Contacts = map[string]*models4debtus.DebtusContactBrief{
			"c1": {Status: models4debtus.DebtusContactStatusActive, Balance: money.Balance{"EUR": 100}},
		}
		seedRecords(t, db, contact.Record, debtusSpace.Record)

		err := DeleteContact(ctx, userCtx, testSpaceID, "c1")
		if err == nil || !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("expected 'not implemented' error, got: %v", err)
		}
	})
}

func TestChangeContactStatus(t *testing.T) {
	ctx := facade.NewContextWithUserID(context.Background(), "u1")
	db := sneattesting.SetupMemoryDB(t)

	space := dbo4spaceus.NewSpaceEntry(testSpaceID)
	space.Data.UserIDs = []string{"u1"}
	contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
		Status: models4debtus.DebtusContactStatusActive,
	})
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	debtusSpace.Data.Contacts = map[string]*models4debtus.DebtusContactBrief{
		"c1": {Status: models4debtus.DebtusContactStatusActive},
	}
	seedRecords(t, db, space.Record, contact.Record, debtusSpace.Record)

	_, debtusContact, err := ChangeContactStatus(ctx, testSpaceID, "c1", models4debtus.DebtusContactStatusArchived)
	if err != nil {
		t.Fatalf("ChangeContactStatus() returned error: %v", err)
	}
	if debtusContact.Data.Status != models4debtus.DebtusContactStatusArchived {
		t.Errorf("returned contact status = %v, want archived", debtusContact.Data.Status)
	}

	saved := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", nil)
	if err = db.Get(context.Background(), saved.Record); err != nil {
		t.Fatalf("failed to read saved contact: %v", err)
	}
	if saved.Data.Status != models4debtus.DebtusContactStatusArchived {
		t.Errorf("saved contact status = %v, want archived", saved.Data.Status)
	}

	savedSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	if err = db.Get(context.Background(), savedSpace.Record); err != nil {
		t.Fatalf("failed to read saved debtus space: %v", err)
	}
	if got := savedSpace.Data.Contacts["c1"].Status; got != models4debtus.DebtusContactStatusArchived {
		t.Errorf("debtus space brief status = %v, want archived", got)
	}

	// Second call with the same status must be a no-op without error.
	if _, _, err = ChangeContactStatus(ctx, testSpaceID, "c1", models4debtus.DebtusContactStatusArchived); err != nil {
		t.Errorf("no-op ChangeContactStatus() returned error: %v", err)
	}
}
