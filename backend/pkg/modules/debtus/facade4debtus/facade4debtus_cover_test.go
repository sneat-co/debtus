package facade4debtus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/auth/unsorted4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dbo4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/decimal"
	"github.com/strongo/strongoapp/person"
)

// fakeUserEmailFoundDal returns an existing UserEmail (no error) from
// GetUserEmailByID so GetOrCreateEmailUser takes the "found" branch.
type fakeUserEmailFoundDal struct{}

func (fakeUserEmailFoundDal) GetUserEmailByID(_ context.Context, _ dal.ReadSession, id string) (models4auth.UserEmailEntry, error) {
	return models4auth.NewUserEmail(id, models4auth.NewUserEmailData(0, true, "email")), nil
}

func (fakeUserEmailFoundDal) SaveUserEmail(_ context.Context, _ dal.ReadwriteTransaction, _ models4auth.UserEmailEntry) error {
	return nil
}

// ============================================================================
// GetOrCreateEmailUser
// ============================================================================

func TestUserFacade_GetOrCreateEmailUser(t *testing.T) {
	ctx := context.Background()

	// NOTE: the "email not found" branch (creates a new user) is not coverable:
	// it calls dbo4userus.NewUserEntry("") with an empty user ID, which panics in
	// NewUserKey. Documented as a gap.

	t.Run("returns_existing_user_when_email_found", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		orig := unsorted4auth.UserEmail
		unsorted4auth.UserEmail = fakeUserEmailFoundDal{}
		t.Cleanup(func() { unsorted4auth.UserEmail = orig })

		userEmail, isNewUser, err := User.GetOrCreateEmailUser(ctx, "found@example.com", true, nil, common4all.ClientInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isNewUser {
			t.Error("expected isNewUser=false for found email")
		}
		if userEmail.ID != "found@example.com" {
			t.Errorf("userEmail.ID = %q, want found@example.com", userEmail.ID)
		}
	})
}

// ============================================================================
// getReceiptTransferAndUsers
// ============================================================================

func TestGetReceiptTransferAndUsers(t *testing.T) {
	ctx := context.Background()

	setupReceiptDal := func(t *testing.T) {
		t.Helper()
		old := dal4debtus.Default.Receipt
		t.Cleanup(func() { dal4debtus.Default.Receipt = old })
		dal4debtus.Default.Receipt = fakeReceiptDal{}
	}

	t.Run("loads_receipt_transfer_and_users_with_counterparty", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)

		transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		})
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1",
			TransferID:    "t1",
		})
		creatorUser := dbo4userus.NewUserEntry("u1")
		creatorDebtusUser := models4debtus.NewDebtusUserEntry("u1")
		cpUser := dbo4userus.NewUserEntry("u2")
		cpDebtusUser := models4debtus.NewDebtusUserEntry("u2")
		seedRecords(t, db, transfer.Record, receipt.Record,
			creatorUser.Record, creatorDebtusUser.Record, cpUser.Record, cpDebtusUser.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			r, tr, cu, cdu, cpu, _, err := getReceiptTransferAndUsers(ctx, tx, "r1", "u2")
			if err != nil {
				return err
			}
			if r.ID != "r1" || tr.ID != "t1" {
				t.Errorf("unexpected receipt/transfer ids: %s/%s", r.ID, tr.ID)
			}
			if cu.ID != "u1" {
				t.Errorf("unexpected creator id: %s", cu.ID)
			}
			if cdu.Data == nil {
				t.Error("expected non-nil creatorDebtusUser data")
			}
			if cpu.ID != "u2" {
				t.Errorf("unexpected counterparty id: %s", cpu.ID)
			}
			return nil
		}, dal.TxWithCrossGroup())
		if err != nil {
			t.Fatalf("getReceiptTransferAndUsers() returned error: %v", err)
		}
	})

	t.Run("no_counterparty_user_id_loads_only_creator", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)

		transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"contactID":"c2"}`, // no counterparty userID
		})
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1",
			TransferID:    "t1",
		})
		creatorUser := dbo4userus.NewUserEntry("u1")
		creatorDebtusUser := models4debtus.NewDebtusUserEntry("u1")
		seedRecords(t, db, transfer.Record, receipt.Record, creatorUser.Record, creatorDebtusUser.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, _, _, _, cpu, _, err := getReceiptTransferAndUsers(ctx, tx, "r1", "u1")
			if err != nil {
				return err
			}
			if cpu.ID != "" {
				t.Errorf("expected empty counterparty, got %s", cpu.ID)
			}
			return nil
		}, dal.TxWithCrossGroup())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("creator_mismatch_returns_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)

		transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "different",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"different","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		})
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1", // mismatch with transfer.CreatorUserID
			TransferID:    "t1",
		})
		seedRecords(t, db, transfer.Record, receipt.Record)

		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, _, _, _, _, _, err := getReceiptTransferAndUsers(ctx, tx, "r1", "u1")
			return err
		}, dal.TxWithCrossGroup())
		if err == nil {
			t.Fatal("expected data integrity error for creator mismatch")
		}
	})

	t.Run("receipt_not_found_returns_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, _, _, _, _, _, err := getReceiptTransferAndUsers(ctx, tx, "missing", "u1")
			return err
		}, dal.TxWithCrossGroup())
		if !dal.IsNotFound(err) {
			t.Fatalf("expected not-found error, got: %v", err)
		}
	})

	t.Run("transfer_not_found_returns_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupReceiptDal(t)
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1",
			TransferID:    "missing-transfer",
		})
		seedRecords(t, db, receipt.Record)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, _, _, _, _, _, err := getReceiptTransferAndUsers(ctx, tx, "r1", "u1")
			return err
		}, dal.TxWithCrossGroup())
		if !dal.IsNotFound(err) {
			t.Fatalf("expected not-found error, got: %v", err)
		}
	})
}

// ============================================================================
// CreateTransferInput.Direction — empty-creator panic (reached directly,
// bypassing NewUserEntry which would panic earlier)
// ============================================================================

func TestCreateTransferInput_Direction_EmptyCreatorPanics(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
	input := CreateTransferInput{
		Source:  testTransferSource{},
		Request: validCreateTransferRequest(),
		From:    from,
		To:      to,
		// CreatorUser left as zero value => ID == ""
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty creator user id")
		}
	}()
	_ = input.Direction()
}

// ============================================================================
// CreateTransferInput.Validate — remaining branches
// ============================================================================

func TestCreateTransferInput_Validate_ExtraBranches(t *testing.T) {
	makeCreator := func(id string) dbo4userus.UserEntry {
		// Construct a UserEntry with a non-empty ID but nil Data, avoiding
		// NewUserEntry which always allocates Data.
		u := dbo4userus.UserEntry{}
		u.ID = id
		return u
	}

	t.Run("creator_data_nil", func(t *testing.T) {
		input := CreateTransferInput{
			Source:      testTransferSource{},
			CreatorUser: makeCreator("u1"), // ID set, Data nil
			Request:     validCreateTransferRequest(),
			From:        &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"},
			To:          &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"},
		}
		if err := input.Validate(); err == nil || err.Error() != "creatorUser.Data == nil" {
			t.Errorf("expected creatorUser.Data == nil error, got: %v", err)
		}
	})

	t.Run("both_users_empty_missing_from_contact", func(t *testing.T) {
		input := newCreateTransferInput("u1", nil, nil)
		input.From = &models4debtus.TransferCounterpartyInfo{ContactID: ""}
		input.To = &models4debtus.TransferCounterpartyInfo{ContactID: "c2"}
		if err := input.Validate(); err == nil {
			t.Error("expected error for empty from.ContactID when both users empty")
		}
	})

	t.Run("both_users_empty_missing_to_contact", func(t *testing.T) {
		input := newCreateTransferInput("u1", nil, nil)
		input.From = &models4debtus.TransferCounterpartyInfo{ContactID: "c1"}
		input.To = &models4debtus.TransferCounterpartyInfo{ContactID: ""}
		if err := input.Validate(); err == nil {
			t.Error("expected error for empty to.ContactID when both users empty")
		}
	})

	t.Run("3d_party_missing_from_contact", func(t *testing.T) {
		// creator is neither from.UserID nor to.UserID => default branch
		input := newCreateTransferInput("u1", nil, nil)
		input.CreatorUser = dbo4userus.NewUserEntry("u3")
		input.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: ""}
		input.To = &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
		input.Request.BillID = "bill1"
		if err := input.Validate(); err == nil {
			t.Error("expected error for 3d-party empty from.ContactID")
		}
	})

	t.Run("3d_party_missing_to_contact", func(t *testing.T) {
		input := newCreateTransferInput("u1", nil, nil)
		input.CreatorUser = dbo4userus.NewUserEntry("u3")
		input.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
		input.To = &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: ""}
		input.Request.BillID = "bill1"
		if err := input.Validate(); err == nil {
			t.Error("expected error for 3d-party empty to.ContactID")
		}
	})
}

// fakeContactDal is a minimal dal4debtus.ContactDal stub used to drive
// CreateContact's switch on the number of matching contact IDs.
type fakeContactDal struct {
	ids      []string
	titleErr error
}

func (f fakeContactDal) GetContactIDsByTitle(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _, _ string, _ bool) ([]string, error) {
	return f.ids, f.titleErr
}
func (fakeContactDal) GetLatestContacts(_ context.Context, _ string, _ dal.ReadSession, _ coretypes.SpaceID, _, _ int) ([]models4debtus.DebtusSpaceContactEntry, error) {
	return nil, nil
}
func (fakeContactDal) InsertContact(_ context.Context, _ dal.ReadwriteTransaction, _ *models4debtus.DebtusSpaceContactDbo) (models4debtus.DebtusSpaceContactEntry, error) {
	return models4debtus.DebtusSpaceContactEntry{}, errors.New("not implemented in fakeContactDal")
}
func (fakeContactDal) GetContactsWithDebts(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) ([]models4debtus.DebtusSpaceContactEntry, error) {
	return nil, nil
}

// ============================================================================
// CreateContact — switch branches on number of matching contact IDs
// ============================================================================

func TestCreateContact_SwitchBranches(t *testing.T) {
	ctx := facade.NewContextWithUserID(context.Background(), "u1")

	setContactDal := func(t *testing.T, d fakeContactDal) {
		t.Helper()
		orig := dal4debtus.Default.Contact
		dal4debtus.Default.Contact = d
		t.Cleanup(func() { dal4debtus.Default.Contact = orig })
	}

	t.Run("title_lookup_error", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setContactDal(t, fakeContactDal{titleErr: errors.New("boom")})
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			uc := facade.NewContextWithUserID(ctx, "u1")
			_, _, _, err := CreateContact(uc, tx, testSpaceID, dto4contactus.ContactDetails{})
			return err
		}, dal.TxWithCrossGroup())
		if err == nil {
			t.Fatal("expected error from GetContactIDsByTitle")
		}
	})

	t.Run("too_many_contacts", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setContactDal(t, fakeContactDal{ids: []string{"c1", "c2"}})
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			uc := facade.NewContextWithUserID(ctx, "u1")
			_, _, _, err := CreateContact(uc, tx, testSpaceID, dto4contactus.ContactDetails{})
			return err
		}, dal.TxWithCrossGroup())
		if err == nil {
			t.Fatal("expected 'too many counterparties' error")
		}
	})

	// NOTE: the case-1 branch (exactly one matching contact) is not coverable:
	// after loading the debtus contact it calls tx.Get(ctx, contact.Record) where
	// `contact` is the zero-value named-return dal4contactus.ContactEntry with a
	// nil Record, which panics in dalgo2memory. Documented as a gap.
}

// fakeReminderDal counts DelayDiscardRemindersForTransfers calls and can be
// configured to return an error.
type fakeReminderDal struct {
	discardErr   error
	discardCalls int
}

func (f *fakeReminderDal) DelayDiscardRemindersForTransfers(_ context.Context, _ []string, _ string) error {
	f.discardCalls++
	return f.discardErr
}
func (*fakeReminderDal) DelayCreateReminderForTransferUser(_ context.Context, _, _ string) error {
	return nil
}
func (*fakeReminderDal) GetActiveReminderIDsByTransferID(_ context.Context, _ dal.ReadSession, _ string) ([]string, error) {
	return nil, nil
}
func (*fakeReminderDal) GetSentReminderIDsByTransferID(_ context.Context, _ dal.ReadSession, _ string) ([]string, error) {
	return nil, nil
}

// ============================================================================
// UpdateTransferOnReturn — extra branches
// ============================================================================

func TestUpdateTransferOnReturn_ExtraBranches(t *testing.T) {
	ctx := context.Background()

	makeTransfer := func(id string, amount decimal.Decimal64p2, fromUserID, fromContactID, toUserID, toContactID string) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: fromUserID,
			Currency:      money.CurrencyEUR,
			AmountInCents: amount,
			IsOutstanding: true,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		td.FromJson = `{"userID":"` + fromUserID + `","contactID":"` + fromContactID + `"}`
		td.ToJson = `{"userID":"` + toUserID + `","contactID":"` + toContactID + `"}`
		return models4debtus.NewTransfer(id, td)
	}

	t.Run("reminder_discard_called", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		orig := dal4debtus.Default.Reminder
		fake := &fakeReminderDal{}
		dal4debtus.Default.Reminder = fake
		t.Cleanup(func() { dal4debtus.Default.Reminder = orig })

		origT := makeTransfer("t1", 100, "u1", "c1", "u2", "c2")
		returnT := makeTransfer("t2", 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origT.Record)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnT, origT, 100)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.discardCalls != 1 {
			t.Errorf("expected 1 discard call, got %d", fake.discardCalls)
		}
	})

	t.Run("reminder_discard_error_is_wrapped", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		orig := dal4debtus.Default.Reminder
		dal4debtus.Default.Reminder = &fakeReminderDal{discardErr: errTestReminder}
		t.Cleanup(func() { dal4debtus.Default.Reminder = orig })

		origT := makeTransfer("t1", 100, "u1", "c1", "u2", "c2")
		returnT := makeTransfer("t2", 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origT.Record)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnT, origT, 100)
		})
		if err == nil {
			t.Fatal("expected error from reminder discard")
		}
	})

	t.Run("already_fully_returned_outstanding_zero", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		orig := dal4debtus.Default.Reminder
		dal4debtus.Default.Reminder = nil
		t.Cleanup(func() { dal4debtus.Default.Reminder = orig })

		// Transfer with an existing return that fully covers the amount =>
		// outstandingValue <= 0 and a NEW return transfer (different ID) so the
		// "already has info" loop does not short-circuit.
		origT := makeTransfer("t1", 100, "u1", "c1", "u2", "c2")
		_ = origT.Data.AddReturn(models4debtus.TransferReturnJson{TransferID: "tOld", Time: time.Now(), Amount: 100})
		returnT := makeTransfer("t2", 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origT.Record)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnT, origT, 100)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fixes_to_contact_id_from_return", func(t *testing.T) {
		// returnTransfer.To().ContactID != "" && transfer.From().ContactID == "" &&
		// returnTransfer.To().UserID == transfer.From().UserID
		db := sneattesting.SetupMemoryDB(t)
		orig := dal4debtus.Default.Reminder
		dal4debtus.Default.Reminder = nil
		t.Cleanup(func() { dal4debtus.Default.Reminder = orig })

		origT := makeTransfer("t1", 100, "u1", "", "u2", "c2") // From has no contactID
		returnT := makeTransfer("t2", 100, "u2", "c2", "u1", "c1")
		seedRecords(t, db, origT.Record)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnT, origT, 100)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("from_contact_id_mismatch_panics", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		origT := makeTransfer("t1", 100, "u1", "c1", "u2", "cX")   // To().ContactID=cX
		returnT := makeTransfer("t2", 100, "u2", "cY", "u9", "c1") // From().ContactID=cY != cX, userIDs differ
		seedRecords(t, db, origT.Record)
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for From().ContactID mismatch")
			}
		}()
		_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return Transfers.UpdateTransferOnReturn(ctx, tx, returnT, origT, 100)
		})
	})
}

// ============================================================================
// updateContactWithTransferInfo — extra branches
// ============================================================================

func TestUpdateContactWithTransferInfo_ExtraBranches(t *testing.T) {
	ctx := context.Background()

	makeInterestTransfer := func(id string) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		}
		td.DtCreated = time.Now()
		td.TransferInterest = models4debtus.NewInterest("compound", 10, 365)
		return models4debtus.NewTransfer(id, td)
	}

	t.Run("return_to_multiple_interest_debts_errors", func(t *testing.T) {
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		// seed two interest-bearing outstanding transfers
		if err := updateContactWithTransferInfo(ctx, 100, makeInterestTransfer("t1"), contact, nil); err != nil {
			t.Fatalf("seed t1 error: %v", err)
		}
		if err := updateContactWithTransferInfo(ctx, 100, makeInterestTransfer("t2"), contact, nil); err != nil {
			t.Fatalf("seed t2 error: %v", err)
		}
		// return transfer referencing BOTH outstanding interest debts
		returnTd := &models4debtus.TransferData{
			CreatorUserID:       "u1",
			Currency:            money.CurrencyEUR,
			AmountInCents:       50,
			IsReturn:            true,
			ReturnToTransferIDs: []string{"t1", "t2"},
			FromJson:            `{"userID":"u1","contactID":"c1"}`,
			ToJson:              `{"userID":"u2","contactID":"c2"}`,
		}
		returnTd.DtCreated = time.Now()
		returnTransfer := models4debtus.NewTransfer("t3", returnTd)
		err := updateContactWithTransferInfo(ctx, 50, returnTransfer, contact, nil)
		if err == nil {
			t.Error("expected error for return to multiple interest debts")
		}
	})

	t.Run("return_to_unknown_transfer_id_is_logged", func(t *testing.T) {
		contact := newDebtusSpaceContactEntryWithBalance(testSpaceID, "c2")
		if err := updateContactWithTransferInfo(ctx, 100, makeInterestTransfer("t1"), contact, nil); err != nil {
			t.Fatalf("seed t1 error: %v", err)
		}
		returnTd := &models4debtus.TransferData{
			CreatorUserID:       "u1",
			Currency:            money.CurrencyEUR,
			AmountInCents:       50,
			IsReturn:            true,
			ReturnToTransferIDs: []string{"unknown-id"}, // not in OutstandingWithInterest
			FromJson:            `{"userID":"u1","contactID":"c1"}`,
			ToJson:              `{"userID":"u2","contactID":"c2"}`,
		}
		returnTd.DtCreated = time.Now()
		returnTransfer := models4debtus.NewTransfer("t2", returnTd)
		if err := updateContactWithTransferInfo(ctx, 50, returnTransfer, contact, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

var errTestReminder = errors.New("reminder discard failed")

// ============================================================================
// ReceiptUsersLinker.updateTransfer — pure mutation on changes struct
// ============================================================================

func TestReceiptUsersLinker_updateTransfer(t *testing.T) {
	newParty := func(userID, contactID string) *userLinkingParty {
		p := &userLinkingParty{}
		p.user = dbo4userus.NewUserEntry(userID)
		p.user.Data.Names = &person.NameFields{FirstName: "N" + userID}
		p.contact = dal4contactus.NewContactEntryWithData(testSpaceID, contactID, &dbo4contactus.ContactDbo{})
		p.contact.Data.UserID = userID
		p.contact.Data.Names = &person.NameFields{FirstName: "C" + contactID}
		p.debtusContact = models4debtus.NewDebtusSpaceContactEntry(testSpaceID, contactID, &models4debtus.DebtusSpaceContactDbo{})
		return p
	}

	newChanges := func() *receiptDbChanges {
		c := newReceiptDbChanges()
		// inviter is creator u1 (contact cInviter references invited user u2 side? per code:
		// updateTransferCounterpartyInfo("inviter", Creator(), inviter.user, invited.contact))
		c.inviter = newParty("u1", "cInviter") // inviter.contact.UserID == u1
		c.invited = newParty("u2", "cInvited") // invited.contact.UserID == u2
		return c
	}

	t.Run("links_counterparty_user2counterparty", func(t *testing.T) {
		changes := newChanges()
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`, // creator on From side => U2C
			ToJson:        `{"contactID":"cInviter"}`,               // counterparty no userID yet
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changes.transfer.Data.Counterparty().UserID != "u2" {
			t.Errorf("counterparty UserID = %q, want u2", changes.transfer.Data.Counterparty().UserID)
		}
	})

	t.Run("invalid_transfer_returns_error", func(t *testing.T) {
		changes := newChanges()
		changes.transfer = models4debtus.TransferEntry{} // empty ID + nil Data
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err == nil {
			t.Error("expected error for invalid transfer")
		}
	})

	t.Run("creator_user_id_mismatch_returns_error", func(t *testing.T) {
		changes := newChanges()
		td := &models4debtus.TransferData{
			CreatorUserID: "uX", // != inviter.user.ID (u1)
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`,
			ToJson:        `{"contactID":"cInviter"}`,
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err == nil {
			t.Error("expected error for creator user id mismatch")
		}
	})

	t.Run("counterparty_already_has_matching_user_and_contact", func(t *testing.T) {
		// Counterparty side already fully populated => skip the assignment branches.
		changes := newChanges()
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`,
			ToJson:        `{"userID":"u2","contactID":"cInviter"}`, // counterparty fully set
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("counterparty_user_id_conflict_returns_error", func(t *testing.T) {
		changes := newChanges()
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`,
			ToJson:        `{"userID":"u9","contactID":"cInviter"}`, // counterparty userID != invited (u2)
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err == nil {
			t.Error("expected error for counterparty user id conflict")
		}
	})

	t.Run("inviter_user_missing_returns_error", func(t *testing.T) {
		changes := newChanges()
		changes.inviter.user = dbo4userus.UserEntry{} // ID == "" and Data nil
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`,
			ToJson:        `{"contactID":"cInviter"}`,
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err == nil {
			t.Error("expected error for missing inviter user")
		}
	})

	t.Run("inviter_contact_missing_returns_error", func(t *testing.T) {
		changes := newChanges()
		changes.inviter.contact = dal4contactus.ContactEntry{} // ID == "" and Data nil
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"cInvited"}`,
			ToJson:        `{"contactID":"cInviter"}`,
		}
		changes.transfer = models4debtus.NewTransfer("t1", td)
		linker := &ReceiptUsersLinker{changes: changes}
		if err := linker.updateTransfer(); err == nil {
			t.Error("expected error for missing inviter contact")
		}
	})
}

// fakeReceiptDalUpdateErr embeds the get behaviour of fakeReceiptDal but fails
// on UpdateReceipt to drive MarkReceiptAsViewed's error path.
type fakeReceiptDalUpdateErr struct{ fakeReceiptDal }

func (fakeReceiptDalUpdateErr) UpdateReceipt(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.ReceiptEntry) error {
	return errors.New("update receipt failed")
}

// ============================================================================
// MarkReceiptAsViewed — UpdateReceipt error path
// ============================================================================

func TestMarkReceiptAsViewed_UpdateError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	orig := dal4debtus.Default.Receipt
	dal4debtus.Default.Receipt = fakeReceiptDalUpdateErr{}
	t.Cleanup(func() { dal4debtus.Default.Receipt = orig })

	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		Status:        models4debtus.ReceiptStatusSent,
		CreatorUserID: "u1",
		TransferID:    "t1",
	})
	seedRecords(t, db, receipt.Record)

	if _, err := MarkReceiptAsViewed(ctx, "r1", "u1"); err == nil {
		t.Fatal("expected error from UpdateReceipt")
	}
}

// ============================================================================
// ChangeContactStatus — error branches
// ============================================================================

func TestChangeContactStatus_ErrorBranches(t *testing.T) {
	ctx := facade.NewContextWithUserID(context.Background(), "u1")

	newSpace := func() dal.Record {
		space := dbo4spaceus.NewSpaceEntry(testSpaceID)
		space.Data.UserIDs = []string{"u1"}
		return space.Record
	}

	t.Run("contact_not_found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedRecords(t, db, newSpace())
		_, _, err := ChangeContactStatus(ctx, testSpaceID, "missing", models4debtus.DebtusContactStatusArchived)
		if err == nil {
			t.Error("expected error when contact not found")
		}
	})

	t.Run("debtus_space_not_found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		// seed space + contact but NOT the debtus space => tx.Get(debtusSpace) fails
		seedRecords(t, db, newSpace(), contact.Record)
		_, _, err := ChangeContactStatus(ctx, testSpaceID, "c1", models4debtus.DebtusContactStatusArchived)
		if err == nil {
			t.Error("expected error when debtus space not found")
		}
	})
}

// ============================================================================
// GetDebtusSpaceContactsByIDs / GetDebtusSpaceContact — nil-tx branch
// ============================================================================

func TestGetDebtusSpaceContact_NilTx(t *testing.T) {
	ctx := context.Background()

	t.Run("GetDebtusSpaceContactsByIDs_nil_tx", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		c1 := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{
			Status: models4debtus.DebtusContactStatusActive,
		})
		seedRecords(t, db, c1.Record)
		contacts, err := GetDebtusSpaceContactsByIDs(ctx, nil, testSpaceID, []string{"c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(contacts) != 1 {
			t.Fatalf("expected 1 contact, got %d", len(contacts))
		}
	})

	t.Run("GetDebtusSpaceContact_not_found", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		contact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "missing", nil)
		if err := GetDebtusSpaceContact(ctx, nil, contact); !dal.IsNotFound(err) {
			t.Errorf("expected not-found, got: %v", err)
		}
	})
}
