package facade4debtus

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus-ext/backend/contactusmodels/briefs4contactus"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/contactus/backend/dto4contactus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/strongo/strongoapp/person"
)

// insertingContactDal is a dal4debtus.ContactDal whose InsertContact actually
// persists the contact (with a caller-chosen ID) so the deeper body of
// createContactWithinTransaction can be exercised, unlike fakeContactDal whose
// InsertContact only returns an error.
type insertingContactDal struct {
	fakeContactDal
	newID string
}

func (d insertingContactDal) InsertContact(ctx context.Context, tx dal.ReadwriteTransaction, dbo *models4debtus.DebtusSpaceContactDbo) (models4debtus.DebtusSpaceContactEntry, error) {
	contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, d.newID, dbo)
	if err := tx.Set(ctx, contact.Record); err != nil {
		return models4debtus.DebtusSpaceContactEntry{}, err
	}
	return contact, nil
}

// briefStub returns a minimal valid contact brief for use in contactus spaces.
func briefStub() *briefs4contactus.ContactBrief {
	return &briefs4contactus.ContactBrief{
		Names: &person.NameFields{FirstName: "X"},
	}
}

// ============================================================================
// createContactWithinTransaction — happy path with no counterparty.
// This drives the full body past the early asserts (InsertContact,
// AddOrUpdateDebtusContact, data-integrity verifications), which the existing
// tests do not reach (they stop at the assert errors).
// ============================================================================

func TestCreateContactWithinTransaction_NoCounterparty_HappyPath(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	orig := dal4debtus.Default.Contact
	dal4debtus.Default.Contact = insertingContactDal{newID: "newContact1"}
	t.Cleanup(func() { dal4debtus.Default.Contact = orig })

	appUser := dbo4userus.NewUserEntry("u1")
	appUser.Data.Names = &person.NameFields{FirstName: "Alice"}

	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	debtusSpace.Data.Contacts = make(map[string]*models4debtus.DebtusContactBrief)

	// creator.Contact must have Data.UserID == appUser.ID for the integrity check.
	creatorContact := dal4contactus.NewContactEntryWithData(testSpaceID, "newContact1", &dbo4contactus.ContactDbo{})
	creatorContact.Data.UserID = "u1"

	changes := &createContactDbChanges{
		user:        appUser,
		debtusSpace: debtusSpace,
	}
	changes.creator.Contact = creatorContact

	details := dto4contactus.ContactDetails{NameFields: person.NameFields{FirstName: "Bob"}}

	var got error
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		got = createContactWithinTransaction(ctx, tx, changes, testSpaceID, "", details)
		return nil
	}, dal.TxWithCrossGroup())

	if got != nil {
		t.Fatalf("createContactWithinTransaction() returned error: %v", got)
	}
	// createContactWithinTransaction operates on a local copy of changes.creator,
	// so its mutations land in the persisted record and in debtusSpace's JSON,
	// not on changes.creator itself.
	brief, ok := debtusSpace.Data.Contacts["newContact1"]
	if !ok {
		t.Fatal("expected debtusSpace.Data.Contacts to contain newContact1")
	}
	if brief == nil {
		t.Fatal("expected non-nil brief for newContact1")
	}
	// Verify the contact was actually persisted by InsertContact.
	persisted, err := GetDebtusSpaceContactByID(ctx, nil, testSpaceID, "newContact1")
	if err != nil {
		t.Fatalf("expected persisted contact, got error: %v", err)
	}
	if persisted.Data.CreatedBy != "u1" {
		t.Errorf("persisted CreatedBy = %q, want u1", persisted.Data.CreatedBy)
	}
}

func TestCreateContactWithinTransaction_NilTx(t *testing.T) {
	changes := &createContactDbChanges{
		user:        dbo4userus.NewUserEntry("u1"),
		debtusSpace: models4debtus.NewDebtusSpaceEntry(testSpaceID),
	}
	err := createContactWithinTransaction(context.Background(), nil, changes, testSpaceID, "u2", dto4contactus.ContactDetails{})
	if err == nil || err.Error() != "tx == nil" {
		t.Errorf("expected 'tx == nil' error, got: %v", err)
	}
}

func TestCreateContactWithinTransaction_AppUserDataNil(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// appUser with ID set but Data nil => "appUser.Data == nil" branch.
	appUser := dbo4userus.UserEntry{}
	appUser.ID = "u1"

	changes := &createContactDbChanges{
		user:        appUser,
		debtusSpace: models4debtus.NewDebtusSpaceEntry(testSpaceID),
	}
	var got error
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		got = createContactWithinTransaction(ctx, tx, changes, testSpaceID, "u2", dto4contactus.ContactDetails{})
		return nil
	}, dal.TxWithCrossGroup())
	if got == nil || got.Error() != "appUser.Data == nil" {
		t.Errorf("expected 'appUser.Data == nil' error, got: %v", got)
	}
}

// ============================================================================
// workaroundReinsertContact
// ============================================================================

func TestWorkaroundReinsertContact(t *testing.T) {
	ctx := context.Background()

	newChanges := func() *receiptDbChanges {
		c := newReceiptDbChanges()
		c.inviter = &userLinkingParty{}
		c.inviter.contactusSpace = dal4contactus.NewContactusSpaceEntry(testSpaceID)
		c.invited = &userLinkingParty{}
		c.invited.contact = dal4contactus.NewContactEntryWithData(testSpaceID, "inv1", &dbo4contactus.ContactDbo{})
		return c
	}

	t.Run("contact_found_is_noop", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		existing := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})
		seedRecords(t, db, existing.Record)

		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{SpaceID: testSpaceID})
		invitedContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})
		changes := newChanges()

		if err := workaroundReinsertContact(ctx, receipt, invitedContact, changes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changes.IsChanged(changes.invited.contact.Record) {
			t.Error("expected invited.contact NOT to be flagged when contact found")
		}
	})

	t.Run("contact_not_found_flags_invited_contact", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)

		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{SpaceID: testSpaceID})
		invitedContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "missing", &models4debtus.DebtusSpaceContactDbo{})
		changes := newChanges()

		if err := workaroundReinsertContact(ctx, receipt, invitedContact, changes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changes.IsChanged(changes.invited.contact.Record) {
			t.Error("expected invited.contact to be flagged as changed when contact not found")
		}
	})

	t.Run("contact_not_found_acknowledged_with_contact_info", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)

		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			SpaceID: testSpaceID,
			Status:  models4debtus.ReceiptStatusAcknowledged,
		})
		invitedContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "missing", &models4debtus.DebtusSpaceContactDbo{})
		changes := newChanges()
		// inviter.contactusSpace has the brief for the invitedContact.ID so the
		// "invitedUser already has the Contact info" branch runs.
		changes.inviter.contactusSpace.Data.AddContact("missing", briefStub())

		if err := workaroundReinsertContact(ctx, receipt, invitedContact, changes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changes.invited.debtusContact.ID != "missing" {
			t.Errorf("expected invited.debtusContact reassigned to invitedContact, got ID %q", changes.invited.debtusContact.ID)
		}
	})
}

// ============================================================================
// usersLinker.updateInviterContact — pure-mutation branches.
// ============================================================================

func TestUsersLinker_updateInviterContact(t *testing.T) {
	ctx := context.Background()

	newParty := func(userID, contactID string) *userLinkingParty {
		p := &userLinkingParty{spaceID: testSpaceID}
		p.user = dbo4userus.NewUserEntry(userID)
		p.user.Data.Names = &person.NameFields{FirstName: "U" + userID}
		p.contact = dal4contactus.NewContactEntryWithData(testSpaceID, contactID, &dbo4contactus.ContactDbo{})
		p.contact.Data.UserID = userID
		p.contact.Data.Names = &person.NameFields{}
		p.contactusSpace = dal4contactus.NewContactusSpaceEntry(testSpaceID)
		p.debtusSpace = models4debtus.NewDebtusSpaceEntry(testSpaceID)
		p.debtusSpace.Data.Contacts = make(map[string]*models4debtus.DebtusContactBrief)
		p.debtusContact = models4debtus.NewDebtusSpaceContactEntry(testSpaceID, contactID, &models4debtus.DebtusSpaceContactDbo{})
		return p
	}

	// NOTE: the switch arms `case ""` and `case invited.user.ID` inside
	// updateInviterContact are unreachable in isolation: the validation guard just
	// above the switch requires inviter.contact.Data.UserID == inviter.user.ID, and
	// inviter.user.ID != invited.user.ID, so the switch (keyed on
	// inviter.contact.Data.UserID) can only take the `default` arm — which returns
	// a data-integrity error. The new-link "" arm is only reachable from the full
	// linkUsersWithinTransaction orchestration (integration-only, documented gap).

	t.Run("inviter_contact_already_links_to_third_user_returns_error", func(t *testing.T) {
		// inviter.contact.UserID == inviter.user.ID (passes guard) but is a
		// different user than invited.user.ID => default switch arm => error.
		linker := newUsersLinker(newUsersLinkingDbChanges())

		inviter := newParty("u1", "c1") // contact.UserID == u1 == user.ID
		invited := newParty("u2", "c2")

		_, err := linker.updateInviterContact(ctx, inviter, invited, "lb")
		if err == nil {
			t.Fatal("expected data-integrity error when inviter contact already linked to a different user")
		}
	})

	t.Run("copies_missing_names_from_invited_user_before_error", func(t *testing.T) {
		// The name-copy logic runs before the switch; verify it copies names from
		// the invited user when the inviter contact has none.
		linker := newUsersLinker(newUsersLinkingDbChanges())

		inviter := newParty("u1", "c1")
		inviter.contact.Data.Names = &person.NameFields{} // empty names
		invited := newParty("u2", "c2")
		invited.user.Data.Names = &person.NameFields{FirstName: "Bob", LastName: "Builder"}

		// default arm still returns an error, but the names are copied first.
		_, _ = linker.updateInviterContact(ctx, inviter, invited, "lb")
		if inviter.contact.Data.Names.FirstName != "Bob" {
			t.Errorf("FirstName = %q, want Bob (copied from invited user)", inviter.contact.Data.Names.FirstName)
		}
		if inviter.contact.Data.Names.LastName != "Builder" {
			t.Errorf("LastName = %q, want Builder", inviter.contact.Data.Names.LastName)
		}
		if !linker.changes.IsChanged(inviter.contact.Record) {
			t.Error("expected inviter.contact flagged as changed after name copy")
		}
	})

	t.Run("inviter_contact_user_mismatch_panics", func(t *testing.T) {
		// inviter.contact.Data.UserID != inviter.user.ID => guard panics.
		linker := newUsersLinker(newUsersLinkingDbChanges())
		inviter := newParty("u1", "c1")
		inviter.contact.Data.UserID = "uX" // != inviter.user.ID
		invited := newParty("u2", "c2")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for inviter contact UserID mismatch")
			}
		}()
		_, _ = linker.updateInviterContact(ctx, inviter, invited, "lb")
	})

	t.Run("empty_inviter_user_id_returns_error", func(t *testing.T) {
		linker := newUsersLinker(newUsersLinkingDbChanges())
		inviter := newParty("u1", "c1")
		inviter.user = dbo4userus.UserEntry{} // ID == ""
		invited := newParty("u2", "c2")
		if _, err := linker.updateInviterContact(ctx, inviter, invited, "lb"); err == nil {
			t.Error("expected error for empty inviter.user.ID")
		}
	})
}
