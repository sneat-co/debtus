package api4splitusbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
	"github.com/strongo/strongoapp"
)

// --- fixtures ---

const (
	splitBillID      = "bill1"
	splitSpaceID     = coretypes.SpaceID("space1")
	splitPayerUserID = "payer1"
)

// splitBill builds a 60.00 EUR bill paid by payer1, split equally with two
// contacts (cBea, cCat) owing 20.00 each — the Task-1 shape handleCreateSplit
// persists.
func splitBill() models4splitus.BillEntry {
	return newBillEntry(splitBillID, &models4splitus.BillDbo{
		BillCommon: models4splitus.BillCommon{
			SpaceID:       splitSpaceID,
			Status:        models4splitus.BillStatusOutstanding,
			SplitMode:     models4splitus.SplitModeEqually,
			CreatorUserID: splitPayerUserID,
			Name:          "Dinner",
			Currency:      "EUR",
			AmountTotal:   decimal.Decimal64p2(6000),
			Members: []*briefs4splitus.BillMemberBrief{
				{
					MemberBrief: briefs4splitus.MemberBrief{ID: "1", UserID: splitPayerUserID, Name: "Alex"},
					Paid:        decimal.Decimal64p2(6000),
					Owes:        decimal.Decimal64p2(2000),
				},
				{
					MemberBrief: briefs4splitus.MemberBrief{ID: "2", Name: "Bea", ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
						splitPayerUserID: {ContactID: "cBea", ContactName: "Bea"},
					}},
					Owes: decimal.Decimal64p2(2000),
				},
				{
					MemberBrief: briefs4splitus.MemberBrief{ID: "3", Name: "Cat", ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
						splitPayerUserID: {ContactID: "cCat", ContactName: "Cat"},
					}},
					Owes: decimal.Decimal64p2(2000),
				},
			},
		},
	})
}

// splitDebtTransfer builds a Debtus u2c debt transfer linked to splitBillID:
// payer1 lent `amount` to contactID, of which `returned` has been returned
// (settled) in Debtus. returned == amount means fully settled.
func splitDebtTransfer(id, contactID string, amount, returned decimal.Decimal64p2) models4debtus.TransferEntry {
	td := &models4debtus.TransferData{
		CreatorUserID:         splitPayerUserID,
		Currency:              "EUR",
		AmountInCents:         amount,
		AmountInCentsReturned: returned,
		IsOutstanding:         returned < amount,
		BillIDs:               []string{splitBillID},
		FromJson:              fmt.Sprintf(`{"userID":%q}`, splitPayerUserID),
		ToJson:                fmt.Sprintf(`{"contactID":%q}`, contactID),
	}
	return models4debtus.NewTransfer(id, td)
}

// --- seam stubs ---

func stubGetBillByID(t *testing.T, bill models4splitus.BillEntry, err error) func() {
	t.Helper()
	orig := getBillByID
	getBillByID = func(_ context.Context, _ dal.ReadSession, billID string) (models4splitus.BillEntry, error) {
		if err != nil {
			return models4splitus.BillEntry{}, err
		}
		if billID != bill.ID {
			return models4splitus.BillEntry{}, fmt.Errorf("%w: id=%s", dal.ErrRecordNotFound, billID)
		}
		return bill, nil
	}
	return func() { getBillByID = orig }
}

func stubBillTransfers(t *testing.T, transfers []models4debtus.TransferEntry, err error) func() {
	t.Helper()
	orig := getBillTransfers
	getBillTransfers = func(_ context.Context, _ coretypes.SpaceID, _ string) ([]models4debtus.TransferEntry, error) {
		return transfers, err
	}
	return func() { getBillTransfers = orig }
}

func stubListBillsBySpace(t *testing.T, bills []models4splitus.BillEntry, err error) func() {
	t.Helper()
	orig := listBillsBySpace
	listBillsBySpace = func(_ context.Context, _ coretypes.SpaceID) ([]models4splitus.BillEntry, error) {
		return bills, err
	}
	return func() { listBillsBySpace = orig }
}

func getSplitRequest(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/api4splitus/split"+query, nil)
}

func getSplitsRequest(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/api4splitus/splits"+query, nil)
}

// --- GET /api4splitus/split: auth & param errors ---

func TestHandleGetSplit_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplit_MissingParams(t *testing.T) {
	for name, query := range map[string]string{
		"missing_spaceID": "?id=bill1",
		"missing_id":      "?spaceID=space1",
		"missing_both":    "",
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleGetSplit(context.Background(), w, getSplitRequest(query), token4auth.AuthInfo{UserID: "u1"})
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleGetSplit_NotMember verifies the SEC-6 read-authz half of
// splitus#ac:participants-from-contactus-membership-enforced: a user who is
// not a member of the space receives an authorization error when attempting
// to READ a split — and the bill is never even loaded.
func TestHandleGetSplit_NotMember(t *testing.T) {
	defer stubSpaceMembership(t, false)()
	billLoaded := false
	orig := getBillByID
	getBillByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4splitus.BillEntry, error) {
		billLoaded = true
		return splitBill(), nil
	}
	defer func() { getBillByID = orig }()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: "outsider"})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
	if billLoaded {
		t.Error("bill must not be loaded for a non-member")
	}
}

func TestHandleGetSplit_MembershipCheckError(t *testing.T) {
	orig := userBelongsToSpace
	userBelongsToSpace = func(_ context.Context, _ string, _ coretypes.SpaceID) (bool, error) {
		return false, errors.New("space db error")
	}
	defer func() { userBelongsToSpace = orig }()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplit_UnknownBill(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)() // knows only bill1

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=no-such-bill"), token4auth.AuthInfo{UserID: splitPayerUserID})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplit_GetBillError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, models4splitus.BillEntry{}, errors.New("db error"))()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetSplit_BillFromAnotherSpace verifies a bill belonging to a
// different space than the requested one is reported as 404 (no cross-space
// existence leak), even to a member of the requested space.
func TestHandleGetSplit_BillFromAnotherSpace(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=other_space&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplit_TransfersError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	defer stubBillTransfers(t, nil, errors.New("transfers query failed"))()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// --- GET /api4splitus/split: settled-state read-through (AC:
// splitus#ac:settle-up-single-source-of-truth) ---

// decodeGetSplitResponse unmarshals the handler response and indexes
// participants by name for assertion convenience.
func decodeGetSplitResponse(t *testing.T, w *httptest.ResponseRecorder) (getSplitResponse, map[string]splitParticipantDto) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var response getSplitResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v — body: %s", err, w.Body.String())
	}
	byName := make(map[string]splitParticipantDto, len(response.Participants))
	for _, p := range response.Participants {
		byName[p.Name] = p
	}
	return response, byName
}

// TestHandleGetSplit_SettledStateDerivedFromDebtus is the core scenario of
// splitus#ac:settle-up-single-source-of-truth: Bea's transfer has been fully
// returned (settled) in Debtus, Cat's has not — Splitus reports Bea settled
// and Cat outstanding purely by reading the Debtus transfers; the Bill (whose
// stored status remains "outstanding" and holds no settled flag) is not
// consulted for settled state and not written to.
func TestHandleGetSplit_SettledStateDerivedFromDebtus(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	defer stubBillTransfers(t, []models4debtus.TransferEntry{
		splitDebtTransfer("t1", "cBea", 2000, 2000), // fully returned → settled
		splitDebtTransfer("t2", "cCat", 2000, 0),    // unreturned → outstanding
	}, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	response, byName := decodeGetSplitResponse(t, w)

	if bea := byName["Bea"]; bea.Status != splitShareStatusSettled {
		t.Errorf("Bea (fully returned in Debtus) status = %q, want %q", bea.Status, splitShareStatusSettled)
	} else if bea.ContactID != "cBea" || bea.Share != 2000 {
		t.Errorf("Bea = %+v, want contactID=cBea share=2000", bea)
	}
	if cat := byName["Cat"]; cat.Status != splitShareStatusOutstanding {
		t.Errorf("Cat (unreturned in Debtus) status = %q, want %q", cat.Status, splitShareStatusOutstanding)
	}
	if payer := byName["Alex"]; payer.Status != splitShareStatusSettled || !payer.IsPayer {
		t.Errorf("payer = %+v, want settled payer", payer)
	}
	if response.Status != models4splitus.BillStatusOutstanding {
		t.Errorf("split status = %q, want %q while Cat is outstanding", response.Status, models4splitus.BillStatusOutstanding)
	}
}

// TestHandleGetSplit_PartiallyReturnedIsOutstanding: a partial return in
// Debtus leaves the share outstanding.
func TestHandleGetSplit_PartiallyReturnedIsOutstanding(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	defer stubBillTransfers(t, []models4debtus.TransferEntry{
		splitDebtTransfer("t1", "cBea", 2000, 1500), // partial return
		splitDebtTransfer("t2", "cCat", 2000, 2000),
	}, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	_, byName := decodeGetSplitResponse(t, w)

	if bea := byName["Bea"]; bea.Status != splitShareStatusOutstanding {
		t.Errorf("Bea (partially returned) status = %q, want %q", bea.Status, splitShareStatusOutstanding)
	}
	if cat := byName["Cat"]; cat.Status != splitShareStatusSettled {
		t.Errorf("Cat (fully returned) status = %q, want %q", cat.Status, splitShareStatusSettled)
	}
}

// TestHandleGetSplit_AllSettled: when every share's transfer is fully
// returned in Debtus the whole split reads as settled — derived on read,
// never written to the Bill (the stubbed bill still stores "outstanding").
func TestHandleGetSplit_AllSettled(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	bill := splitBill()
	defer stubGetBillByID(t, bill, nil)()
	defer stubBillTransfers(t, []models4debtus.TransferEntry{
		splitDebtTransfer("t1", "cBea", 2000, 2000),
		splitDebtTransfer("t2", "cCat", 2000, 2000),
	}, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	response, byName := decodeGetSplitResponse(t, w)

	for _, name := range []string{"Alex", "Bea", "Cat"} {
		if p := byName[name]; p.Status != splitShareStatusSettled {
			t.Errorf("%s status = %q, want %q", name, p.Status, splitShareStatusSettled)
		}
	}
	if response.Status != models4splitus.BillStatusSettled {
		t.Errorf("split status = %q, want derived %q", response.Status, models4splitus.BillStatusSettled)
	}
	// Splitus never records a separate settled state of its own: the stored
	// bill must still say "outstanding" after the read.
	if bill.Data.Status != models4splitus.BillStatusOutstanding {
		t.Errorf("stored bill status mutated to %q — settled state must never be written to the Bill", bill.Data.Status)
	}
}

// TestHandleGetSplit_NoTransfersMeansOutstanding: with no Debtus transfers
// found for the bill there is no record of settlement, so non-payer shares
// stay outstanding (fail-safe default).
func TestHandleGetSplit_NoTransfersMeansOutstanding(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	defer stubBillTransfers(t, nil, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	_, byName := decodeGetSplitResponse(t, w)

	for _, name := range []string{"Bea", "Cat"} {
		if p := byName[name]; p.Status != splitShareStatusOutstanding {
			t.Errorf("%s status = %q, want %q with no transfers", name, p.Status, splitShareStatusOutstanding)
		}
	}
}

// TestHandleGetSplit_ReturnTransfersAreSkipped: a return-transfer that also
// carries the bill ID must not be mistaken for a settled debt record — only
// original (non-return) transfers determine each share's outstanding value.
func TestHandleGetSplit_ReturnTransfersAreSkipped(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	returnTransfer := splitDebtTransfer("t3", "cCat", 2000, 0)
	returnTransfer.Data.IsReturn = true // GetOutstandingValue()==0 for returns
	defer stubBillTransfers(t, []models4debtus.TransferEntry{
		splitDebtTransfer("t2", "cCat", 2000, 0), // the actual debt, unreturned
		returnTransfer,
	}, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	_, byName := decodeGetSplitResponse(t, w)

	if cat := byName["Cat"]; cat.Status != splitShareStatusOutstanding {
		t.Errorf("Cat status = %q, want %q — return-transfer must not mark the share settled", cat.Status, splitShareStatusOutstanding)
	}
}

// --- GET /api4splitus/splits ---

func TestHandleGetSplits_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplits_MissingSpaceID(t *testing.T) {
	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest(""), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetSplits_NotMember: SEC-6 read-authz for the list endpoint —
// splitus#ac:participants-from-contactus-membership-enforced.
func TestHandleGetSplits_NotMember(t *testing.T) {
	defer stubSpaceMembership(t, false)()
	listed := false
	orig := listBillsBySpace
	listBillsBySpace = func(_ context.Context, _ coretypes.SpaceID) ([]models4splitus.BillEntry, error) {
		listed = true
		return nil, nil
	}
	defer func() { listBillsBySpace = orig }()

	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{UserID: "outsider"})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
	if listed {
		t.Error("bills must not be listed for a non-member")
	}
}

func TestHandleGetSplits_MembershipCheckError(t *testing.T) {
	orig := userBelongsToSpace
	userBelongsToSpace = func(_ context.Context, _ string, _ coretypes.SpaceID) (bool, error) {
		return false, errors.New("space db error")
	}
	defer func() { userBelongsToSpace = orig }()

	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetSplits_ListError(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubListBillsBySpace(t, nil, errors.New("list failed"))()

	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetSplits_ReturnsSpaceBills: the list endpoint returns the
// space's bills as summaries (id, title, amount, currency, status, member
// count).
func TestHandleGetSplits_ReturnsSpaceBills(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	other := newBillEntry("bill2", &models4splitus.BillDbo{
		BillCommon: models4splitus.BillCommon{
			SpaceID:       splitSpaceID,
			Status:        models4splitus.BillStatusOutstanding,
			CreatorUserID: splitPayerUserID,
			Name:          "Taxi",
			Currency:      "EUR",
			AmountTotal:   decimal.Decimal64p2(900),
			Members: []*briefs4splitus.BillMemberBrief{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "1", UserID: splitPayerUserID, Name: "Alex"}},
				{MemberBrief: briefs4splitus.MemberBrief{ID: "2", Name: "Bea"}},
			},
		},
	})
	defer stubListBillsBySpace(t, []models4splitus.BillEntry{splitBill(), other}, nil)()

	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var response getSplitsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v — body: %s", err, w.Body.String())
	}
	if len(response.Splits) != 2 {
		t.Fatalf("expected 2 splits, got %d: %+v", len(response.Splits), response.Splits)
	}
	first := response.Splits[0]
	if first.ID != splitBillID || first.Title != "Dinner" || first.Amount != 6000 ||
		first.Currency != "EUR" || first.Status != models4splitus.BillStatusOutstanding || first.MembersCount != 3 {
		t.Errorf("splits[0] = %+v, want Dinner bill summary", first)
	}
	second := response.Splits[1]
	if second.ID != "bill2" || second.Title != "Taxi" || second.MembersCount != 2 {
		t.Errorf("splits[1] = %+v, want Taxi bill summary", second)
	}
}

func TestHandleGetSplits_EmptySpace(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubListBillsBySpace(t, nil, nil)()

	w := httptest.NewRecorder()
	handleGetSplits(context.Background(), w, getSplitsRequest("?spaceID=space1"), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var response getSplitsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(response.Splits) != 0 {
		t.Errorf("expected 0 splits, got %d", len(response.Splits))
	}
}

// --- routing ---

// TestInitApiForSplitus_ReadRoutes verifies the two read endpoints are
// registered (the create/legacy routes are asserted in api_bills_test.go).
func TestInitApiForSplitus_ReadRoutes(t *testing.T) {
	type routeKey struct{ method, path string }
	registered := make(map[routeKey]bool)
	InitApiForSplitus(func(method, path string, _ strongoapp.HttpHandlerWithContext) {
		registered[routeKey{method, path}] = true
	})
	for _, e := range []routeKey{
		{http.MethodGet, "/api4splitus/split"},
		{http.MethodGet, "/api4splitus/splits"},
	} {
		if !registered[e] {
			t.Errorf("expected route %s %s to be registered", e.method, e.path)
		}
	}
}

// --- production getBillTransfers seam ---

// TestGetBillTransfers_QueriesByBillID exercises the PRODUCTION seam body
// against the in-memory DB: only transfers whose TransferData.BillIDs
// contains the bill ID come back — the reverse linkage handleCreateSplit
// writes via CreateTransferRequest.BillID.
func TestGetBillTransfers_QueriesByBillID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	linked := splitDebtTransfer("t1", "cBea", 2000, 0)
	unrelated := splitDebtTransfer("t2", "cZoe", 500, 0)
	unrelated.Data.BillIDs = []string{"someOtherBill"}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		if err := tx.Set(ctx, linked.Record); err != nil {
			return err
		}
		return tx.Set(ctx, unrelated.Record)
	}); err != nil {
		t.Fatalf("failed to seed transfers: %v", err)
	}

	transfers, err := getBillTransfers(ctx, splitSpaceID, splitBillID)
	if err != nil {
		t.Fatalf("getBillTransfers() error: %v", err)
	}
	if len(transfers) != 1 {
		t.Fatalf("expected 1 transfer linked to %s, got %d", splitBillID, len(transfers))
	}
	if transfers[0].ID != "t1" {
		t.Errorf("transfers[0].ID = %s, want t1", transfers[0].ID)
	}
	if got := transfers[0].Data.BillIDs; len(got) != 1 || got[0] != splitBillID {
		t.Errorf("transfers[0].Data.BillIDs = %v, want [%s]", got, splitBillID)
	}
}

// TestHandleGetSplit_SkipsTransferWithoutContact covers the defensive branch
// for a linked transfer whose counterparty has no contact ID — it must not
// affect any participant's settled state.
func TestHandleGetSplit_SkipsTransferWithoutContact(t *testing.T) {
	defer stubSpaceMembership(t, true)()
	defer stubGetBillByID(t, splitBill(), nil)()
	noContact := splitDebtTransfer("t9", "", 2000, 2000)
	noContact.Data.ToJson = `{"userID":"someUser"}` // no contactID
	defer stubBillTransfers(t, []models4debtus.TransferEntry{
		noContact,
		splitDebtTransfer("t1", "cBea", 2000, 2000),
		splitDebtTransfer("t2", "cCat", 2000, 0),
	}, nil)()

	w := httptest.NewRecorder()
	handleGetSplit(context.Background(), w, getSplitRequest("?spaceID=space1&id=bill1"), token4auth.AuthInfo{UserID: splitPayerUserID})
	_, byName := decodeGetSplitResponse(t, w)
	if bea := byName["Bea"]; bea.Status != splitShareStatusSettled {
		t.Errorf("Bea status = %q, want %q", bea.Status, splitShareStatusSettled)
	}
	if cat := byName["Cat"]; cat.Status != splitShareStatusOutstanding {
		t.Errorf("Cat status = %q, want %q", cat.Status, splitShareStatusOutstanding)
	}
}
