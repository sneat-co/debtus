package facade4debtus

import (
	"context"
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
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
// ============================================================================

const (
	testSpaceA coretypes.SpaceID = "spaceA-lender"
	testSpaceB coretypes.SpaceID = "spaceB-borrower"
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
func TestCreateTransfer_CrossSpace_Skipped(t *testing.T) {
	t.Skip("cross-space CreateTransfer is not wired end-to-end yet: " +
		"createTransferWithinTransaction never populates ParticipantEntries." +
		"DebtusContact/DebtusSpace/SpaceID for either side (transfer_facade.go), " +
		"so it cannot yet resolve/update a counterparty living in a different " +
		"space. See the package doc comment in this file and the PR gap report " +
		"for the concrete fix proposal.")

	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	lender := dbo4userus.NewUserEntry("u1")
	borrowerContactInSpaceA := dal4contactus.NewContactEntryWithData(testSpaceA, "c2", &dbo4contactus.ContactDbo{})
	lenderSpace := models4debtus.NewDebtusSpaceEntry(testSpaceA)
	borrowerSpace := models4debtus.NewDebtusSpaceEntry(testSpaceB)

	seedRecords(t, db, lender.Record, borrowerContactInSpaceA.Record, lenderSpace.Record, borrowerSpace.Record)

	input := CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: lender,
		Request: CreateTransferRequest{
			SpaceRequest: validCreateTransferRequest().SpaceRequest,
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

	// TODO once unskipped: assert reciprocal balances in space A vs space B,
	// and that a settle transfer nets both to zero.
}
