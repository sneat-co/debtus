package api4unsorted

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/bots"
	"github.com/strongo/delaying"
	"go.uber.org/mock/gomock"
)

// errRecordsReader returns an error on the first Next() call (not ErrNoMoreRecords),
// driving the dal.SelectAllIDs error branch.
type errRecordsReader struct{}

func (errRecordsReader) Next() (dal.Record, error) { return nil, errFake }
func (errRecordsReader) Cursor() (string, error)   { return "", nil }
func (errRecordsReader) Close() error              { return nil }

// TestDelayedChangeTransfersCounterparty_SelectAllIDsError covers api_admin.go:124-126:
// dal.SelectAllIDs returns an error when the reader errors mid-stream.
func TestDelayedChangeTransfersCounterparty_SelectAllIDsError(t *testing.T) {
	origDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, nil }
	origQuery := executeQueryToRecordsReader
	executeQueryToRecordsReader = func(_ context.Context, _ dal.DB, _ dal.Query) (dal.RecordsReader, error) {
		return errRecordsReader{}, nil
	}
	t.Cleanup(func() {
		facade.GetSneatDB = origDB
		executeQueryToRecordsReader = origQuery
	})

	err := DelayedChangeTransfersCounterparty(context.Background(), 1, 2, "")
	if err == nil {
		t.Error("expected error from SelectAllIDs, got nil")
	}
}

// errDelayer is a delaying.Delayer whose EnqueueWork always returns errFake.
type errDelayer struct{}

func (errDelayer) ID() string          { return "errDelayer" }
func (errDelayer) Implementation() any { return nil }
func (errDelayer) EnqueueWork(_ context.Context, _ delaying.Params, _ ...any) error {
	return errFake
}
func (errDelayer) EnqueueWorkMulti(_ context.Context, _ delaying.Params, _ ...[]any) error {
	return errFake
}

// TestHandleAdminMergeUserContacts_EnqueueWorkError covers api_admin.go:89-91:
// delayChangeTransfersCounterparty.EnqueueWork returns an error.
func TestHandleAdminMergeUserContacts_EnqueueWorkError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	// HasContact is false (empty map) → no Update, no Delete; EnqueueWork errors first.

	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxWithMock(mockTx)
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{}
		return nil
	}
	origDelayer := delayChangeTransfersCounterparty
	delayChangeTransfersCounterparty = errDelayer{}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
		delayChangeTransfersCounterparty = origDelayer
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// callbackTgChatDal invokes the callback (covering the closure that calls sendToTelegramFn)
// before returning the preset error.
type callbackTgChatDal struct{ err error }

func (f callbackTgChatDal) DoSomething(
	_ context.Context, userTask *sync.WaitGroup, _ string, _ int64,
	_ token4auth.AuthInfo, _ dbo4userus.UserEntry,
	cb func(botsfwtgmodels.TgChatData) error,
) error {
	if cbErr := cb(nil); cbErr != nil {
		userTask.Wait()
		return cbErr
	}
	userTask.Wait()
	return f.err
}

// TestHandleTgHelperCurrencySelected_CallbackInvoked covers api_tg_helpers.go lines 95-96:
// the goroutine closure that builds botID and calls sendToTelegramFn.
func TestHandleTgHelperCurrencySelected_CallbackInvoked(t *testing.T) {
	origTgChat := bots.TgChat
	bots.TgChat = callbackTgChatDal{err: nil}
	origSetCurrency := setLastCurrency
	setLastCurrency = func(_ facade.ContextWithUser, _ money.CurrencyCode) error { return nil }
	origSend := sendToTelegramFn
	sendToTelegramFn = func(_ context.Context, _ dbo4userus.UserEntry, _ string, _ int64, _ botsfwtgmodels.TgChatData, _ *sync.WaitGroup, _ *http.Request) error {
		return nil
	}
	t.Cleanup(func() {
		bots.TgChat = origTgChat
		setLastCurrency = origSetCurrency
		sendToTelegramFn = origSend
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

// TestSendToTelegram_GetBotSettingsError covers api_tg_helpers.go lines 115-119:
// GetBotSettingsByCode fails (no provider configured in tests) → early error return.
func TestSendToTelegram_GetBotSettingsError(t *testing.T) {
	tgChat := botsfwtgmodels.NewTelegramChatBaseData()
	var userTask sync.WaitGroup
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	err := sendToTelegram(context.Background(), dbo4userus.UserEntry{}, "unknown-bot", 12345, tgChat, &userTask, r)
	if err == nil {
		t.Error("expected error from GetBotSettingsByCode, got nil")
	}
}
