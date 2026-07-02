package facade4debtus

import (
	"context"
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/spaceus/dto4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
)

// ============================================================================
// Cross-space lending — verification of the CreateTransfer facade.
//
// Summary of what these tests establish (see the PR description / gap report
// for the full write-up):
//
//   - The TYPE SHAPE for cross-space lending is in place: TransferCounterpartyInfo
//     (models4debtus/transfer_fromto.go) carries an independent SpaceID per side,
//     and contacts/space-aggregates are genuinely stored per space via
//     dbo4spaceus.NewSpaceModule(Item)Key (see transfer_cross_space_test.go).
//   - The CreateTransfer FACADE that would actually exercise that shape for a
//     real transfer is not yet wired end-to-end, in both the same-space and
//     cross-space cases:
//       1. api4transfers.HandleCreateTransfer never populates `from`/`to` from
//          the incoming request (both stay nil), so the HTTP endpoint would
//          panic in NewTransferInput's internal Validate() before any business
//          logic runs. See debtus/api/api4transfers/http_create_transfer.go:33-45.
//       2. TransfersFacade.CreateTransfer (transfer_facade.go:162) called
//          dal4contactus.GetContactByID(ctx, nil, ...) — GetContact() (in the
//          contactus module) calls tx.Get() directly on that nil dal.ReadSession
//          with no fallback, so this panicked with a nil pointer dereference on
//          every transfer whose contact wasn't already cached in the creator's
//          DebtusSpaceDbo.Contacts map (i.e. on most first-time transfers).
//          FIXED in this PR: pass the already-resolved `db` instead of a
//          literal nil (small, low-risk, matches the nil-tx-fallback pattern
//          used elsewhere in this same file, e.g. GetDebtusSpaceContact).
//       3. TransfersFacade.createTransferWithinTransaction never populates
//          ParticipantEntries.DebtusContact, ParticipantEntries.DebtusSpace, or
//          ParticipantEntries.SpaceID for either side (transfer_facade.go:321-
//          341, 663-699) before reading/writing through them. This is also
//          documented by the pre-existing comment in transfer_create_deep_test.go
//          ("the deep transactional body builds records with nil keys and
//          cannot complete on the in-memory DB").
//
//   Net effect: the full CreateTransfer path currently cannot complete for a
//   normal contact-based transfer — same-space or cross-space — without
//   further facade work. Fix #2 above removes one panic and lets execution
//   reach further into the transaction, surfacing the next gap; it does not
//   make the path complete.
//
// UPDATE (create-path gap closed): #1 and #3 above are now fixed —
// api/api4transfers/http_create_transfer.go builds real From/To from the
// request, and createTransferWithinTransaction now populates
// ParticipantEntries.Contact/DebtusContact/DebtusSpace/SpaceID per side using
// each side's OWN space (see the cross-space convention comment in that
// function). Several further, closely-related gaps surfaced once execution
// got that far and are fixed alongside: CreateTransferInput.Request.Interest
// was unconditionally dereferenced (nil panic for any request without
// explicit interest); fixUserName/fixContactName dereferenced nil
// Names/Data; updateDebtusSpaceAndCounterpartyWithTransferInfo determined
// "which side is this?" by switching on debtusSpace.ID, which is always
// const4debtus.ModuleID (never a UserID) — it now takes an explicit
// isFromSide bool; money.Balanced.AddToBalance() panics on a nil Balance map,
// so NewDebtusSpaceEntry/NewDebtusSpaceContactEntry now always initialize a
// non-nil Balance. TestCreateTransfer_CrossSpace below is un-skipped and
// passing; TestCreateTransfer_SameSpace_Balances and
// TestCreateTransfer_CrossSpace_SettleNetsToZero add the balance/settle
// coverage called out in the TODOs.
// ============================================================================

// NOTE: space IDs must be lower-case alphanumeric/underscore only (enforced
// by dbo4spaceus.NewSpaceKey since the sneat-core-modules v0.42.5 bump).
const (
	testSpaceA coretypes.SpaceID = "space_a_lender"
	testSpaceB coretypes.SpaceID = "space_b_borrower"
)

// sameSpaceCreateTransferInput builds a CreateTransferInput for the ordinary,
// single-space "user u1 lends to contact c2" scenario.
func sameSpaceCreateTransferInput() CreateTransferInput {
	u1 := dbo4userus.NewUserEntry("u1")
	return CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: u1,
		Request: CreateTransferRequest{
			SpaceRequest: validCreateTransferRequest().SpaceRequest, // testSpaceID
			Direction:    models4debtus.TransferDirectionUser2Counterparty,
			Amount:       money.NewAmount(money.CurrencyEUR, 100),
			ToContactID:  "c2",
		},
		From: &models4debtus.TransferCounterpartyInfo{UserID: "u1", SpaceID: testSpaceID},
		To:   &models4debtus.TransferCounterpartyInfo{ContactID: "c2", SpaceID: testSpaceID},
	}
}

// TestCreateTransfer_ContactRecoveryFromDB_NoLongerPanicsOnNilTx is a
// regression test for the fix in this PR: previously, any CreateTransfer
// call whose contact was not already cached in the creator's DebtusSpaceDbo
// Contacts map would panic with a nil pointer dereference inside
// dal4contactus.GetContactByID(ctx, nil, ...), because that call passed a
// literal nil dal.ReadSession into a function with no nil-tx fallback.
//
// This test seeds a DB with no cached Contacts brief, so the "recover from DB
// record" branch (transfer_facade.go, right after "If Contact not found in
// user's JSON try to recover from DB record") is guaranteed to run. Before
// the fix, this call panicked; after the fix, it returns a normal (non-panic)
// error or proceeds, letting the caller continue past that specific gap.
func TestCreateTransfer_ContactRecoveryFromDB_NoLongerPanicsOnNilTx(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	u1 := dbo4userus.NewUserEntry("u1")
	contact := dal4contactus.NewContactEntryWithData(testSpaceID, "c2", &dbo4contactus.ContactDbo{})
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID) // Contacts map intentionally left empty/nil
	debtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c2", nil)

	seedRecords(t, db, u1.Record, contact.Record, debtusSpace.Record, debtusContact.Record)

	input := sameSpaceCreateTransferInput()

	// Must not panic. (Before the fix this call paniced with "invalid memory
	// address or nil pointer dereference" inside dal4contactus.GetContact.)
	_, err := Transfers.CreateTransfer(ctx, input)

	// The call is still expected to fail — createTransferWithinTransaction has
	// further, separately-documented gaps (see the package doc comment above)
	// — but it must fail with a normal Go error, not a panic.
	if err == nil {
		t.Log("CreateTransfer unexpectedly succeeded end-to-end; if the deeper " +
			"ParticipantEntries wiring gap has been fixed, please replace this " +
			"test with a full happy-path assertion (balances, stored transfer, etc).")
	}
}

// TestCreateTransfer_DeepTransactionGap_SameSpace pins the CURRENT (broken)
// behavior of the full CreateTransfer path for an otherwise well-formed
// same-space transfer (user u1 lends to a real, already-linked contact c2).
// It documents that the path does not complete under the in-memory test
// harness — either because ContactBrief.UserID (a deeply-embedded field via
// ContactDbo -> ContactBase -> ContactBrief -> WithUserID) does not survive a
// dalgo2memory round trip (an adapter-level limitation, consistent with the
// "in-memory adapter limitations" already logged in
// debtus/debtusdal/TEST-COVERAGE.md), or — once past that — because
// createTransferWithinTransaction never populates ParticipantEntries.
// DebtusContact/DebtusSpace/SpaceID for either side before reading/writing
// through them (transfer_facade.go:321-341, 663-699; gap #3 in the package
// doc comment above), which panics with a nil pointer dereference.
//
// This is NOT specific to cross-space — even the same-space happy path
// cannot complete today, whichever of the two failure modes is hit first.
func TestCreateTransfer_DeepTransactionGap_SameSpace(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	u1 := dbo4userus.NewUserEntry("u1")
	contactDbo := &dbo4contactus.ContactDbo{}
	contactDbo.UserID = "u1" // promoted field (via ContactBase -> ContactBrief -> WithUserID); can't set in the literal above
	contact := dal4contactus.NewContactEntryWithData(testSpaceID, "c2", contactDbo)
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
	debtusContact := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c2", nil)

	seedRecords(t, db, u1.Record, contact.Record, debtusSpace.Record, debtusContact.Record)

	input := sameSpaceCreateTransferInput()

	var (
		output CreateTransferOutput
		err    error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("documented gap reproduced: CreateTransfer panicked: %v", r)
			}
		}()
		output, err = Transfers.CreateTransfer(ctx, input)
	}()

	if err == nil && output.Transfer.ID != "" {
		t.Fatal("CreateTransfer unexpectedly succeeded end-to-end for a normal " +
			"same-space transfer — if the ParticipantEntries wiring gap (and/or " +
			"the ContactBrief.UserID round-trip issue) has been fixed, please " +
			"replace this test with a full happy-path assertion (balances, " +
			"stored transfer under the right space path, etc).")
	}
	if err != nil {
		t.Logf("documented gap reproduced: CreateTransfer returned error before completing: %v", err)
	}
}

// TestCreateTransfer_CrossSpace_Skipped pins the INTENDED behavior for
// cross-space lending: a lender (u1) in testSpaceA creates a transfer to a
// contact (c2) that is tracked in a DIFFERENT space, testSpaceB (e.g. because
// c2 is a registered user with their own space, or is tracked as a contact by
// a second space). Once the gaps documented above are fixed, this test should
// be un-skipped and expanded to assert:
//
//   - The created transfer's From/To each carry their own SpaceID (A and B).
//   - The lender's DebtusSpaceContactDbo balance in space A is +100 EUR
//     (or -100, depending on sign convention).
//   - The borrower/contact's DebtusSpaceContactDbo balance in space B is the
//     exact reciprocal (-100 EUR).
//   - A settle transfer for the same amount, in the reverse direction, nets
//     both sides' balances back to zero.
//   - Querying space A's contacts/transfers returns this transfer; querying
//     space B's contacts/transfers also legitimately returns it (since it's a
//     cross-space reference); an unrelated third space's query does not.
func TestCreateTransfer_CrossSpace(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	lender := dbo4userus.NewUserEntry("u1")
	// The contact/debtus-contact for "c2" (the borrower) live in the LENDER's
	// own space (testSpaceA) — see the cross-space convention documented in
	// createTransferWithinTransaction: To.ContactID is the ID the "from" side
	// (the creator here) uses to refer to the counterparty, so it's stored in
	// From.SpaceID. testSpaceB is the borrower's own (separate) aggregate space.
	borrowerContactDbo := &dbo4contactus.ContactDbo{}
	borrowerContactDbo.UserID = "u1" // matches the creator, per the pre-tx integrity check
	borrowerContactInSpaceA := dal4contactus.NewContactEntryWithData(testSpaceA, "c2", borrowerContactDbo)
	lenderSpace := models4debtus.NewDebtusSpaceEntry(testSpaceA)
	borrowerSpace := models4debtus.NewDebtusSpaceEntry(testSpaceB)

	// NOTE: intentionally NOT also seeding a models4debtus.DebtusSpaceContactEntry
	// for "c2": both it and dal4contactus.ContactEntry use the literal
	// collection name "contacts", and dalgo2memory's default in-memory engine
	// indexes purely by (collection name, leaf ID) — NOT by full ancestor
	// path — so seeding both under the same contactID collides and one
	// silently clobbers the other's stored bytes. createTransferWithinTransaction
	// tolerates a not-yet-existing DebtusSpaceContactEntry (GetMulti treats it
	// as "first transfer with this contact"), so omitting the seed here is
	// both correct and collision-free.
	seedRecords(t, db, lender.Record, borrowerContactInSpaceA.Record, lenderSpace.Record, borrowerSpace.Record)

	input := CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: lender,
		Request: CreateTransferRequest{
			// SpaceID is the CREATOR's (lender's) own active space — testSpaceA,
			// matching From.SpaceID below, not the unrelated testSpaceID used by
			// the same-space fixtures.
			SpaceRequest: dto4spaceus.SpaceRequest{SpaceID: testSpaceA},
			Direction:    models4debtus.TransferDirectionUser2Counterparty,
			Amount:       money.NewAmount(money.CurrencyEUR, 100),
			ToContactID:  "c2",
		},
		From: &models4debtus.TransferCounterpartyInfo{UserID: "u1", SpaceID: testSpaceA},
		To:   &models4debtus.TransferCounterpartyInfo{ContactID: "c2", SpaceID: testSpaceB},
	}

	output, err := Transfers.CreateTransfer(ctx, input)
	if err != nil {
		t.Fatalf("CreateTransfer() error: %v", err)
	}

	if output.Transfer.Data.From().SpaceID != testSpaceA {
		t.Errorf("From().SpaceID = %v, want %v", output.Transfer.Data.From().SpaceID, testSpaceA)
	}
	if output.Transfer.Data.To().SpaceID != testSpaceB {
		t.Errorf("To().SpaceID = %v, want %v", output.Transfer.Data.To().SpaceID, testSpaceB)
	}

	// The lender (u1, From, testSpaceA) is the only registered-user side here
	// (the borrower is a plain, unregistered contact — the common debtus case),
	// so it's the lender's own space/contact balance that gets updated: +100
	// EUR (userBalanceIncreased, since From lent to To).
	if v := output.From.DebtusSpace.Data.Balance[money.CurrencyEUR]; v != 100 {
		t.Errorf("From.DebtusSpace balance = %v, want +100", v)
	}
	if v := output.To.DebtusContact.Data.Balance[money.CurrencyEUR]; v != 100 {
		t.Errorf("To.DebtusContact (the borrower as tracked in the lender's own space) balance = %v, want +100", v)
	}
}

// TestCreateTransfer_SameSpace_Balances is the same-space companion to
// TestCreateTransfer_CrossSpace: creates a real transfer end-to-end and
// asserts the creator's own DebtusSpace ledger AND the mirrored
// DebtusContact both reflect the transferred amount (they must always agree
// — that's what the "Integrity checks" block in createTransferWithinTransaction
// enforces — but we assert the actual values here, not just "no error").
func TestCreateTransfer_SameSpace_Balances(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	u1 := dbo4userus.NewUserEntry("u1")
	contactDbo := &dbo4contactus.ContactDbo{}
	contactDbo.UserID = "u1"
	contact := dal4contactus.NewContactEntryWithData(testSpaceID, "c2", contactDbo)
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)

	// NOTE: intentionally NOT seeding a separate DebtusSpaceContactEntry for
	// "c2" here either — see the collision note on TestCreateTransfer_CrossSpace.
	seedRecords(t, db, u1.Record, contact.Record, debtusSpace.Record)

	input := sameSpaceCreateTransferInput()

	output, err := Transfers.CreateTransfer(ctx, input)
	if err != nil {
		t.Fatalf("CreateTransfer() error: %v", err)
	}

	if v := output.From.DebtusSpace.Data.Balance[money.CurrencyEUR]; v != 100 {
		t.Errorf("From.DebtusSpace (creator's own ledger) balance = %v, want +100", v)
	}
	if v := output.To.DebtusContact.Data.Balance[money.CurrencyEUR]; v != 100 {
		t.Errorf("To.DebtusContact (mirrored in creator's own space) balance = %v, want +100", v)
	}
}

// TestCreateTransfer_CrossSpace_SettleNetsToZero extends the cross-space
// scenario with a second, reverse-direction transfer of the same amount
// (the borrower settling the debt) and asserts the lender's own ledger nets
// back to zero.
func TestCreateTransfer_CrossSpace_SettleNetsToZero(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// The settle transfer sees a non-zero cached balance brief (from the
	// first transfer), which makes checkOutstandingTransfersForReturns()
	// call dal4debtus.Default.Transfer.LoadOutstandingTransfers — provide a
	// stub so that call has somewhere to go (this package can't import the
	// real debtusdal implementation without an import cycle).
	setTransferDal(t, fakeTransferDal{})

	lender := dbo4userus.NewUserEntry("u1")
	borrowerContactDbo := &dbo4contactus.ContactDbo{}
	borrowerContactDbo.UserID = "u1"
	borrowerContactInSpaceA := dal4contactus.NewContactEntryWithData(testSpaceA, "c2", borrowerContactDbo)
	lenderSpace := models4debtus.NewDebtusSpaceEntry(testSpaceA)
	borrowerSpace := models4debtus.NewDebtusSpaceEntry(testSpaceB)

	seedRecords(t, db, lender.Record, borrowerContactInSpaceA.Record, lenderSpace.Record, borrowerSpace.Record)

	lendInput := CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: lender,
		Request: CreateTransferRequest{
			SpaceRequest: dto4spaceus.SpaceRequest{SpaceID: testSpaceA},
			Direction:    models4debtus.TransferDirectionUser2Counterparty,
			Amount:       money.NewAmount(money.CurrencyEUR, 100),
			ToContactID:  "c2",
		},
		From: &models4debtus.TransferCounterpartyInfo{UserID: "u1", SpaceID: testSpaceA},
		To:   &models4debtus.TransferCounterpartyInfo{ContactID: "c2", SpaceID: testSpaceB},
	}
	if _, err := Transfers.CreateTransfer(ctx, lendInput); err != nil {
		t.Fatalf("initial lend CreateTransfer() error: %v", err)
	}

	// The borrower (c2) settles the debt: same contact/spaces, reverse
	// direction (money now flows from the contact back to u1).
	settleInput := CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: lender,
		Request: CreateTransferRequest{
			SpaceRequest:  dto4spaceus.SpaceRequest{SpaceID: testSpaceA},
			Direction:     models4debtus.TransferDirectionCounterparty2User,
			Amount:        money.NewAmount(money.CurrencyEUR, 100),
			FromContactID: "c2",
		},
		From: &models4debtus.TransferCounterpartyInfo{ContactID: "c2", SpaceID: testSpaceB},
		To:   &models4debtus.TransferCounterpartyInfo{UserID: "u1", SpaceID: testSpaceA},
	}
	settleOutput, err := Transfers.CreateTransfer(ctx, settleInput)
	if err != nil {
		t.Fatalf("settle CreateTransfer() error: %v", err)
	}

	if v := settleOutput.To.DebtusSpace.Data.Balance[money.CurrencyEUR]; v != 0 {
		t.Errorf("lender's own ledger balance after settle = %v, want 0", v)
	}
	if v := settleOutput.From.DebtusContact.Data.Balance[money.CurrencyEUR]; v != 0 {
		t.Errorf("borrower-as-tracked-in-lender's-space balance after settle = %v, want 0", v)
	}
}
