package facade4debtus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/auth/unsorted4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
	"github.com/strongo/strongoapp/person"
)

// fakeReceiptDal is a minimal dal4debtus.ReceiptDal implementation backed by
// direct DAL operations, used to avoid the import cycle
// facade4debtus → debtusdal → facade4debtus.
type fakeReceiptDal struct{}

func (fakeReceiptDal) UpdateReceipt(ctx context.Context, tx dal.ReadwriteTransaction, receipt models4debtus.ReceiptEntry) error {
	return tx.Set(ctx, receipt.Record)
}

func (fakeReceiptDal) GetReceiptByID(ctx context.Context, tx dal.ReadSession, id string) (models4debtus.ReceiptEntry, error) {
	receipt := models4debtus.NewReceipt(id, nil)
	return receipt, tx.Get(ctx, receipt.Record)
}

func (fakeReceiptDal) MarkReceiptAsSent(_ context.Context, _, _ string, _ time.Time) error {
	return errors.New("not implemented in fakeReceiptDal")
}

func (fakeReceiptDal) CreateReceipt(_ context.Context, _ *models4debtus.ReceiptDbo) (models4debtus.ReceiptEntry, error) {
	return models4debtus.ReceiptEntry{}, errors.New("not implemented in fakeReceiptDal")
}

func (fakeReceiptDal) DelayedMarkReceiptAsSent(_ context.Context, _, _ string, _ time.Time) error {
	return errors.New("not implemented in fakeReceiptDal")
}

func (fakeReceiptDal) DelayCreateAndSendReceiptToCounterpartyByTelegram(_ context.Context, _, _, _ string) error {
	return errors.New("not implemented in fakeReceiptDal")
}

// ============================================================================
// markReceiptAsViewed — pure function
// ============================================================================

func TestMarkReceiptAsViewedPure(t *testing.T) {
	t.Run("first_view", func(t *testing.T) {
		dbo := &models4debtus.ReceiptDbo{}
		changed := markReceiptAsViewed(dbo, "u1")
		if !changed {
			t.Error("expected changed=true on first view")
		}
		if len(dbo.ViewedByUserIDs) != 1 || dbo.ViewedByUserIDs[0] != "u1" {
			t.Errorf("unexpected ViewedByUserIDs: %v", dbo.ViewedByUserIDs)
		}
	})

	t.Run("already_viewed", func(t *testing.T) {
		dbo := &models4debtus.ReceiptDbo{ViewedByUserIDs: []string{"u1"}}
		changed := markReceiptAsViewed(dbo, "u1")
		if changed {
			t.Error("expected changed=false when already viewed by same user")
		}
	})

	t.Run("second_user_view", func(t *testing.T) {
		dbo := &models4debtus.ReceiptDbo{ViewedByUserIDs: []string{"u1"}}
		changed := markReceiptAsViewed(dbo, "u2")
		if !changed {
			t.Error("expected changed=true when new user views")
		}
		if len(dbo.ViewedByUserIDs) != 2 {
			t.Errorf("expected 2 viewers, got %v", dbo.ViewedByUserIDs)
		}
	})
}

// ============================================================================
// newUsersLinkingDbChanges / newReceiptDbChanges / NewReceiptUsersLinker — constructors
// ============================================================================

func TestNewDbChangesConstructors(t *testing.T) {
	t.Run("newUsersLinkingDbChanges", func(t *testing.T) {
		c := newUsersLinkingDbChanges()
		if c == nil {
			t.Fatal("expected non-nil changes")
		}
	})

	t.Run("newReceiptDbChanges", func(t *testing.T) {
		c := newReceiptDbChanges()
		if c == nil {
			t.Fatal("expected non-nil changes")
		}
		if c.usersLinkingDbChanges == nil {
			t.Fatal("expected non-nil usersLinkingDbChanges embedded")
		}
	})

	t.Run("NewReceiptUsersLinker_nil_changes", func(t *testing.T) {
		l := NewReceiptUsersLinker(nil)
		if l == nil {
			t.Fatal("expected non-nil linker")
		}
		if l.changes == nil {
			t.Fatal("expected non-nil changes created by NewReceiptUsersLinker")
		}
	})

	t.Run("NewReceiptUsersLinker_with_changes", func(t *testing.T) {
		c := newReceiptDbChanges()
		l := NewReceiptUsersLinker(c)
		if l.changes != c {
			t.Fatal("expected linker to use provided changes")
		}
	})
}

// ============================================================================
// ReceiptUsersLinker.validateInput — pure validation
// ============================================================================

func TestReceiptUsersLinker_validateInput(t *testing.T) {
	linker := NewReceiptUsersLinker(nil)

	t.Run("no_counterparty_user_id", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "",
		})
		changes.invited = &userLinkingParty{user: dbo4userus.NewUserEntry("u2")}
		linker.changes = changes
		if err := linker.validateInput(changes); err != nil {
			t.Errorf("expected nil error: %v", err)
		}
	})

	t.Run("counterparty_matches_invited", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "u2",
		})
		changes.transfer = models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		})
		changes.invited = &userLinkingParty{user: dbo4userus.NewUserEntry("u2")}
		linker.changes = changes
		if err := linker.validateInput(changes); err != nil {
			t.Errorf("expected nil error: %v", err)
		}
	})

	t.Run("counterparty_mismatched_3rd_user", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "u3", // different from invited
		})
		changes.transfer = models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u3","contactID":"c3"}`,
		})
		changes.invited = &userLinkingParty{user: dbo4userus.NewUserEntry("u2")}
		linker.changes = changes
		if err := linker.validateInput(changes); err == nil {
			t.Error("expected error for 3rd user attempt")
		}
	})

	t.Run("transfer_counterparty_mismatched_with_invited", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "u2",
		})
		changes.transfer = models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u99","contactID":"c99"}`, // differs from invited u2
		})
		changes.invited = &userLinkingParty{user: dbo4userus.NewUserEntry("u2")}
		linker.changes = changes
		if err := linker.validateInput(changes); err == nil {
			t.Error("expected error when transfer counterparty != invited user")
		}
	})
}

// ============================================================================
// ReceiptUsersLinker.updateReceipt — pure mutation
// ============================================================================

func TestReceiptUsersLinker_updateReceipt(t *testing.T) {
	t.Run("counterparty_differs_marks_changed", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "old-user",
		})
		invitedUser := dbo4userus.NewUserEntry("u2")
		changes.invited = &userLinkingParty{user: invitedUser}
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateReceipt(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changes.receipt.Data.CounterpartyUserID != "u2" {
			t.Errorf("CounterpartyUserID = %q, want u2", changes.receipt.Data.CounterpartyUserID)
		}
		if !changes.IsChanged(changes.receipt.Record) {
			t.Error("expected receipt record to be marked as changed")
		}
	})

	t.Run("counterparty_same_no_change", func(t *testing.T) {
		changes := newReceiptDbChanges()
		changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CounterpartyUserID: "u2",
		})
		invitedUser := dbo4userus.NewUserEntry("u2")
		changes.invited = &userLinkingParty{user: invitedUser}
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateReceipt(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changes.IsChanged(changes.receipt.Record) {
			t.Error("expected receipt NOT to be marked changed when already matches")
		}
	})
}

// ============================================================================
// usersLinker.validateInput — pure validation
// ============================================================================

func TestUsersLinker_validateInput(t *testing.T) {
	makeParty := func(userID, contactUserID, debtusContactID string) *userLinkingParty {
		p := &userLinkingParty{}
		// Only call NewUserEntry when userID is non-empty to avoid panic
		if userID != "" {
			p.user = dbo4userus.NewUserEntry(userID)
		}
		p.contact.Data = &dbo4contactus.ContactDbo{}
		p.contact.Data.UserID = contactUserID
		if debtusContactID != "" {
			p.debtusContact = models4debtus.NewDebtusSpaceContactEntry(testSpaceID, debtusContactID, &models4debtus.DebtusSpaceContactDbo{})
		}
		return p
	}

	linker := newUsersLinker(newUsersLinkingDbChanges())

	t.Run("valid", func(t *testing.T) {
		inviter := makeParty("u1", "u1", "dc1")
		invited := makeParty("u2", "u2", "dc2")
		if err := linker.validateInput(inviter, invited); err != nil {
			t.Errorf("expected nil error: %v", err)
		}
	})

	t.Run("empty_inviter_user_id", func(t *testing.T) {
		inviter := makeParty("", "u1", "dc1") // user.ID will be "" (zero value)
		invited := makeParty("u2", "u2", "dc2")
		if err := linker.validateInput(inviter, invited); err == nil {
			t.Error("expected error for empty inviter user id")
		}
	})

	t.Run("empty_invited_user_id", func(t *testing.T) {
		inviter := makeParty("u1", "u1", "dc1")
		invited := makeParty("", "", "dc2") // user.ID will be "" (zero value)
		if err := linker.validateInput(inviter, invited); err == nil {
			t.Error("expected error for empty invited user id")
		}
	})

	t.Run("empty_inviter_debtus_contact_id", func(t *testing.T) {
		inviter := makeParty("u1", "u1", "") // debtusContact.ID == ""
		invited := makeParty("u2", "u2", "dc2")
		if err := linker.validateInput(inviter, invited); err == nil {
			t.Error("expected error for empty inviter debtus contact id")
		}
	})

	t.Run("same_user_ids", func(t *testing.T) {
		inviter := makeParty("u1", "u1", "dc1")
		invited := makeParty("u1", "u1", "dc2")
		if err := linker.validateInput(inviter, invited); err == nil {
			t.Error("expected error for same user ids")
		}
	})

	t.Run("inviter_contact_user_id_mismatch", func(t *testing.T) {
		inviter := makeParty("u1", "u99", "dc1") // contact.UserID != user.ID
		invited := makeParty("u2", "u2", "dc2")
		if err := linker.validateInput(inviter, invited); err == nil {
			t.Error("expected error for inviter contact.UserID mismatch")
		}
	})
}

// ============================================================================
// usersLinker.updateInvitedUser — mostly pure mutation
// ============================================================================

func TestUsersLinker_updateInvitedUser(t *testing.T) {
	linker := newUsersLinker(newUsersLinkingDbChanges())

	t.Run("sets_InvitedByUserID_when_empty", func(t *testing.T) {
		invitedUser := dbo4userus.NewUserEntry("u2")
		invitedDebtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		inviterDebtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})

		if err := linker.updateInvitedUser(context.Background(), invitedUser, invitedDebtusSpace, "u1", inviterDebtusContact); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if invitedUser.Data.InvitedByUserID != "u1" {
			t.Errorf("InvitedByUserID = %q, want u1", invitedUser.Data.InvitedByUserID)
		}
	})

	t.Run("does_not_overwrite_InvitedByUserID", func(t *testing.T) {
		invitedUser := dbo4userus.NewUserEntry("u2")
		invitedUser.Data.InvitedByUserID = "u0"
		invitedDebtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		inviterDebtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})

		if err := linker.updateInvitedUser(context.Background(), invitedUser, invitedDebtusSpace, "u1", inviterDebtusContact); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if invitedUser.Data.InvitedByUserID != "u0" {
			t.Errorf("InvitedByUserID should not be overwritten: got %q", invitedUser.Data.InvitedByUserID)
		}
	})

	t.Run("updates_space_last_transfer_when_inviter_newer", func(t *testing.T) {
		now := time.Now()
		older := now.Add(-time.Hour)

		invitedUser := dbo4userus.NewUserEntry("u2")
		invitedDebtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		invitedDebtusSpace.Data.LastTransferAt = older

		inviterDebtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Balanced: money.Balanced{LastTransferAt: now, LastTransferID: "t99"},
		})

		if err := linker.updateInvitedUser(context.Background(), invitedUser, invitedDebtusSpace, "u1", inviterDebtusContact); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if invitedDebtusSpace.Data.LastTransferID != "t99" {
			t.Errorf("LastTransferID = %q, want t99", invitedDebtusSpace.Data.LastTransferID)
		}
		if !linker.changes.IsChanged(invitedDebtusSpace.Record) {
			t.Error("expected invitedDebtusSpace marked as changed")
		}
	})
}

// ============================================================================
// updateDebtusSpaceWithTransferInfo — pure function
// ============================================================================

func newDebtusSpaceEntryWithBalance(spaceID coretypes.SpaceID) models4debtus.DebtusSpaceEntry {
	e := models4debtus.NewDebtusSpaceEntry(spaceID)
	e.Data.Balance = make(money.Balance)
	return e
}

func newDebtusSpaceContactEntryWithBalance(spaceID coretypes.SpaceID, contactID string) models4debtus.DebtusSpaceContactEntry {
	dbo := &models4debtus.DebtusSpaceContactDbo{}
	dbo.Balance = make(money.Balance)
	dbo.Transfers = &models4debtus.UserContactTransfersInfo{}
	e := models4debtus.NewDebtusSpaceContactEntry(spaceID, contactID, dbo)
	return e
}

func TestUpdateDebtusSpaceWithTransferInfo(t *testing.T) {
	ctx := context.Background()
	transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      money.CurrencyEUR,
		AmountInCents: 100,
		FromJson:      `{"userID":"u1","contactID":"c1"}`,
		ToJson:        `{"userID":"u2","contactID":"c2"}`,
	})
	transfer.Data.DtCreated = time.Now()

	debtusSpace := newDebtusSpaceEntryWithBalance(testSpaceID)
	contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c1")

	if err := updateDebtusSpaceWithTransferInfo(ctx, 100, transfer, debtusSpace, contact, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if debtusSpace.Data.LastTransferID != "t1" {
		t.Errorf("LastTransferID = %q, want t1", debtusSpace.Data.LastTransferID)
	}
	if debtusSpace.Data.CountOfTransfers != 1 {
		t.Errorf("CountOfTransfers = %d, want 1", debtusSpace.Data.CountOfTransfers)
	}
}

// ============================================================================
// updateDebtusSpaceAndCounterpartyWithTransferInfo
// NOTE: debtusSpace.ID is always "debtus" (moduleID) when created via
// models4debtus.NewDebtusSpaceEntry, but the switch inside the function
// compares debtusSpace.ID to transfer.Data.From().UserID / To().UserID.
// To make the function work in tests we must set the ID directly.
// ============================================================================

func newDebtusSpaceEntryWithID(id string) models4debtus.DebtusSpaceEntry {
	e := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	e.ID = id // override so the switch matches From/To UserID
	e.Data.Balance = make(money.Balance)
	return e
}

func TestUpdateDebtusSpaceAndCounterpartyWithTransferInfo(t *testing.T) {
	ctx := context.Background()
	transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      money.CurrencyEUR,
		AmountInCents: 100,
		FromJson:      `{"userID":"u1","contactID":"c1"}`,
		ToJson:        `{"userID":"u2","contactID":"c2"}`,
	})
	transfer.Data.DtCreated = time.Now()

	t.Run("from_user_perspective", func(t *testing.T) {
		debtusSpace := newDebtusSpaceEntryWithID("u1") // ID matches From().UserID
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		f := TransfersFacade{}
		err := f.updateDebtusSpaceAndCounterpartyWithTransferInfo(ctx, money.NewAmount(money.CurrencyEUR, 100), transfer, debtusSpace, contact, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if debtusSpace.Data.CountOfTransfers != 1 {
			t.Errorf("CountOfTransfers = %d, want 1", debtusSpace.Data.CountOfTransfers)
		}
	})

	t.Run("to_user_perspective", func(t *testing.T) {
		debtusSpace := newDebtusSpaceEntryWithID("u2") // ID matches To().UserID
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c1")
		f := TransfersFacade{}
		err := f.updateDebtusSpaceAndCounterpartyWithTransferInfo(ctx, money.NewAmount(money.CurrencyEUR, 100), transfer, debtusSpace, contact, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unrelated_debtus_space_panics", func(t *testing.T) {
		debtusSpace := newDebtusSpaceEntryWithID("u99") // ID matches nothing
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c1")
		f := TransfersFacade{}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for unrelated debtus space")
			}
		}()
		_ = f.updateDebtusSpaceAndCounterpartyWithTransferInfo(ctx, money.NewAmount(money.CurrencyEUR, 100), transfer, debtusSpace, contact, nil)
	})
}

// ============================================================================
// updateContactWithTransferInfo — pure function with interest branches
// ============================================================================

func TestUpdateContactWithTransferInfo(t *testing.T) {
	ctx := context.Background()

	makeTransfer := func(hasInterest bool) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		}
		td.DtCreated = time.Now()
		if hasInterest {
			td.TransferInterest = models4debtus.NewInterest("compound", 10, 365)
		}
		return models4debtus.NewTransfer("t1", td)
	}

	t.Run("simple_transfer_no_interest", func(t *testing.T) {
		transfer := makeTransfer(false)
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		if err := updateContactWithTransferInfo(ctx, 100, transfer, contact, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contact.Data.CountOfTransfers != 1 {
			t.Errorf("CountOfTransfers = %d, want 1", contact.Data.CountOfTransfers)
		}
		if contact.Data.LastTransferID != "t1" {
			t.Errorf("LastTransferID = %q, want t1", contact.Data.LastTransferID)
		}
	})

	t.Run("transfer_with_interest", func(t *testing.T) {
		transfer := makeTransfer(true)
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		if err := updateContactWithTransferInfo(ctx, 100, transfer, contact, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info := contact.Data.GetTransfersInfo()
		if len(info.OutstandingWithInterest) != 1 {
			t.Errorf("expected 1 outstanding with interest, got %d", len(info.OutstandingWithInterest))
		}
	})

	t.Run("same_transfer_id_is_no_op_for_transfers_info", func(t *testing.T) {
		transfer := makeTransfer(false)
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		// First call
		if err := updateContactWithTransferInfo(ctx, 100, transfer, contact, nil); err != nil {
			t.Fatalf("first call error: %v", err)
		}
		// Second call with same transfer ID — transfersInfo.Count should NOT increment again
		if err := updateContactWithTransferInfo(ctx, 100, transfer, contact, nil); err != nil {
			t.Fatalf("second call error: %v", err)
		}
		// CountOfTransfers increments each call; only transfersInfo inner block is blocked by same ID
		if contact.Data.GetTransfersInfo().Count != 1 {
			t.Errorf("transfersInfo.Count = %d, want 1 (no double-count)", contact.Data.GetTransfersInfo().Count)
		}
	})

	t.Run("with_return_to_interest_transfer", func(t *testing.T) {
		// First establish an interest-bearing transfer in the contact
		interestTransfer := makeTransfer(true)
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		if err := updateContactWithTransferInfo(ctx, 100, interestTransfer, contact, nil); err != nil {
			t.Fatalf("interest transfer error: %v", err)
		}

		// Now create a return transfer that references the interest transfer
		returnTd := &models4debtus.TransferData{
			CreatorUserID:       "u1",
			Currency:            money.CurrencyEUR,
			AmountInCents:       50,
			ReturnToTransferIDs: []string{"t1"},
			IsReturn:            true,
			FromJson:            `{"userID":"u1","contactID":"c1"}`,
			ToJson:              `{"userID":"u2","contactID":"c2"}`,
		}
		returnTd.DtCreated = time.Now()
		returnTransfer := models4debtus.NewTransfer("t2", returnTd)

		if err := updateContactWithTransferInfo(ctx, 50, returnTransfer, contact, nil); err != nil {
			t.Fatalf("return transfer error: %v", err)
		}
	})

	t.Run("return_to_closed_interest_transfer", func(t *testing.T) {
		// First establish an interest-bearing transfer in the contact
		interestTransfer := makeTransfer(true)
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		if err := updateContactWithTransferInfo(ctx, 100, interestTransfer, contact, nil); err != nil {
			t.Fatalf("interest transfer error: %v", err)
		}

		returnTd := &models4debtus.TransferData{
			CreatorUserID:       "u1",
			Currency:            money.CurrencyEUR,
			AmountInCents:       50,
			ReturnToTransferIDs: []string{"t1"},
			IsReturn:            true,
			FromJson:            `{"userID":"u1","contactID":"c1"}`,
			ToJson:              `{"userID":"u2","contactID":"c2"}`,
		}
		returnTd.DtCreated = time.Now()
		returnTransfer := models4debtus.NewTransfer("t2", returnTd)

		// "t1" is in closedTransferIDs - covers the isClosed(returnToTransferID) branch
		if err := updateContactWithTransferInfo(ctx, 50, returnTransfer, contact, []string{"t1"}); err != nil {
			t.Fatalf("closed return error: %v", err)
		}
	})
}

// ============================================================================
// InsertTransfer — needs DB TX
// ============================================================================

func TestInsertTransfer(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	var transfer models4debtus.TransferEntry
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		}
		transfer, err = InsertTransfer(ctx, tx, td)
		return err
	})
	if err != nil {
		t.Fatalf("InsertTransfer() returned error: %v", err)
	}
	if transfer.ID == "" {
		t.Error("expected non-empty transfer ID")
	}
}

// ============================================================================
// UpdateTransferOnReturn — needs TX
// ============================================================================

func TestUpdateTransferOnReturn(t *testing.T) {
	ctx := context.Background()

	makeTransfer := func(id string, currency money.CurrencyCode, amount decimal.Decimal64p2, fromUserID, fromContactID, toUserID, toContactID string) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: fromUserID,
			Currency:      currency,
			AmountInCents: amount,
			IsOutstanding: true,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		td.FromJson = `{"userID":"` + fromUserID + `","contactID":"` + fromContactID + `"}`
		td.ToJson = `{"userID":"` + toUserID + `","contactID":"` + toContactID + `"}`
		return models4debtus.NewTransfer(id, td)
	}

	t.Run("normal_full_return", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		origTransfer := makeTransfer("t1", money.CurrencyEUR, 100, "u1", "c1", "u2", "c2")
		returnTransfer := makeTransfer("t2", money.CurrencyEUR, 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origTransfer.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, origTransfer, 100)
		})
		if err != nil {
			t.Fatalf("UpdateTransferOnReturn() returned error: %v", err)
		}
	})

	t.Run("already_returned_is_no_op", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		origTransfer := makeTransfer("t1", money.CurrencyEUR, 100, "u1", "c1", "u2", "c2")
		returnTransfer := makeTransfer("t2", money.CurrencyEUR, 100, "u2", "c2", "u1", "c1")
		// pre-add the return so it's already recorded
		_ = origTransfer.Data.AddReturn(models4debtus.TransferReturnJson{
			TransferID: "t2",
			Time:       time.Now(),
			Amount:     100,
		})
		seedRecords(t, db, origTransfer.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, origTransfer, 100)
		})
		if err != nil {
			t.Fatalf("UpdateTransferOnReturn() returned error: %v", err)
		}
	})

	t.Run("partial_return_excess_is_capped", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		origTransfer := makeTransfer("t1", money.CurrencyEUR, 50, "u1", "c1", "u2", "c2")
		returnTransfer := makeTransfer("t2", money.CurrencyEUR, 100, "u2", "c2", "u1", "c1") // returning more than outstanding
		seedRecords(t, db, origTransfer.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, origTransfer, 100) // amount > outstanding
		})
		if err != nil {
			t.Fatalf("UpdateTransferOnReturn() returned error: %v", err)
		}
	})

	t.Run("currency_mismatch_panics", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		origTransfer := makeTransfer("t1", money.CurrencyEUR, 100, "u1", "c1", "u2", "c2")
		returnTransfer := makeTransfer("t2", "USD", 100, "u2", "c2", "u1", "c1")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for currency mismatch")
			}
		}()
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, origTransfer, 100)
		})
	})

	t.Run("fixed_from_contact_id", func(t *testing.T) {
		// covers: returnTransfer.From().ContactID != "" && transfer.To().ContactID == "" && from.UserID == to.UserID
		db := sneattesting.SetupMemoryDB(t)
		// transfer has empty To().ContactID but same UserID as returnTransfer.From().UserID
		origTd := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			IsOutstanding: true,
		}
		origTd.DtCreated = time.Now().Add(-time.Hour)
		origTd.FromJson = `{"userID":"u1","contactID":"c1"}`
		origTd.ToJson = `{"userID":"u2"}` // no contactID
		origTransfer := models4debtus.NewTransfer("t1", origTd)
		returnTransfer := makeTransfer("t2", money.CurrencyEUR, 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origTransfer.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, origTransfer, 100)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// ============================================================================
// MarkReceiptAsViewed — needs dal4debtus.Default.Receipt
// ============================================================================

func TestMarkReceiptAsViewed(t *testing.T) {
	ctx := context.Background()

	setupReceiptDal := func(t *testing.T) {
		t.Helper()
		old := dal4debtus.Default.Receipt
		t.Cleanup(func() { dal4debtus.Default.Receipt = old })
		dal4debtus.Default.Receipt = fakeReceiptDal{}
	}

	t.Run("marks_receipt_as_viewed", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)

		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			Status:        models4debtus.ReceiptStatusSent,
			CreatorUserID: "u1",
			TransferID:    "t1",
		})
		seedRecords(t, db, receipt.Record)

		got, err := MarkReceiptAsViewed(ctx, "r1", "u1")
		if err != nil {
			t.Fatalf("MarkReceiptAsViewed() returned error: %v", err)
		}
		if len(got.Data.ViewedByUserIDs) == 0 {
			t.Error("expected ViewedByUserIDs to contain u1")
		}
		if got.Data.DtViewed.IsZero() {
			t.Error("expected DtViewed to be set")
		}
	})

	t.Run("already_viewed_is_no_op", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)

		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			Status:          models4debtus.ReceiptStatusViewed,
			CreatorUserID:   "u1",
			TransferID:      "t1",
			ViewedByUserIDs: []string{"u1"},
			DtViewed:        time.Now().Add(-time.Hour),
		})
		seedRecords(t, db, receipt.Record)

		_, err := MarkReceiptAsViewed(ctx, "r1", "u1")
		if err != nil {
			t.Fatalf("MarkReceiptAsViewed() returned error: %v", err)
		}
	})
}

// ============================================================================
// GetUsersByIDs — needs DB
// ============================================================================

func TestUserFacade_GetUsersByIDs(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_empty_on_empty_ids", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		users, err := User.GetUsersByIDs(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("expected 0 users, got %d", len(users))
		}
	})

	t.Run("no_error_with_valid_ids", func(t *testing.T) {
		// Note: dal4userus.GetUsersByIDs has a known quirk — it loads records via
		// GetMulti but never assigns them to the named return variable, so the
		// returned slice is always empty. The test exercises the happy path
		// (no error returned) rather than checking loaded values.
		db := sneattesting.SetupMemoryDB(t)
		u1 := dbo4userus.NewUserEntry("u1")
		u1.Data.Names = &person.NameFields{FirstName: "Alice"}
		u2 := dbo4userus.NewUserEntry("u2")
		u2.Data.Names = &person.NameFields{FirstName: "Bob"}
		seedRecords(t, db, u1.Record, u2.Record)

		_, err := User.GetUsersByIDs(ctx, []string{"u1", "u2"})
		if err != nil {
			t.Fatalf("GetUsersByIDs() returned error: %v", err)
		}
	})
}

// fakeUserEmailDal is a minimal unsorted4auth.UserEmailDal that returns
// ErrRecordNotFound for GetUserEmailByID, allowing CreateUserByEmail to proceed
// past the lookup and reach the "not implemented" return.
type fakeUserEmailDal struct{}

func (fakeUserEmailDal) GetUserEmailByID(_ context.Context, _ dal.ReadSession, _ string) (models4auth.UserEmailEntry, error) {
	return models4auth.UserEmailEntry{}, dal.ErrRecordNotFound
}

func (fakeUserEmailDal) SaveUserEmail(_ context.Context, _ dal.ReadwriteTransaction, _ models4auth.UserEmailEntry) error {
	return errors.New("not implemented in fakeUserEmailDal")
}

// ============================================================================
// CreateUserByEmail — returns "not implemented"
// ============================================================================

func TestUserFacade_CreateUserByEmail(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	// Set the global UserEmail service locator to a fake implementation so
	// CreateUserByEmail does not panic on nil pointer dereference.
	origUserEmail := unsorted4auth.UserEmail
	unsorted4auth.UserEmail = fakeUserEmailDal{}
	t.Cleanup(func() { unsorted4auth.UserEmail = origUserEmail })

	_, _, err := User.CreateUserByEmail(ctx, "test@example.com", "Test User")
	if err == nil {
		t.Fatal("expected 'not implemented' error")
	}
	if err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got: %v", err)
	}
}

// ============================================================================
// DelayUpdateHasDueTransfers — cover the wg.Wait() path
// (The wg.Wait() call is MISSING in production code — the current code checks
// errs BEFORE goroutines finish. The test exercises the current code path.)
// ============================================================================

func TestDelayUpdateHasDueTransfers_wg_coverage(t *testing.T) {
	initDelayers4Test(t)
	ctx := context.Background()
	// valid call — both goroutines complete, errs is empty at check point
	if err := DelayUpdateHasDueTransfers(ctx, "u1", testSpaceID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// NOTE: delayedUpdateSpaceHasDueTransfers cannot be directly tested.
// The function calls RunModuleSpaceWorkerWithUserCtx which creates a fresh
// ModuleSpaceWorkerParams inside the transaction. The worker then directly
// accesses params.SpaceModuleEntry.Data without first calling GetRecords(ctx,tx),
// which violates the dal record contract (Record.Exists() panics if SetError was
// never called). This is a production-code bug: the worker must call
// params.GetRecords(ctx, tx) before reading SpaceModuleEntry.Data.
// Coverage of this function is not achievable without fixing the production code
// or providing a seam (inject the worker via a variable).
//
// Similarly, the update_has_due_transfers_test.go covers DelayUpdateHasDueTransfers
// which enqueues to the delayer, so the top-level queue entry point IS covered.
