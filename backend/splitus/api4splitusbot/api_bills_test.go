package api4splitusbot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
	"github.com/strongo/strongoapp"
)

func newBillEntry(id string, data *models4splitus.BillDbo) models4splitus.BillEntry {
	key := dal.NewKeyWithID("bills", id)
	return record.NewDataWithID(id, key, data)
}

// --- billToResponse ---

func TestBillToResponse(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		bill           models4splitus.BillEntry
		expectedStatus int
	}{
		{
			name:           "EmptyUserID",
			userID:         "",
			bill:           newBillEntry("bill1", &models4splitus.BillDbo{}),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "EmptyBillID",
			userID:         "user1",
			bill:           newBillEntry("", &models4splitus.BillDbo{}),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "NilBillData",
			userID: "user1",
			bill: models4splitus.BillEntry{
				RecordWithID: record.WithID[string]{ID: "bill1"},
				Data:         nil,
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Success",
			userID: "user1",
			bill: newBillEntry("bill1", &models4splitus.BillDbo{
				BillCommon: models4splitus.BillCommon{
					Name:        "Test Bill",
					Currency:    "USD",
					AmountTotal: decimal.Decimal64p2(1000),
					Members: []*briefs4splitus.BillMemberBrief{
						{
							MemberBrief: briefs4splitus.MemberBrief{UserID: "user1"},
							Owes:        decimal.Decimal64p2(500),
						},
					},
				},
			}),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx := context.Background()
			billToResponse(ctx, w, tt.userID, tt.bill)
			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// --- handleGetBill ---

func TestHandleGetBill_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/bill-get", nil)
	ctx := context.Background()
	handleGetBill(ctx, w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- handleCreateBill ---

func TestHandleCreateBill_Validation(t *testing.T) {
	tests := []struct {
		name           string
		formValues     url.Values
		expectedStatus int
	}{
		{
			name:           "InvalidSplit",
			formValues:     url.Values{"split": {"invalid"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "MissingSpaceID",
			formValues:     url.Values{"split": {"equally"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "MissingAmount",
			formValues:     url.Values{"split": {"equally"}, "spaceID": {"space1"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "InvalidAmount",
			formValues:     url.Values{"split": {"equally"}, "spaceID": {"space1"}, "amount": {"notanumber"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "InvalidMembersJSON",
			formValues:     url.Values{"split": {"equally"}, "spaceID": {"space1"}, "amount": {"10.00"}, "members": {"invalid-json"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NoMembers",
			formValues:     url.Values{"split": {"equally"}, "spaceID": {"space1"}, "amount": {"10.00"}, "members": {"[]"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "MemberMissingBothIDs",
			formValues:     url.Values{"split": {"equally"}, "spaceID": {"space1"}, "amount": {"10.00"}, "members": {`[{"amount":500}]`}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(tt.formValues.Encode())
			r := httptest.NewRequest(http.MethodPost, "/api4debtus/bill-create", body)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			handleCreateBill(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d, got %d — body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// --- handleGetBill seam tests ---

func TestHandleGetBill_GetBillError(t *testing.T) {
	orig := getBillByID
	getBillByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4splitus.BillEntry, error) {
		return models4splitus.BillEntry{}, errors.New("db error")
	}
	defer func() { getBillByID = orig }()

	r := httptest.NewRequest(http.MethodGet, "/api4debtus/bill-get?id=bill1", nil)
	w := httptest.NewRecorder()
	handleGetBill(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleGetBill_Success(t *testing.T) {
	orig := getBillByID
	getBillByID = func(_ context.Context, _ dal.ReadSession, billID string) (models4splitus.BillEntry, error) {
		return newBillEntry(billID, &models4splitus.BillDbo{
			BillCommon: models4splitus.BillCommon{Name: "Test", Currency: "USD"},
		}), nil
	}
	defer func() { getBillByID = orig }()

	r := httptest.NewRequest(http.MethodGet, "/api4debtus/bill-get?id=bill1", nil)
	w := httptest.NewRecorder()
	handleGetBill(context.Background(), w, r, token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

// --- handleCreateBill deeper paths ---

// makeCreateBillRequest builds an httptest.Request with the given form values.
func makeCreateBillRequest(values url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api4debtus/bill-create", strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// validMembersWithUserID returns form values for a single-member (userID only) bill.
func validMembersWithUserID() url.Values {
	return url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"userID":"user2","amount":500}]`},
	}
}

func TestHandleCreateBill_GetUsersByIDsError(t *testing.T) {
	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return nil, errors.New("users db error")
	}
	defer func() { getUsersByIDs = origUsers }()

	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(validMembersWithUserID()), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_MemberHasBothIDs(t *testing.T) {
	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return nil, nil
	}
	defer func() { getUsersByIDs = origUsers }()

	origGetContacts := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		return nil, nil
	}
	defer func() { getContactsByIDs = origGetContacts }()

	// Member has both userID and contactID — triggers the "both IDs" error branch.
	values := url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"userID":"user2","contactID":"c1","amount":500}]`},
	}
	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(values), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_GetContactsByIDsError(t *testing.T) {
	origContacts := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		return nil, errors.New("contacts db error")
	}
	defer func() { getContactsByIDs = origContacts }()

	values := url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"contactID":"c1","amount":500}]`},
	}
	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(values), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_ContactNotFound(t *testing.T) {
	origContacts := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		// Return a contact with a different ID than what the member references (c1 vs c2).
		return []dal4contactus.ContactEntry{
			dal4contactus.NewContactEntry("space1", "c2"),
		}, nil
	}
	defer func() { getContactsByIDs = origContacts }()

	values := url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"contactID":"c1","amount":500}]`},
	}
	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(values), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (contact not found), got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_TotalMismatch(t *testing.T) {
	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return []dbo4userus.UserEntry{}, nil
	}
	defer func() { getUsersByIDs = origUsers }()

	// amount=5.00 but member.amount=3.00 → mismatch
	values := url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"userID":"user2","amount":300}]`},
	}
	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(values), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (total mismatch), got %d — body: %s", w.Code, w.Body.String())
	}
}

// membersWithNamedUser returns form values where the single member has a userID
// that will be resolved to a named user, satisfying SetBillMembers name validation.
func membersWithNamedUser() url.Values {
	return url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"userID":"user2","amount":500}]`},
	}
}

// stubNamedUsers stubs getUsersByIDs to return a user with a full name.
func stubNamedUsers(t *testing.T) func() {
	t.Helper()
	orig := getUsersByIDs
	getUsersByIDs = func(_ context.Context, userIDs []string) ([]dbo4userus.UserEntry, error) {
		users := make([]dbo4userus.UserEntry, len(userIDs))
		for i, id := range userIDs {
			u := dbo4userus.NewUserEntry(id)
			// Names is a pointer field — must initialise before setting.
			u.Data.SetName("full", "Test User")
			users[i] = u
		}
		return users, nil
	}
	return func() { getUsersByIDs = orig }
}

func TestHandleCreateBill_ContactFoundSuccess(t *testing.T) {
	// Cover the "contact found" happy path: contact.ID matches member.ContactID.
	origContacts := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		contact := dal4contactus.NewContactEntry("space1", "c1")
		contact.Data.SetName("full", "Alice")
		return []dal4contactus.ContactEntry{contact}, nil
	}
	defer func() { getContactsByIDs = origContacts }()

	origCreate := createBill
	createBill = func(_ context.Context, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ *models4splitus.BillDbo) (models4splitus.BillEntry, error) {
		return newBillEntry("bill1", &models4splitus.BillDbo{
			BillCommon: models4splitus.BillCommon{Name: "Alice's Bill", Currency: "USD"},
		}), nil
	}
	defer func() { createBill = origCreate }()

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(ctx, nil)
	}
	defer func() { runReadwriteTransaction = origTx }()

	// member has contactID=c1 and amount matches total (5.00 = 500 in decimal64p2)
	values := url.Values{
		"split":   {"equally"},
		"spaceID": {"space1"},
		"amount":  {"5.00"},
		"members": {`[{"contactID":"c1","amount":500}]`},
	}
	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(values), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_SetBillMembersError(t *testing.T) {
	// member has no name (userID not in users list) → SetBillMembers returns error
	origUsers := getUsersByIDs
	getUsersByIDs = func(_ context.Context, _ []string) ([]dbo4userus.UserEntry, error) {
		return []dbo4userus.UserEntry{}, nil // empty — member gets no name
	}
	defer func() { getUsersByIDs = origUsers }()

	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(validMembersWithUserID()), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 (SetBillMembers error), got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_CreateBillError(t *testing.T) {
	defer stubNamedUsers(t)()

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
	handleCreateBill(context.Background(), w, makeCreateBillRequest(membersWithNamedUser()), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateBill_Success(t *testing.T) {
	defer stubNamedUsers(t)()

	origCreate := createBill
	createBill = func(_ context.Context, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ *models4splitus.BillDbo) (models4splitus.BillEntry, error) {
		return newBillEntry("bill1", &models4splitus.BillDbo{
			BillCommon: models4splitus.BillCommon{Name: "New Bill", Currency: "USD"},
		}), nil
	}
	defer func() { createBill = origCreate }()

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(ctx, nil)
	}
	defer func() { runReadwriteTransaction = origTx }()

	w := httptest.NewRecorder()
	handleCreateBill(context.Background(), w, makeCreateBillRequest(membersWithNamedUser()), token4auth.AuthInfo{UserID: "user1"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

// --- InitApiForSplitus ---

func TestInitApiForSplitus(t *testing.T) {
	type routeKey struct{ method, path string }
	registered := make(map[routeKey]bool)
	handle := strongoapp.HandleHttpWithContext(func(method, path string, handler strongoapp.HttpHandlerWithContext) {
		registered[routeKey{method, path}] = true
	})
	InitApiForSplitus(handle)

	expected := []routeKey{
		{http.MethodPost, "/api4debtus/bill-create"},
		{http.MethodGet, "/api4debtus/bill-get"},
	}
	for _, e := range expected {
		if !registered[e] {
			t.Errorf("expected route %s %s to be registered", e.method, e.path)
		}
	}
}
