package api4splitusbot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
)

func makeCreateSplitRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api4splitus/create-split", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Host = "localhost" // common4all.GetEnvironment panics on unknown hosts
	return r
}

// stubSplitContacts stubs getContactsByIDs to return a named contact for each
// requested ID whose name is listed in namesByContactID; IDs not listed get an
// entry with empty data — mimicking dalgo's GetMulti behavior for records that
// do not exist in the space.
func stubSplitContacts(t *testing.T, namesByContactID map[string]string) func() {
	t.Helper()
	orig := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, spaceID coretypes.SpaceID, contactIDs []string) ([]dal4contactus.ContactEntry, error) {
		contacts := make([]dal4contactus.ContactEntry, len(contactIDs))
		for i, id := range contactIDs {
			contact := dal4contactus.NewContactEntry(spaceID, id)
			if name, ok := namesByContactID[id]; ok {
				contact.Data.SetName("full", name)
			}
			contacts[i] = contact
		}
		return contacts, nil
	}
	return func() { getContactsByIDs = orig }
}

// stubCreateBillCapture stubs createBill + runReadwriteTransaction, capturing
// the BillDbo passed to the facade and returning a bill entry with ID "bill1".
func stubCreateBillCapture(t *testing.T, captured **models4splitus.BillDbo) func() {
	t.Helper()
	origCreate := createBill
	createBill = func(_ context.Context, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, dbo *models4splitus.BillDbo) (models4splitus.BillEntry, error) {
		if captured != nil {
			*captured = dbo
		}
		return newBillEntry("bill1", dbo), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(ctx, nil)
	}
	return func() {
		createBill = origCreate
		runReadwriteTransaction = origTx
	}
}

// stubCreateDebtusTransferCapture stubs the Debtus transfer seam, capturing
// every CreateTransferInput and returning transfers with IDs "transfer1", ….
func stubCreateDebtusTransferCapture(t *testing.T, captured *[]facade4debtus.CreateTransferInput) func() {
	t.Helper()
	orig := createDebtusTransfer
	createDebtusTransfer = func(_ context.Context, input facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		*captured = append(*captured, input)
		id := "transfer" + string(rune('0'+len(*captured)))
		return facade4debtus.CreateTransferOutput{Transfer: models4debtus.NewTransfer(id, new(models4debtus.TransferData))}, nil
	}
	return func() { createDebtusTransfer = orig }
}

func TestHandleCreateSplit_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleCreateSplit_NotMember verifies the SEC-6 seam: an authenticated
// caller who is not a member of the requested space gets 403 and neither a
// bill nor any Debtus transfer is created.
func TestHandleCreateSplit_NotMember(t *testing.T) {
	defer stubSpaceMembership(t, false)()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "outsider"})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
	if capturedBill != nil {
		t.Error("no bill must be created for a non-member")
	}
	if len(transferInputs) != 0 {
		t.Errorf("no transfers must be created for a non-member, got %d", len(transferInputs))
	}
}

func TestHandleCreateSplit_Validation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "InvalidJSON", body: `not-json`},
		{name: "MissingSpaceID", body: `{"currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`},
		{name: "MissingAmount", body: `{"spaceID":"space1","currency":"EUR","participantContactIDs":["cBea"]}`},
		{name: "InvalidAmount", body: `{"spaceID":"space1","currency":"EUR","amount":"abc","participantContactIDs":["cBea"]}`},
		{name: "NegativeAmount", body: `{"spaceID":"space1","currency":"EUR","amount":"-1.00","participantContactIDs":["cBea"]}`},
		{name: "MissingCurrency", body: `{"spaceID":"space1","amount":"90.00","participantContactIDs":["cBea"]}`},
		{name: "NoParticipants", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":[]}`},
		{name: "EmptyParticipantID", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":[""]}`},
		{name: "DuplicateParticipants", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea","cBea"]}`},
		{name: "UnsupportedSplitMode", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"shares","participantContactIDs":["cBea"]}`},
		{name: "ExactAmountMissingShares", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"exact-amount","participantContactIDs":["cBea"]}`},
		{name: "PercentageMissingShares", body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"percentage","participantContactIDs":["cBea"]}`},
		{
			name: "ExactAmountSharesDontSum",
			body: `{"spaceID":"space1","currency":"EUR","amount":"100.00","splitMode":"exact-amount","participantContactIDs":["cBea","cCam"],` +
				`"shares":[{"contactID":"","amount":"40.00"},{"contactID":"cBea","amount":"35.00"},{"contactID":"cCam","amount":"24.99"}]}`,
		},
		{
			name: "PercentageSharesDontSum",
			body: `{"spaceID":"space1","currency":"EUR","amount":"100.00","splitMode":"percentage","participantContactIDs":["cBea","cCam"],` +
				`"shares":[{"contactID":"","percent":"33.33"},{"contactID":"cBea","percent":"33.33"},{"contactID":"cCam","percent":"33.33"}]}`,
		},
		{
			name: "ExactAmountShareForUnknownContact",
			body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"exact-amount","participantContactIDs":["cBea"],` +
				`"shares":[{"contactID":"","amount":"45.00"},{"contactID":"cUnknown","amount":"45.00"}]}`,
		},
		{
			name: "ExactAmountMissingPayerShare",
			body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"exact-amount","participantContactIDs":["cBea"],` +
				`"shares":[{"contactID":"cBea","amount":"90.00"}]}`,
		},
		{
			name: "ExactAmountDuplicateShare",
			body: `{"spaceID":"space1","currency":"EUR","amount":"90.00","splitMode":"exact-amount","participantContactIDs":["cBea"],` +
				`"shares":[{"contactID":"","amount":"45.00"},{"contactID":"cBea","amount":"20.00"},{"contactID":"cBea","amount":"25.00"}]}`,
		},
	}

	defer stubSpaceMembership(t, true)()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleCreateSplit(context.Background(), w, makeCreateSplitRequest(tt.body), token4auth.AuthInfo{UserID: "user1"})
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleCreateSplit_ParticipantNotSpaceContact verifies that a participant
// contact ID that does not resolve to an existing contact of the space is
// rejected with 400 and nothing is persisted.
func TestHandleCreateSplit_ParticipantNotSpaceContact(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea"})() // cCam does not exist
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea","cCam"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "cCam") {
		t.Errorf("error message should name the unknown contact ID, got: %s", w.Body.String())
	}
	if capturedBill != nil {
		t.Error("no bill must be created when a participant is not a space contact")
	}
	if len(transferInputs) != 0 {
		t.Errorf("no transfers must be created when a participant is not a space contact, got %d", len(transferInputs))
	}
}

// TestHandleCreateSplit_EqualSplitPostsDebtusTransfers verifies the core AC:
// a 90.00 expense paid by the current user and split equally among the payer
// and two contacts posts exactly two 30.00 u2c Debtus transfers (one per
// non-payer participant) and persists a Bill that carries no balance of its
// own — the who-owes-what records are the Debtus transfers.
func TestHandleCreateSplit_EqualSplitPostsDebtusTransfers(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","title":"Dinner","currency":"EUR","amount":"90.00","participantContactIDs":["cBea","cCam"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// One Debtus transfer per non-payer participant, direction u2c, 30.00 each.
	if len(transferInputs) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transferInputs))
	}
	expectedAmounts := map[string]decimal.Decimal64p2{"cBea": 3000, "cCam": 3000}
	for i, input := range transferInputs {
		contactID := input.Request.ToContactID
		expected, ok := expectedAmounts[contactID]
		if !ok {
			t.Errorf("transfers[%d]: unexpected ToContactID %q", i, contactID)
			continue
		}
		delete(expectedAmounts, contactID)
		if input.Request.Direction != models4debtus.TransferDirectionUser2Counterparty {
			t.Errorf("transfers[%d]: expected direction u2c, got %q", i, input.Request.Direction)
		}
		if input.Request.Amount.Value != expected {
			t.Errorf("transfers[%d]: expected amount %v, got %v", i, expected, input.Request.Amount.Value)
		}
		if input.Request.Amount.Currency != "EUR" {
			t.Errorf("transfers[%d]: expected currency EUR, got %q", i, input.Request.Amount.Currency)
		}
		if input.Request.SpaceID != "space1" {
			t.Errorf("transfers[%d]: expected spaceID space1, got %q", i, input.Request.SpaceID)
		}
		if input.Request.BillID != "bill1" {
			t.Errorf("transfers[%d]: expected billID bill1, got %q", i, input.Request.BillID)
		}
		if input.From == nil || input.From.UserID != "user1" {
			t.Errorf("transfers[%d]: expected From.UserID user1, got %+v", i, input.From)
		}
		if input.To == nil || input.To.ContactID != contactID {
			t.Errorf("transfers[%d]: expected To.ContactID %q, got %+v", i, contactID, input.To)
		}
		if input.CreatorUser.ID != "user1" {
			t.Errorf("transfers[%d]: expected CreatorUser.ID user1, got %q", i, input.CreatorUser.ID)
		}
	}
	if len(expectedAmounts) != 0 {
		t.Errorf("missing transfers for contacts: %v", expectedAmounts)
	}

	// The bill is persisted with an equal split and no balance of its own.
	if capturedBill == nil {
		t.Fatal("expected a bill to be persisted")
	}
	if capturedBill.SplitMode != models4splitus.SplitModeEqually {
		t.Errorf("expected split mode equally, got %q", capturedBill.SplitMode)
	}
	if capturedBill.SpaceID != "space1" {
		t.Errorf("expected bill.SpaceID space1, got %q", capturedBill.SpaceID)
	}
	if capturedBill.AmountTotal != 9000 {
		t.Errorf("expected bill.AmountTotal 90.00, got %v", capturedBill.AmountTotal)
	}
	members := capturedBill.GetBillMembers()
	if len(members) != 3 {
		t.Fatalf("expected 3 bill members (payer + 2 contacts), got %d", len(members))
	}
	var payerPaid decimal.Decimal64p2
	for i, m := range members {
		if m.Owes != 3000 {
			t.Errorf("members[%d]: expected Owes 30.00, got %v", i, m.Owes)
		}
		if m.UserID == "user1" {
			payerPaid = m.Paid
		} else if m.Paid != 0 {
			t.Errorf("members[%d]: non-payer must not have Paid, got %v", i, m.Paid)
		}
	}
	if payerPaid != 9000 {
		t.Errorf("expected payer Paid 90.00, got %v", payerPaid)
	}
	// No splitus-owned who-owes-what state beyond the split definition itself.
	if len(capturedBill.SettlementIDs) != 0 {
		t.Errorf("bill must not carry settlement state, got %v", capturedBill.SettlementIDs)
	}

	// Response references the bill and the Debtus transfers.
	body := w.Body.String()
	for _, expect := range []string{"bill1", "transfer1", "transfer2"} {
		if !strings.Contains(body, expect) {
			t.Errorf("response should contain %q, got: %s", expect, body)
		}
	}
}

// TestHandleCreateSplit_RemainderCents verifies deterministic distribution of
// remainder cents: 100.00 across 3 participants leaves the payer owing 33.34
// and posts two 33.33 transfers, summing exactly to 100.00.
func TestHandleCreateSplit_RemainderCents(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"100.00","participantContactIDs":["cBea","cCam"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	if len(transferInputs) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transferInputs))
	}
	var total decimal.Decimal64p2
	for i, input := range transferInputs {
		if input.Request.Amount.Value != 3333 {
			t.Errorf("transfers[%d]: expected amount 33.33, got %v", i, input.Request.Amount.Value)
		}
		total += input.Request.Amount.Value
	}
	if capturedBill == nil {
		t.Fatal("expected a bill to be persisted")
	}
	for _, m := range capturedBill.GetBillMembers() {
		if m.UserID == "user1" {
			if m.Owes != 3334 {
				t.Errorf("expected payer to absorb the remainder cent (33.34), got %v", m.Owes)
			}
			total += m.Owes
		}
	}
	if total != 10000 {
		t.Errorf("shares must sum exactly to 100.00, got %v", total)
	}
}

// TestHandleCreateSplit_ExactAmountSplitPostsDebtusTransfers verifies AC
// splitus#ac:custom-shares-must-sum's happy path: custom exact-amount shares
// that reconcile exactly to the 100.00 total (40.00/35.00/25.00) are used
// verbatim — no equal-split remainder logic applies — and post transfers for
// exactly what each contact was assigned.
func TestHandleCreateSplit_ExactAmountSplitPostsDebtusTransfers(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","title":"Dinner","currency":"EUR","amount":"100.00","splitMode":"exact-amount",` +
		`"participantContactIDs":["cBea","cCam"],` +
		`"shares":[{"contactID":"","amount":"40.00"},{"contactID":"cBea","amount":"35.00"},{"contactID":"cCam","amount":"25.00"}]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	if len(transferInputs) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transferInputs))
	}
	expectedAmounts := map[string]decimal.Decimal64p2{"cBea": 3500, "cCam": 2500}
	for i, input := range transferInputs {
		contactID := input.Request.ToContactID
		expected, ok := expectedAmounts[contactID]
		if !ok {
			t.Errorf("transfers[%d]: unexpected ToContactID %q", i, contactID)
			continue
		}
		delete(expectedAmounts, contactID)
		if input.Request.Amount.Value != expected {
			t.Errorf("transfers[%d]: expected amount %v, got %v", i, expected, input.Request.Amount.Value)
		}
	}
	if len(expectedAmounts) != 0 {
		t.Errorf("missing transfers for contacts: %v", expectedAmounts)
	}

	if capturedBill == nil {
		t.Fatal("expected a bill to be persisted")
	}
	if capturedBill.SplitMode != models4splitus.SplitModeExactAmount {
		t.Errorf("expected bill.SplitMode exact-amount, got %q", capturedBill.SplitMode)
	}
	var total decimal.Decimal64p2
	for _, m := range capturedBill.GetBillMembers() {
		total += m.Owes
		if m.UserID == "user1" {
			if m.Owes != 4000 {
				t.Errorf("expected payer Owes 40.00, got %v", m.Owes)
			}
			if m.Paid != 10000 {
				t.Errorf("expected payer Paid 100.00, got %v", m.Paid)
			}
		}
	}
	if total != 10000 {
		t.Errorf("shares must sum exactly to 100.00, got %v", total)
	}
}

// TestHandleCreateSplit_ExactAmountSharesDontSum_Rejected verifies AC
// splitus#ac:custom-shares-must-sum: exact shares that don't sum to the
// total (99.99 vs 100.00) are rejected with an error naming the discrepancy,
// and nothing is persisted.
func TestHandleCreateSplit_ExactAmountSharesDontSum_Rejected(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"100.00","splitMode":"exact-amount",` +
		`"participantContactIDs":["cBea","cCam"],` +
		`"shares":[{"contactID":"","amount":"40.00"},{"contactID":"cBea","amount":"35.00"},{"contactID":"cCam","amount":"24.99"}]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "99.99") || !strings.Contains(body, "100.00") {
		t.Errorf("error should name the discrepancy (99.99 vs 100.00), got: %s", body)
	}
	if capturedBill != nil {
		t.Error("no bill must be created when shares don't sum to the total")
	}
	if len(transferInputs) != 0 {
		t.Errorf("no transfers must be created when shares don't sum to the total, got %d", len(transferInputs))
	}
}

// TestHandleCreateSplit_PercentageSharesDontSum_Rejected verifies AC
// splitus#ac:custom-shares-must-sum: percentages that don't sum to 100%
// (99.99%) are rejected and nothing is persisted.
func TestHandleCreateSplit_PercentageSharesDontSum_Rejected(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"100.00","splitMode":"percentage",` +
		`"participantContactIDs":["cBea","cCam"],` +
		`"shares":[{"contactID":"","percent":"33.33"},{"contactID":"cBea","percent":"33.33"},{"contactID":"cCam","percent":"33.33"}]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "99.99") {
		t.Errorf("error should name the discrepancy (percentages sum to 99.99%%), got: %s", body)
	}
	if capturedBill != nil {
		t.Error("no bill must be created when percentages don't sum to 100%")
	}
	if len(transferInputs) != 0 {
		t.Errorf("no transfers must be created when percentages don't sum to 100%%, got %d", len(transferInputs))
	}
}

// TestHandleCreateSplit_PercentageSplitLargestRemainder verifies AC
// splitus#ac:custom-shares-must-sum's rounding requirement: percentages that
// sum exactly to 100% (33.34/33.33/33.33) but don't divide the 10.00 total
// evenly in cents are converted via a deterministic largest-remainder rule
// so the computed amounts still sum EXACTLY to the total — the extra cent
// goes to the largest percentage (the payer, 33.34%).
func TestHandleCreateSplit_PercentageSplitLargestRemainder(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea", "cCam": "Cam"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"10.00","splitMode":"percentage",` +
		`"participantContactIDs":["cBea","cCam"],` +
		`"shares":[{"contactID":"","percent":"33.34"},{"contactID":"cBea","percent":"33.33"},{"contactID":"cCam","percent":"33.33"}]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	if len(transferInputs) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transferInputs))
	}
	var total decimal.Decimal64p2
	for i, input := range transferInputs {
		if input.Request.Amount.Value != 333 {
			t.Errorf("transfers[%d]: expected amount 3.33, got %v", i, input.Request.Amount.Value)
		}
		total += input.Request.Amount.Value
	}
	if capturedBill == nil {
		t.Fatal("expected a bill to be persisted")
	}
	if capturedBill.SplitMode != models4splitus.SplitModePercentage {
		t.Errorf("expected bill.SplitMode percentage, got %q", capturedBill.SplitMode)
	}
	for _, m := range capturedBill.GetBillMembers() {
		if m.UserID == "user1" {
			if m.Owes != 334 {
				t.Errorf("expected payer to absorb the remainder cent (3.34) as the largest percentage, got %v", m.Owes)
			}
			total += m.Owes
		}
	}
	if total != 1000 {
		t.Errorf("shares must sum exactly to 10.00, got %v", total)
	}
}

func TestHandleCreateSplit_CreateBillError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea"})()
	var transferInputs []facade4debtus.CreateTransferInput
	defer stubCreateDebtusTransferCapture(t, &transferInputs)()

	origCreate := createBill
	createBill = func(_ context.Context, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ *models4splitus.BillDbo) (models4splitus.BillEntry, error) {
		return models4splitus.BillEntry{}, errors.New("create bill db error")
	}
	defer func() { createBill = origCreate }()
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(ctx, nil)
	}
	defer func() { runReadwriteTransaction = origTx }()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
	if len(transferInputs) != 0 {
		t.Errorf("no transfers must be created when bill creation fails, got %d", len(transferInputs))
	}
}

func TestHandleCreateSplit_TransferError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubNamedUsers(t)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea"})()
	var capturedBill *models4splitus.BillDbo
	defer stubCreateBillCapture(t, &capturedBill)()

	orig := createDebtusTransfer
	createDebtusTransfer = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		return facade4debtus.CreateTransferOutput{}, errors.New("debtus transfer error")
	}
	defer func() { createDebtusTransfer = orig }()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// --- error-path coverage ---

func TestHandleCreateSplit_MembershipCheckError(t *testing.T) {
	origMembership := userBelongsToSpace
	userBelongsToSpace = func(_ context.Context, _ string, _ coretypes.SpaceID) (bool, error) {
		return false, errors.New("space db error")
	}
	defer func() { userBelongsToSpace = origMembership }()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateSplit_GetContactsError(t *testing.T) {
	defer stubSpaceMembership(t, true)()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{name: "GenericError", err: errors.New("contacts db error"), expectedStatus: http.StatusInternalServerError},
		{name: "NotFound", err: dal.NewErrNotFoundByKey(dal.NewKeyWithID("contacts", "cBea"), errors.New("no such record")), expectedStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origContacts := getContactsByIDs
			getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
				return nil, tt.err
			}
			defer func() { getContactsByIDs = origContacts }()

			w := httptest.NewRecorder()
			r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
			handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d — body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleCreateSplit_GetUsersError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea"})()

	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return nil, errors.New("users db error")
	}
	defer func() { getUsersByIDs = origUsers }()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateSplit_PayerNotFound(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubSplitContacts(t, map[string]string{"cBea": "Bea"})()

	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return nil, nil // payer user record does not exist
	}
	defer func() { getUsersByIDs = origUsers }()

	w := httptest.NewRecorder()
	r := makeCreateSplitRequest(`{"spaceID":"space1","currency":"EUR","amount":"90.00","participantContactIDs":["cBea"]}`)
	handleCreateSplit(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSplitusTransferSource_PopulateTransfer(t *testing.T) {
	var data models4debtus.TransferData
	splitusTransferSource{createdOnID: "host1"}.PopulateTransfer(&data)
	if data.CreatedOnPlatform != "api4splitus" {
		t.Errorf("expected CreatedOnPlatform api4splitus, got %q", data.CreatedOnPlatform)
	}
	if data.CreatedOnID != "host1" {
		t.Errorf("expected CreatedOnID host1, got %q", data.CreatedOnID)
	}
}
