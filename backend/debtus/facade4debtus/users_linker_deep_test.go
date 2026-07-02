package facade4debtus

import (
	"context"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/contactus/backend/dto4contactus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/strongoapp/person"
)

// giveFamilySpace registers a family-type space brief on the user so that
// dbo4userus.UserDbo.GetFamilySpaceID() returns the given space ID.
func giveFamilySpace(user dbo4userus.UserEntry, spaceID coretypes.SpaceID) {
	if user.Data.Spaces == nil {
		user.Data.Spaces = make(map[string]*dbo4userus.UserSpaceBrief)
	}
	user.Data.Spaces[string(spaceID)] = &dbo4userus.UserSpaceBrief{
		SpaceBrief: dbo4spaceus.SpaceBrief{Type: coretypes.SpaceTypeFamily},
	}
}

// newLinkParty builds a userLinkingParty with all sub-entities present and ZERO
// balances, so models4debtus.MustMatchCounterparty (which panics only on a
// balance mismatch) is satisfied for both directions.
func newLinkParty(userID, contactID string) *userLinkingParty {
	p := &userLinkingParty{spaceID: testSpaceID}
	p.user = dbo4userus.NewUserEntry(userID)
	p.user.Data.Names = &person.NameFields{FirstName: "U" + userID}
	p.contact = dal4contactus.NewContactEntryWithData(testSpaceID, contactID, &dbo4contactus.ContactDbo{})
	p.contact.Data.UserID = userID
	p.contact.Data.Names = &person.NameFields{FirstName: "C" + contactID}
	p.contactusSpace = dal4contactus.NewContactusSpaceEntry(testSpaceID)
	p.debtusSpace = models4debtus.NewDebtusSpaceEntry(testSpaceID)
	p.debtusSpace.Data.Contacts = make(map[string]*models4debtus.DebtusContactBrief)
	p.debtusContact = models4debtus.NewDebtusSpaceContactEntry(testSpaceID, contactID, &models4debtus.DebtusSpaceContactDbo{})
	return p
}

// ============================================================================
// linkUsersWithinTransaction — drives the deep orchestration:
// getOrCreateInvitedContactByInviterUserAndInviterContact (new-contact branch)
// -> createContactWithinTransaction (counterparty path) -> updateInvitedUser
// -> updateInviterContact. With a normal (already-self-linked) inviter contact,
// the updateInviterContact switch lands on the data-integrity default arm and
// returns an error — but the whole creation + balance-matching path runs first.
// ============================================================================

func TestLinkUsersWithinTransaction_DeepPath(t *testing.T) {
	ctx := context.Background()

	withInsertingDal := func(t *testing.T, newID string) {
		t.Helper()
		orig := dal4debtus.Default.Contact
		dal4debtus.Default.Contact = insertingContactDal{newID: newID}
		t.Cleanup(func() { dal4debtus.Default.Contact = orig })
	}

	t.Run("creates_invited_contact_then_inviter_link_conflict", func(t *testing.T) {
		initDelayers4Test(t)
		db := sneattesting.SetupMemoryDB(t)
		withInsertingDal(t, "invitedNew")

		changes := newUsersLinkingDbChanges()
		changes.inviter = newLinkParty("u1", "c1") // self-linked: contact.UserID == u1
		changes.invited = newLinkParty("u2", "c2")
		// invited's contactus space has no contact referencing u1, so
		// getOrCreateInvitedContactByInviterUserAndInviterContact takes the
		// "create new invited contact" branch.
		linker := newUsersLinker(changes)

		var got error
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			got = linker.linkUsersWithinTransaction(ctx, tx, "linkedBy")
			return nil
		}, dal.TxWithCrossGroup())

		if got == nil {
			t.Fatal("expected data-integrity error from updateInviterContact default arm")
		}
		if !strings.Contains(got.Error(), "different from current user") {
			t.Errorf("unexpected error: %v", got)
		}
		// The invited contact WAS created via createContactWithinTransaction.
		if changes.invited.debtusContact.ID == "" {
			t.Error("expected invited.debtusContact to have been created with a non-empty ID")
		}
		// invitedUser got linked back to inviter.
		if changes.invited.user.Data.InvitedByUserID != "u1" {
			t.Errorf("invited.user.InvitedByUserID = %q, want u1", changes.invited.user.Data.InvitedByUserID)
		}
	})

	t.Run("empty_inviter_user_id_returns_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		changes := newUsersLinkingDbChanges()
		changes.inviter = newLinkParty("u1", "c1")
		changes.inviter.user = dbo4userus.UserEntry{} // ID == ""
		changes.invited = newLinkParty("u2", "c2")
		linker := newUsersLinker(changes)

		var got error
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			got = linker.linkUsersWithinTransaction(ctx, tx, "lb")
			return nil
		}, dal.TxWithCrossGroup())
		if got == nil || !strings.Contains(got.Error(), "inviter.user.ID is empty") {
			t.Errorf("expected inviter.user.ID empty error, got: %v", got)
		}
	})

	t.Run("empty_invited_user_id_returns_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		changes := newUsersLinkingDbChanges()
		changes.inviter = newLinkParty("u1", "c1")
		changes.invited = newLinkParty("u2", "c2")
		changes.invited.user = dbo4userus.UserEntry{} // ID == ""
		linker := newUsersLinker(changes)

		var got error
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			got = linker.linkUsersWithinTransaction(ctx, tx, "lb")
			return nil
		}, dal.TxWithCrossGroup())
		if got == nil || !strings.Contains(got.Error(), "invited.user.ID is empty") {
			t.Errorf("expected invited.user.ID empty error, got: %v", got)
		}
	})

	t.Run("nil_tx_returns_error", func(t *testing.T) {
		changes := newUsersLinkingDbChanges()
		changes.inviter = newLinkParty("u1", "c1")
		changes.invited = newLinkParty("u2", "c2")
		linker := newUsersLinker(changes)
		got := linker.linkUsersWithinTransaction(ctx, nil, "lb")
		if got == nil || !strings.Contains(got.Error(), "without transaction") {
			t.Errorf("expected without-transaction error, got: %v", got)
		}
	})
}

// ============================================================================
// getOrCreateInvitedContactByInviterUserAndInviterContact — the "link existing
// invited contact" branch: the invited user's contactus space already has a
// contact whose UserID == inviter.user.ID, so the function re-loads that
// debtus contact from the DB and copies inviter's counterparty fields into it.
// ============================================================================

func TestGetOrCreateInvitedContact_LinkExisting(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	changes := newUsersLinkingDbChanges()
	changes.inviter = newLinkParty("u1", "c1")
	changes.invited = newLinkParty("u2", "c2")

	// Seed an existing invited-side debtus contact that references the inviter.
	existingInvitedContactID := "existingInvited"
	existingInvited := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, existingInvitedContactID, &models4debtus.DebtusSpaceContactDbo{})
	seedRecords(t, db, existingInvited.Record)

	// Register it in invited.contactusSpace with UserID == inviter.user.ID so the
	// matching loop finds it.
	brief := briefStub()
	brief.UserID = "u1"
	changes.invited.contactusSpace.Data.AddContact(existingInvitedContactID, brief)

	// invited.contact must have Names populated (the branch reads/writes them).
	changes.invited.contact.Data.Names = &person.NameFields{}

	linker := newUsersLinker(changes)

	var got error
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		got = linker.getOrCreateInvitedContactByInviterUserAndInviterContact(ctx, tx, changes)
		return nil
	}, dal.TxWithCrossGroup())

	if got != nil {
		t.Fatalf("unexpected error: %v", got)
	}
	// The existing invited debtus contact was loaded and adopted.
	if changes.invited.debtusContact.ID != existingInvitedContactID {
		t.Errorf("invited.debtusContact.ID = %q, want %q", changes.invited.debtusContact.ID, existingInvitedContactID)
	}
	// Inviter's first name was copied onto the (empty) invited contact name.
	if changes.invited.contact.Data.Names.FirstName != "Uu1" {
		t.Errorf("invited.contact FirstName = %q, want Uu1 (copied from inviter user)", changes.invited.contact.Data.Names.FirstName)
	}
}

func TestGetOrCreateInvitedContact_SameUserPanics(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	changes := newUsersLinkingDbChanges()
	changes.inviter = newLinkParty("u1", "c1")
	changes.invited = newLinkParty("u1", "c2") // same user ID as inviter
	linker := newUsersLinker(changes)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when inviter and invited are the same user")
		}
	}()
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return linker.getOrCreateInvitedContactByInviterUserAndInviterContact(ctx, tx, changes)
	}, dal.TxWithCrossGroup())
}

// ============================================================================
// createContactWithinTransaction — counterparty path. The creator contact is
// linked to an unlinked counterparty contact (zero balances), exercising the
// balance-reversal, MustMatchCounterparty and counterparty back-link branches.
// The standalone call ends in the final integrity check because the function is
// designed to run with a counterpartyUserID matching the linked contact; we
// assert the linking side effects that occur before that check.
// ============================================================================

func TestCreateContactWithinTransaction_CounterpartyPath(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	orig := dal4debtus.Default.Contact
	dal4debtus.Default.Contact = insertingContactDal{newID: "creatorC"}
	t.Cleanup(func() { dal4debtus.Default.Contact = orig })

	appUser := dbo4userus.NewUserEntry("u1")
	appUser.Data.Names = &person.NameFields{FirstName: "Alice"}

	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	debtusSpace.Data.Contacts = make(map[string]*models4debtus.DebtusContactBrief)

	creatorContact := dal4contactus.NewContactEntryWithData(testSpaceID, "creatorC", &dbo4contactus.ContactDbo{})
	creatorContact.Data.UserID = "u1"

	cpContact := dal4contactus.NewContactEntryWithData(testSpaceID, "cpC", &dbo4contactus.ContactDbo{})
	cpContact.Data.UserID = "" // unlinked counterparty => gets linked to creator
	cpDebtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "cpC", &models4debtus.DebtusSpaceContactDbo{})

	changes := &createContactDbChanges{
		user:        appUser,
		debtusSpace: debtusSpace,
	}
	changes.creator.Contact = creatorContact
	changes.counterparty.Contact = cpContact
	changes.counterparty.DebtusContact = cpDebtusContact

	var got error
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		got = createContactWithinTransaction(ctx, tx, changes, "", dto4contactus.ContactDetails{NameFields: person.NameFields{FirstName: "Bob"}})
		return nil
	}, dal.TxWithCrossGroup())

	// The counterparty back-link side effects happen before the terminal check.
	if cpContact.Data.UserID != "u1" {
		t.Errorf("counterparty contact UserID = %q, want u1 (linked to creator)", cpContact.Data.UserID)
	}
	if cpDebtusContact.Data.CounterpartyContactID != "creatorC" {
		t.Errorf("counterparty CounterpartyContactID = %q, want creatorC", cpDebtusContact.Data.CounterpartyContactID)
	}
	// The standalone call terminates at the final integrity check.
	if got == nil {
		t.Fatal("expected terminal integrity error for standalone counterparty call")
	}
}
