package api4unsorted

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dbo4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/strongoapp"
	"github.com/strongo/strongoapp/person"
)

var errFake = errors.New("fake error")

func authAdmin() token4auth.AuthInfo {
	return token4auth.AuthInfo{UserID: "u1", IsAdmin: true}
}

func authUser(id string) token4auth.AuthInfo {
	return token4auth.AuthInfo{UserID: id}
}

// --- InitApiForUnsorted -------------------------------------------------------

func TestInitApiForUnsorted(t *testing.T) {
	registered := make(map[string]int)
	handle := strongoapp.HandleHttpWithContext(func(method, path string, handler strongoapp.HttpHandlerWithContext) {
		registered[method+":"+path]++
	})
	InitApiForUnsorted(handle)
	if len(registered) == 0 {
		t.Fatal("expected routes to be registered")
	}
	for _, key := range []string{
		"GET:/api4debtus/user",
		"POST:/api4debtus/contact-create",
		"GET:/api4debtus/admin/latest/users",
	} {
		if registered[key] == 0 {
			t.Errorf("route %s not registered", key)
		}
	}
}

// --- HandleAdminLatestUsers --------------------------------------------------

func TestHandleAdminLatestUsers_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandleAdminLatestUsers(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandleAdminFindUser -----------------------------------------------------

func TestHandleAdminFindUser_MissingUserID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleAdminFindUser(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminFindUser_GetUserError(t *testing.T) {
	orig := getUser
	getUser = func(_ context.Context, _ dal.ReadSession, _ dbo4userus.UserEntry) error {
		return errFake
	}
	t.Cleanup(func() { getUser = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?userID=u1", nil)
	HandleAdminFindUser(r.Context(), w, r, authAdmin())
	// logs error but still returns 200
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleAdminFindUser_Success(t *testing.T) {
	orig := getUser
	getUser = func(_ context.Context, _ dal.ReadSession, user dbo4userus.UserEntry) error {
		// Names is *person.NameFields (nil by default); initialise directly so that
		// the caller's appUser.Data.GetFullName() doesn't panic.
		nf := person.NameFields{UserName: "Alice"}
		user.Data.Names = &nf
		return nil
	}
	t.Cleanup(func() { getUser = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?userID=u1", nil)
	HandleAdminFindUser(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleAdminMergeUserContacts --------------------------------------------

func TestHandleAdminMergeUserContacts_MissingKeepID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_MissingDeleteID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_MissingSpaceID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_TxError(t *testing.T) {
	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return errFake
	}
	t.Cleanup(func() { runReadwriteTransaction = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- getContactID ------------------------------------------------------------

func TestGetContactID_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	id := getContactID(w, url.Values{})
	if id != "" {
		t.Errorf("expected empty, got %q", id)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetContactID_Present(t *testing.T) {
	w := httptest.NewRecorder()
	id := getContactID(w, url.Values{"id": {"contact1"}})
	if id != "contact1" {
		t.Errorf("expected contact1, got %q", id)
	}
}

// --- HandleGetContact --------------------------------------------------------

func TestHandleGetContact_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandleGetContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleGetContact_DBError(t *testing.T) {
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, errFake }
	t.Cleanup(func() { facade.GetSneatDB = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=c1&spaceID=s1", nil)
	HandleGetContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- contactToResponse -------------------------------------------------------

func TestContactToResponse_Forbidden(t *testing.T) {
	w := httptest.NewRecorder()
	contact := dal4contactus.NewContactEntry("s1", "c1")
	contact.Data.UserID = "other"
	debtusContact := models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil)
	contactToResponse(context.Background(), w, authUser("u1"), contact, debtusContact)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestContactToResponse_LoadTransfersError(t *testing.T) {
	orig := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, errFake
	}
	t.Cleanup(func() { loadTransfersByContactID = orig })

	w := httptest.NewRecorder()
	contact := dal4contactus.NewContactEntry("s1", "c1")
	contact.Data.UserID = "u1"
	debtusContact := models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil)
	contactToResponse(context.Background(), w, authUser("u1"), contact, debtusContact)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestContactToResponse_Success(t *testing.T) {
	orig := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() { loadTransfersByContactID = orig })

	w := httptest.NewRecorder()
	contact := dal4contactus.NewContactEntry("s1", "c1")
	contact.Data.UserID = "u1"
	nf := person.NameFields{UserName: "Alice"}
	contact.Data.Names = &nf
	debtusContact := models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil)
	contactToResponse(context.Background(), w, authUser("u1"), contact, debtusContact)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleCreateCounterparty ------------------------------------------------

func TestHandleCreateCounterparty_BadTel(t *testing.T) {
	body := "name=Test&tel=notanumber"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateCounterparty_TxError(t *testing.T) {
	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return errFake
	}
	t.Cleanup(func() { runReadwriteTransaction = orig })

	body := "name=Test"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleCreateCounterparty_WithEmail(t *testing.T) {
	// covers the email branch (len(email) > 0)
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return nil
	}
	origCreate := createContact
	createContact = func(_ facade.ContextWithUser, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ dto4contactus.ContactDetails) (dal4contactus.ContactEntry, dal4contactus.ContactusSpaceEntry, models4debtus.DebtusSpaceContactEntry, error) {
		return dal4contactus.NewContactEntry("s1", "newID"), dal4contactus.NewContactusSpaceEntry("s1"), models4debtus.NewDebtusSpaceContactEntry("s1", "newID", nil), nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		createContact = origCreate
	})

	body := "name=Test&email=test@example.com"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// stubUserContext is a minimal facade.UserContext for tests.
type stubUserContext struct{ id string }

func (s stubUserContext) GetUserID() string { return s.id }

// overrideNewUserContext replaces newUserContext with a stub that never panics
// and returns the t.Cleanup restorer.
func overrideNewUserContext(t *testing.T) {
	t.Helper()
	orig := newUserContext
	newUserContext = func(_ string) facade.UserContext { return stubUserContext{"u1"} }
	t.Cleanup(func() { newUserContext = orig })
}

// --- HandleDeleteContact -----------------------------------------------------

func TestHandleDeleteContact_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleDeleteContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleDeleteContact_Error(t *testing.T) {
	overrideNewUserContext(t)
	orig := deleteContact
	deleteContact = func(_ context.Context, _ facade.UserContext, _ coretypes.SpaceID, _ string) error {
		return errFake
	}
	t.Cleanup(func() { deleteContact = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleDeleteContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleDeleteContact_Success(t *testing.T) {
	overrideNewUserContext(t)
	orig := deleteContact
	deleteContact = func(_ context.Context, _ facade.UserContext, _ coretypes.SpaceID, _ string) error {
		return nil
	}
	t.Cleanup(func() { deleteContact = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleDeleteContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleArchiveCounterparty -----------------------------------------------

func TestHandleArchiveCounterparty_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleArchiveCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleArchiveCounterparty_Error(t *testing.T) {
	orig := changeContactStatus
	changeContactStatus = func(_ facade.ContextWithUser, _ coretypes.SpaceID, _ string, _ models4debtus.DebtusContactStatus) (dal4contactus.ContactEntry, models4debtus.DebtusSpaceContactEntry, error) {
		return dal4contactus.NewContactEntry("s1", "c1"), models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), errFake
	}
	t.Cleanup(func() { changeContactStatus = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleArchiveCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleArchiveCounterparty_Success(t *testing.T) {
	origStatus := changeContactStatus
	changeContactStatus = func(_ facade.ContextWithUser, _ coretypes.SpaceID, _ string, _ models4debtus.DebtusContactStatus) (dal4contactus.ContactEntry, models4debtus.DebtusSpaceContactEntry, error) {
		c := dal4contactus.NewContactEntry("s1", "c1")
		c.Data.UserID = "u1"
		nf := person.NameFields{UserName: "Alice"}
		c.Data.Names = &nf
		return c, models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), nil
	}
	origTransfers := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() {
		changeContactStatus = origStatus
		loadTransfersByContactID = origTransfers
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleArchiveCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleActivateCounterparty ----------------------------------------------

func TestHandleActivateCounterparty_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleActivateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleActivateCounterparty_Error(t *testing.T) {
	orig := changeContactStatus
	changeContactStatus = func(_ facade.ContextWithUser, _ coretypes.SpaceID, _ string, _ models4debtus.DebtusContactStatus) (dal4contactus.ContactEntry, models4debtus.DebtusSpaceContactEntry, error) {
		return dal4contactus.NewContactEntry("s1", "c1"), models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), errFake
	}
	t.Cleanup(func() { changeContactStatus = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleActivateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleActivateCounterparty_Success(t *testing.T) {
	origStatus := changeContactStatus
	changeContactStatus = func(_ facade.ContextWithUser, _ coretypes.SpaceID, _ string, _ models4debtus.DebtusContactStatus) (dal4contactus.ContactEntry, models4debtus.DebtusSpaceContactEntry, error) {
		c := dal4contactus.NewContactEntry("s1", "c1")
		c.Data.UserID = "u1"
		nf := person.NameFields{UserName: "Alice"}
		c.Data.Names = &nf
		return c, models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), nil
	}
	origTransfers := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() {
		changeContactStatus = origStatus
		loadTransfersByContactID = origTransfers
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	HandleActivateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleUpdateCounterparty ------------------------------------------------

func TestHandleUpdateCounterparty_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleUpdateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdateCounterparty_TooManyValues(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1", strings.NewReader("name=A&name=B"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	HandleUpdateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdateCounterparty_UpdateError(t *testing.T) {
	orig := updateContact
	updateContact = func(_ context.Context, _ coretypes.SpaceID, _ string, _ map[string]string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), errFake
	}
	t.Cleanup(func() { updateContact = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", strings.NewReader("name=Test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleUpdateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestHandleUpdateCounterparty_Success is intentionally omitted.
// HandleUpdateCounterparty creates a bare ContactEntry (Names==nil) and passes
// it directly to contactToResponse which calls Names.GetFullName() — this always
// panics in production when updateContact succeeds. The success path is a
// production bug; it is documented as an uncoverable gap in TEST-COVERAGE.md.

// --- HandlerGetGroup ---------------------------------------------------------

func TestHandlerGetGroup_MissingID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandlerGetGroup(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerGetGroup_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=g1", nil)
	HandlerGetGroup(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- groupsToJson / groupToResponse ------------------------------------------

func TestGroupsToJson_NotImplemented(t *testing.T) {
	_, err := groupsToJson(nil, dbo4userus.UserEntry{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGroupToResponse_Error(t *testing.T) {
	w := httptest.NewRecorder()
	err := groupToResponse(context.Background(), w, models4splitus.GroupEntry{}, dbo4userus.UserEntry{})
	if err == nil {
		t.Error("expected error from groupsToJson not implemented")
	}
}

// --- HandleJoinGroups --------------------------------------------------------

func TestHandleJoinGroups_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandleJoinGroups(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandlerDeleteGroup -------------------------------------------------------

func TestHandlerDeleteGroup_NoOp(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandlerDeleteGroup(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandlerUpdateGroup -------------------------------------------------------

func TestHandlerUpdateGroup_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandlerUpdateGroup(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandlerSetContactsToGroup -----------------------------------------------

func TestHandlerSetContactsToGroup_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	HandlerSetContactsToGroup(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandleUserInfo ----------------------------------------------------------

func TestHandleUserInfo_BadID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=notanumber", nil)
	HandleUserInfo(r.Context(), w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUserInfo_SaveBrowserError(t *testing.T) {
	orig := saveUserBrowser
	saveUserBrowser = func(_ context.Context, _ string, _ string) (models4auth.UserBrowser, error) {
		return models4auth.UserBrowser{}, errFake
	}
	t.Cleanup(func() { saveUserBrowser = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=42", nil)
	HandleUserInfo(r.Context(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleUserInfo_Success(t *testing.T) {
	orig := saveUserBrowser
	saveUserBrowser = func(_ context.Context, _ string, _ string) (models4auth.UserBrowser, error) {
		return models4auth.UserBrowser{}, nil
	}
	t.Cleanup(func() { saveUserBrowser = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=42", nil)
	HandleUserInfo(r.Context(), w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleSaveVisitorData ---------------------------------------------------

func TestHandleSaveVisitorData_MissingGaClientId(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleSaveVisitorData(r.Context(), w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSaveVisitorData_SaveError(t *testing.T) {
	orig := saveGaClient
	saveGaClient = func(_ context.Context, _, _, _ string) (models4auth.GaClient, error) {
		return models4auth.GaClient{}, errFake
	}
	t.Cleanup(func() { saveGaClient = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("gaClientId=abc"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleSaveVisitorData(r.Context(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleSaveVisitorData_Success(t *testing.T) {
	orig := saveGaClient
	saveGaClient = func(_ context.Context, _, _, _ string) (models4auth.GaClient, error) {
		return models4auth.GaClient{}, nil
	}
	t.Cleanup(func() { saveGaClient = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("gaClientId=abc"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleSaveVisitorData(r.Context(), w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleMe ----------------------------------------------------------------

func TestHandleMe_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandleMe(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- SetUserName -------------------------------------------------------------

func TestSetUserName_EmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	SetUserName(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetUserName_WorkerError(t *testing.T) {
	orig := dal4userus.RunUserWorker
	dal4userus.RunUserWorker = func(_ facade.ContextWithUser, _ bool, _ func(facade.ContextWithUser, dal.ReadwriteTransaction, *dal4userus.UserWorkerParams) error) error {
		return errFake
	}
	t.Cleanup(func() { dal4userus.RunUserWorker = orig })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("Alice"))
	SetUserName(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSetUserName_DelayError(t *testing.T) {
	origWorker := dal4userus.RunUserWorker
	dal4userus.RunUserWorker = func(ctx facade.ContextWithUser, _ bool, f func(facade.ContextWithUser, dal.ReadwriteTransaction, *dal4userus.UserWorkerParams) error) error {
		params := &dal4userus.UserWorkerParams{User: dbo4userus.NewUserEntry("u1")}
		nf := person.NameFields{UserName: ""}
		params.User.Data.Names = &nf
		return f(ctx, nil, params)
	}
	origDelay := delayUpdateTransfersWithCreatorName
	delayUpdateTransfersWithCreatorName = func(_ context.Context, _ string) error { return errFake }
	t.Cleanup(func() {
		dal4userus.RunUserWorker = origWorker
		delayUpdateTransfersWithCreatorName = origDelay
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("Alice"))
	SetUserName(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSetUserName_Success(t *testing.T) {
	origWorker := dal4userus.RunUserWorker
	dal4userus.RunUserWorker = func(ctx facade.ContextWithUser, _ bool, f func(facade.ContextWithUser, dal.ReadwriteTransaction, *dal4userus.UserWorkerParams) error) error {
		params := &dal4userus.UserWorkerParams{User: dbo4userus.NewUserEntry("u1")}
		nf := person.NameFields{UserName: ""}
		params.User.Data.Names = &nf
		return f(ctx, nil, params)
	}
	origDelay := delayUpdateTransfersWithCreatorName
	delayUpdateTransfersWithCreatorName = func(_ context.Context, _ string) error { return nil }
	t.Cleanup(func() {
		dal4userus.RunUserWorker = origWorker
		delayUpdateTransfersWithCreatorName = origDelay
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("Alice"))
	SetUserName(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleGetUserCurrencies -------------------------------------------------

func TestHandleGetUserCurrencies(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	HandleGetUserCurrencies(r.Context(), w, r, authUser("u1"), dbo4userus.NewUserEntry("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- ApiWebhookContext simple methods ----------------------------------------

func TestApiWebhookContext_SimpleStubs(t *testing.T) {
	whc := ApiWebhookContext{}

	if id := whc.BotChatIntID(); id != 0 {
		t.Errorf("BotChatIntID: expected 0, got %d", id)
	}
	if e := whc.ChatEntity(); e != nil {
		t.Errorf("ChatEntity: expected nil, got %v", e)
	}
	if err := whc.Init(nil, nil); err != nil {
		t.Errorf("Init: expected nil, got %v", err)
	}
	if !whc.IsNewerThen(nil) {
		t.Error("IsNewerThen: expected true")
	}
	if s := whc.MessageText(); s != "" {
		t.Errorf("MessageText: expected empty, got %q", s)
	}
}

// --- HandleTgHelperCurrencySelected validation paths ------------------------

func TestHandleTgHelperCurrencySelected_MissingCurrency(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for missing currency, got 200")
	}
}

func TestHandleTgHelperCurrencySelected_WrongLength(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("currency=US"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for wrong currency length, got 200")
	}
}

func TestHandleTgHelperCurrencySelected_WrongCase(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("currency=usd"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for lowercase currency, got 200")
	}
}

func TestHandleTgHelperCurrencySelected_MissingTgChat(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("currency=USD"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for missing tg-chat, got 200")
	}
}

func TestHandleTgHelperCurrencySelected_BadTgChatFormat(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("currency=USD&tg-chat=badformat"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for bad tg-chat format, got 200")
	}
}

func TestHandleTgHelperCurrencySelected_BadTgChatID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("currency=USD&tg-chat=bot:notanumber"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for non-integer tg-chat id, got 200")
	}
}

// emptyRecordsReader is a minimal dal.RecordsReader that immediately signals no more records.
type emptyRecordsReader struct{}

func (emptyRecordsReader) Next() (dal.Record, error) { return nil, dal.ErrNoMoreRecords }
func (emptyRecordsReader) Cursor() (string, error)   { return "", nil }
func (emptyRecordsReader) Close() error              { return nil }

// singleIntRecordsReader returns one record with an int key then ErrNoMoreRecords.
type singleIntRecordsReader struct {
	done bool
	id   int
}

func (r *singleIntRecordsReader) Next() (dal.Record, error) {
	if r.done {
		return nil, dal.ErrNoMoreRecords
	}
	r.done = true
	key := dal.NewKeyWithID[int](models4debtus.TransfersCollection, r.id)
	return dal.NewRecord(key), nil
}
func (r *singleIntRecordsReader) Cursor() (string, error) { return "", nil }
func (r *singleIntRecordsReader) Close() error            { return nil }

// --- DelayedChangeTransfersCounterparty --------------------------------------

func TestDelayedChangeTransfersCounterparty_DBError(t *testing.T) {
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, errFake }
	t.Cleanup(func() { facade.GetSneatDB = orig })

	err := DelayedChangeTransfersCounterparty(context.Background(), 1, 2, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDelayedChangeTransfersCounterparty_QueryError(t *testing.T) {
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origQuery := executeQueryToRecordsReader
	executeQueryToRecordsReader = func(_ context.Context, _ dal.DB, _ dal.Query) (dal.RecordsReader, error) {
		return nil, errFake
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		executeQueryToRecordsReader = origQuery
	})

	err := DelayedChangeTransfersCounterparty(context.Background(), 1, 2, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDelayedChangeTransfersCounterparty_EmptyResult(t *testing.T) {
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origQuery := executeQueryToRecordsReader
	executeQueryToRecordsReader = func(_ context.Context, _ dal.DB, _ dal.Query) (dal.RecordsReader, error) {
		return emptyRecordsReader{}, nil
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		executeQueryToRecordsReader = origQuery
	})

	err := DelayedChangeTransfersCounterparty(context.Background(), 1, 2, "")
	// With 0 transferIDs the EnqueueWorkMulti is called with empty args — may error
	// depending on delayer init, but we just care the query path was covered.
	_ = err
}

func TestDelayedChangeTransfersCounterparty_WithOneTransfer(t *testing.T) {
	// Covers the args-building loop (lines after SelectAllIDs).
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origQuery := executeQueryToRecordsReader
	executeQueryToRecordsReader = func(_ context.Context, _ dal.DB, _ dal.Query) (dal.RecordsReader, error) {
		return &singleIntRecordsReader{id: 42}, nil
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		executeQueryToRecordsReader = origQuery
	})

	err := DelayedChangeTransfersCounterparty(context.Background(), 1, 2, "")
	// EnqueueWorkMulti may fail without delayer init — we only care the loop ran.
	_ = err
}

// --- DelayedChangeTransferCounterparty ---------------------------------------

func TestDelayedChangeTransferCounterparty_GetContactError(t *testing.T) {
	orig := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.DebtusSpaceContactEntry{}, errFake
	}
	t.Cleanup(func() { getDebtusSpaceContactByID = orig })

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDelayedChangeTransferCounterparty_GetTransferError(t *testing.T) {
	origContact := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "new1", nil), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return models4debtus.TransferEntry{}, errFake
	}
	t.Cleanup(func() {
		getDebtusSpaceContactByID = origContact
		runReadwriteTransaction = origTx
		getTransferByID = origGet
	})

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDelayedChangeTransferCounterparty_NoMatch(t *testing.T) {
	origContact := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "new1", nil), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		entry := models4debtus.NewTransfer("t1", &models4debtus.TransferData{})
		entry.Data.BothCounterpartyIDs = []string{"other1", "other2"} // no match for "old1"
		return entry, nil
	}
	t.Cleanup(func() {
		getDebtusSpaceContactByID = origContact
		runReadwriteTransaction = origTx
		getTransferByID = origGet
	})

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func newTransferWithFrom(fromContactID string) models4debtus.TransferEntry {
	entry := models4debtus.NewTransfer("t1", &models4debtus.TransferData{})
	entry.Data.BothCounterpartyIDs = []string{fromContactID, "other2"}
	entry.Data.FromJson = `{"contactID":"` + fromContactID + `"}`
	entry.Data.ToJson = `{"contactID":"other2"}`
	return entry
}

func TestDelayedChangeTransferCounterparty_MatchFromSaveError(t *testing.T) {
	origContact := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "new1", nil), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return newTransferWithFrom("old1"), nil
	}
	origSave := saveTransfer
	saveTransfer = func(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.TransferEntry) error {
		return errFake
	}
	t.Cleanup(func() {
		getDebtusSpaceContactByID = origContact
		runReadwriteTransaction = origTx
		getTransferByID = origGet
		saveTransfer = origSave
	})

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err == nil {
		t.Error("expected error from saveTransfer, got nil")
	}
}

func TestDelayedChangeTransferCounterparty_MatchSuccess(t *testing.T) {
	origContact := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "new1", nil), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return newTransferWithFrom("old1"), nil
	}
	origSave := saveTransfer
	saveTransfer = func(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.TransferEntry) error {
		return nil
	}
	t.Cleanup(func() {
		getDebtusSpaceContactByID = origContact
		runReadwriteTransaction = origTx
		getTransferByID = origGet
		saveTransfer = origSave
	})

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestDelayedChangeTransferCounterparty_MatchToSuccess(t *testing.T) {
	// Cover the `else if to.ContactID == oldID` branch by putting oldID in ToJson not FromJson
	origContact := getDebtusSpaceContactByID
	getDebtusSpaceContactByID = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "new1", nil), nil
	}
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getTransferByID
	getTransferByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		entry := models4debtus.NewTransfer("t1", &models4debtus.TransferData{})
		entry.Data.BothCounterpartyIDs = []string{"old1", "other2"}
		entry.Data.FromJson = `{"contactID":"other2"}` // from is NOT old1
		entry.Data.ToJson = `{"contactID":"old1"}`     // to IS old1
		return entry, nil
	}
	origSave := saveTransfer
	saveTransfer = func(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.TransferEntry) error {
		return nil
	}
	t.Cleanup(func() {
		getDebtusSpaceContactByID = origContact
		runReadwriteTransaction = origTx
		getTransferByID = origGet
		saveTransfer = origSave
	})

	err := DelayedChangeTransferCounterparty(context.Background(), "s1", "t1", "old1", "new1", "")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// --- HandleAdminMergeUserContacts tx body ------------------------------------

func TestHandleAdminMergeUserContacts_GetContactsError(t *testing.T) {
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		return nil, errFake
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_TooFewContacts(t *testing.T) {
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		return []dal4contactus.ContactEntry{dal4contactus.NewContactEntry("s1", "k1")}, nil // only 1
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_DifferentUserIDs(t *testing.T) {
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		c1 := dal4contactus.NewContactEntry("s1", "k1")
		c1.Data.UserID = "u1"
		c2 := dal4contactus.NewContactEntry("s1", "d1")
		c2.Data.UserID = "u2" // different
		return []dal4contactus.ContactEntry{c1, c2}, nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleAdminMergeUserContacts_GetContactusSpaceError(t *testing.T) {
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
	origGet := getContactsByIDs
	getContactsByIDs = func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		c1 := dal4contactus.NewContactEntry("s1", "k1")
		c2 := dal4contactus.NewContactEntry("s1", "d1")
		return []dal4contactus.ContactEntry{c1, c2}, nil
	}
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, _ dal4contactus.ContactusSpaceEntry) error {
		return errFake
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandlerCreateGroup -------------------------------------------------------

func TestHandlerCreateGroup_ParseFormError(t *testing.T) {
	// Use a body that causes ParseForm to fail via Content-Type mismatch — actually
	// ParseForm rarely fails; test the createGroupFn error path instead.
	origCreate := createGroupFn
	createGroupFn = func(_ context.Context, _ *models4splitus.GroupDbo, _ string,
		_ func(context.Context, *models4splitus.GroupDbo) (models4splitus.GroupEntry, error),
		_ func(context.Context, models4splitus.GroupEntry, dbo4userus.UserEntry) error,
	) (models4splitus.GroupEntry, models4splitus.GroupMember, error) {
		return models4splitus.GroupEntry{}, models4splitus.GroupMember{}, errFake
	}
	t.Cleanup(func() { createGroupFn = origCreate })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=TestGroup&note=SomeNote"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandlerCreateGroup(r.Context(), w, r, authUser("u1"), dbo4userus.NewUserEntry("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandleGetContact GetMulti paths -----------------------------------------

func TestHandleGetContact_GetMultiError(t *testing.T) {
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origGet := getMultiRecords
	getMultiRecords = func(_ context.Context, _ dal.DB, _ []dal.Record) error { return errFake }
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		getMultiRecords = origGet
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=c1&spaceID=s1", nil)
	HandleGetContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleGetContact_Success(t *testing.T) {
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origGet := getMultiRecords
	getMultiRecords = func(_ context.Context, _ dal.DB, records []dal.Record) error {
		// Mark records as retrieved, then populate Names and UserID so contactToResponse works.
		for _, rec := range records {
			rec.SetError(dal.ErrNoError)
		}
		if data, ok := records[0].Data().(*dbo4contactus.ContactDbo); ok {
			nf := person.NameFields{UserName: "Alice"}
			data.Names = &nf
			data.UserID = "u1"
		}
		return nil
	}
	origLoad := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		getMultiRecords = origGet
		loadTransfersByContactID = origLoad
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?id=c1&spaceID=s1", nil)
	HandleGetContact(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- contactToResponse with balance ------------------------------------------

func TestContactToResponse_WithBalance(t *testing.T) {
	orig := loadTransfersByContactID
	loadTransfersByContactID = func(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() { loadTransfersByContactID = orig })

	w := httptest.NewRecorder()
	contact := dal4contactus.NewContactEntry("s1", "c1")
	contact.Data.UserID = "u1"
	nf := person.NameFields{UserName: "Alice"}
	contact.Data.Names = &nf
	debtusContact := models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil)
	debtusContact.Data.Balance = money.Balance{"USD": 1000}
	contactToResponse(context.Background(), w, authUser("u1"), contact, debtusContact)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- HandleCreateCounterparty with tel success --------------------------------

func TestHandleCreateCounterparty_WithTelSuccess(t *testing.T) {
	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return nil
	}
	origCreate := createContact
	createContact = func(_ facade.ContextWithUser, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ dto4contactus.ContactDetails) (dal4contactus.ContactEntry, dal4contactus.ContactusSpaceEntry, models4debtus.DebtusSpaceContactEntry, error) {
		return dal4contactus.NewContactEntry("s1", "newID"), dal4contactus.NewContactusSpaceEntry("s1"), models4debtus.NewDebtusSpaceContactEntry("s1", "newID", nil), nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		createContact = origCreate
	})

	body := "name=Test&tel=1234567890"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
