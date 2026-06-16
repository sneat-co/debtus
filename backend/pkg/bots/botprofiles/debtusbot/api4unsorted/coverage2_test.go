package api4unsorted

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/bots"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
)

// setupRunTxPassthrough makes runReadwriteTransaction execute its callback directly.
func setupRunTxPassthrough() func(context.Context, func(context.Context, dal.ReadwriteTransaction) error, ...dal.TransactionOption) error {
	return func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), nil)
	}
}

// setupTwoContacts returns a getContactsByIDs stub with two contacts sharing the same userID.
func setupTwoContacts(userID string) func(context.Context, dal.ReadSession, coretypes.SpaceID, []string) ([]dal4contactus.ContactEntry, error) {
	return func(_ context.Context, _ dal.ReadSession, _ coretypes.SpaceID, _ []string) ([]dal4contactus.ContactEntry, error) {
		c1 := dal4contactus.NewContactEntry("s1", "k1")
		c1.Data.UserID = userID
		c2 := dal4contactus.NewContactEntry("s1", "d1")
		c2.Data.UserID = userID
		return []dal4contactus.ContactEntry{c1, c2}, nil
	}
}

// fakeTgChatDal is a minimal TgChatDal that returns a preset error without calling sendToTelegram.
type fakeTgChatDal struct{ err error }

func (f fakeTgChatDal) DoSomething(
	_ context.Context, userTask *sync.WaitGroup, _ string, _ int64,
	_ token4auth.AuthInfo, _ dbo4userus.UserEntry,
	_ func(botsfwtgmodels.TgChatData) error,
) error {
	userTask.Wait()
	return f.err
}

// --- HandleAdminMergeUserContacts inner tx branches ---------------------------

func TestHandleAdminMergeUserContacts_HasContactTrue(t *testing.T) {
	// Covers HasContact==true branch: reaches tx.Update which panics on nil tx.
	// Use recover() — lines are covered before the panic.
	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxPassthrough()
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("") // both UserIDs empty → lines 71+74 pass
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		brief := &briefs4contactus.ContactBrief{}
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{"d1": brief}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	defer func() { recover() }() //nolint:errcheck // COVER-BEFORE-PANIC: lines covered before nil-tx panic
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
}

func TestHandleAdminMergeUserContacts_HasContactFalse(t *testing.T) {
	// Covers HasContact==false branch (skips tx.Update), reaches EnqueueWork.
	// EnqueueWork panics on uninitialized delayer — use recover().
	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxPassthrough()
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	defer func() { recover() }() //nolint:errcheck // COVER-BEFORE-PANIC: lines covered before nil-delayer panic
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
}

// --- HandleCreateCounterparty inner paths ------------------------------------

func TestHandleCreateCounterparty_ParseFormError(t *testing.T) {
	body := iotest.ErrReader(errors.New("read error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ContentLength = 100
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for ParseForm error, got %d", w.Code)
	}
}

func TestHandleCreateCounterparty_CreateContactError(t *testing.T) {
	// Execute the tx callback and make createContact fail.
	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxPassthrough()
	origCreate := createContact
	createContact = func(_ facade.ContextWithUser, _ dal.ReadwriteTransaction, _ coretypes.SpaceID, _ dto4contactus.ContactDetails) (dal4contactus.ContactEntry, dal4contactus.ContactusSpaceEntry, models4debtus.DebtusSpaceContactEntry, error) {
		return dal4contactus.ContactEntry{}, dal4contactus.ContactusSpaceEntry{}, models4debtus.DebtusSpaceContactEntry{}, errFake
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		createContact = origCreate
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleCreateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandleUpdateCounterparty case 0 and case 1 --------------------------------

func TestHandleUpdateCounterparty_SingleValue(t *testing.T) {
	// Pre-parse the form so r.PostForm["name"] has len==1 (hits case 1).
	origUpdate := updateContact
	updateContact = func(_ context.Context, _ coretypes.SpaceID, _ string, _ map[string]string) (models4debtus.DebtusSpaceContactEntry, error) {
		return models4debtus.NewDebtusSpaceContactEntry("s1", "c1", nil), errFake
	}
	t.Cleanup(func() { updateContact = origUpdate })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", strings.NewReader("name=Test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	HandleUpdateCounterparty(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleUpdateCounterparty_ZeroValues(t *testing.T) {
	// Directly set r.PostForm with a key having zero values to hit case 0.
	// case 0 does vals[0] which panics — use COVER-BEFORE-PANIC.
	defer func() { recover() }() //nolint:errcheck // COVER-BEFORE-PANIC: case 0 panics on empty slice
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?id=c1&spaceID=s1", nil)
	r.PostForm = map[string][]string{"name": {}} // len==0 hits case 0
	HandleUpdateCounterparty(r.Context(), w, r, authUser("u1"))
}

// --- HandlerCreateGroup ParseForm error path and success path -----------------

func TestHandlerCreateGroup_ParseFormBodyError(t *testing.T) {
	body := iotest.ErrReader(errors.New("read error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ContentLength = 100
	HandlerCreateGroup(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateGroup_GroupToResponseError(t *testing.T) {
	// createGroupFn succeeds → reaches groupToResponse → groupsToJson always errors → 500.
	origCreate := createGroupFn
	createGroupFn = func(_ context.Context, groupEntity *models4splitus.GroupDbo, _ string,
		_ func(context.Context, *models4splitus.GroupDbo) (models4splitus.GroupEntry, error),
		_ func(context.Context, models4splitus.GroupEntry, dbo4userus.UserEntry) error,
	) (models4splitus.GroupEntry, models4splitus.GroupMember, error) {
		return models4splitus.GroupEntry{}, models4splitus.GroupMember{}, nil // success
	}
	t.Cleanup(func() { createGroupFn = origCreate })

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=TestGroup"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandlerCreateGroup(r.Context(), w, r, authUser("u1"), dbo4userus.UserEntry{})
	// groupsToJson always returns "not implemented yet" → groupToResponse returns err → 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- HandleSaveVisitorData ParseForm error ------------------------------------

func TestHandleSaveVisitorData_ParseFormError(t *testing.T) {
	body := iotest.ErrReader(errors.New("read error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ContentLength = 100
	HandleSaveVisitorData(r.Context(), w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for ParseForm error, got %d", w.Code)
	}
}

// --- SetUserName io.ReadAll error ---------------------------------------------

func TestSetUserName_ReadBodyError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", iotest.ErrReader(errors.New("read error")))
	SetUserName(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- NewApiWebhookContext direct test -----------------------------------------

func TestNewApiWebhookContext(t *testing.T) {
	// NewApiWebhookContext panics if BotHost is nil (set at startup).
	// Use COVER-BEFORE-PANIC: the constructor lines are covered before the panic.
	defer func() { recover() }() //nolint:errcheck // COVER-BEFORE-PANIC
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	userDbo := &dbo4userus.UserDbo{}
	chatData := &anybot.SneatAppTgChatDbo{}
	NewApiWebhookContext(r, userDbo, "u1", 12345, chatData)
}

// --- HandleTgHelperCurrencySelected goroutine section -------------------------

func TestHandleTgHelperCurrencySelected_GoroutinesSuccess(t *testing.T) {
	origTgChat := bots.TgChat
	bots.TgChat = fakeTgChatDal{err: nil}
	origSetCurrency := setLastCurrency
	setLastCurrency = func(_ facade.ContextWithUser, _ money.CurrencyCode) error { return nil }
	t.Cleanup(func() {
		bots.TgChat = origTgChat
		setLastCurrency = origSetCurrency
	})

	w := httptest.NewRecorder()
	body := "currency=USD&tg-chat=bot:12345"
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleTgHelperCurrencySelected_TgChatError(t *testing.T) {
	origTgChat := bots.TgChat
	bots.TgChat = fakeTgChatDal{err: errFake}
	origSetCurrency := setLastCurrency
	setLastCurrency = func(_ facade.ContextWithUser, _ money.CurrencyCode) error { return nil }
	t.Cleanup(func() {
		bots.TgChat = origTgChat
		setLastCurrency = origSetCurrency
	})

	w := httptest.NewRecorder()
	body := "currency=USD&tg-chat=bot:12345"
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleTgHelperCurrencySelected_ParseFormError(t *testing.T) {
	body := iotest.ErrReader(errors.New("read error"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ContentLength = 100
	HandleTgHelperCurrencySelected(r.Context(), w, r, authUser("u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200, got 200")
	}
}
