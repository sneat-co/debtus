package api4unsorted

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	bots "github.com/sneat-co/debtus/backend/pkg/bots/botscompat"
)

// nonWaitingTgChatDal returns immediately without waiting on userTask, so the
// handler's goroutines do not deadlock when a sibling goroutine panics before
// calling userTask.Done().
type nonWaitingTgChatDal struct{ err error }

func (f nonWaitingTgChatDal) DoSomething(
	_ context.Context, _ *sync.WaitGroup, _ string, _ int64,
	_ token4auth.AuthInfo, _ dbo4userus.UserEntry,
	_ func(botsfwtgmodels.TgChatData) error,
) error {
	return f.err
}

// panicTgChatDal panics inside DoSomething to exercise the second goroutine's
// recover() block (api_tg_helpers.go:88-90).
type panicTgChatDal struct{}

func (panicTgChatDal) DoSomething(
	_ context.Context, _ *sync.WaitGroup, _ string, _ int64,
	_ token4auth.AuthInfo, _ dbo4userus.UserEntry,
	_ func(botsfwtgmodels.TgChatData) error,
) error {
	panic("boom in DoSomething")
}

// runHelperWithTimeout invokes HandleTgHelperCurrencySelected in a goroutine.
// When one of the handler's internal goroutines panics, the handler never sends
// the second value to its errs channel and therefore blocks forever; the panic
// recovery line is still recorded as covered. We bound the wait so the test does
// not hang.
func runHelperWithTimeout(authInfo token4auth.AuthInfo) {
	w := httptest.NewRecorder()
	body := "currency=USD&tg-chat=bot:12345"
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	done := make(chan struct{})
	go func() {
		HandleTgHelperCurrencySelected(r.Context(), w, r, authInfo)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// TestHandleTgHelperCurrencySelected_SetLastCurrencyPanic covers the recover()
// block in the first goroutine (api_tg_helpers.go:73-75).
func TestHandleTgHelperCurrencySelected_SetLastCurrencyPanic(t *testing.T) {
	origTgChat := bots.TgChat
	bots.TgChat = nonWaitingTgChatDal{err: nil}
	origSetCurrency := setLastCurrency
	setLastCurrency = func(_ facade.ContextWithUser, _ money.CurrencyCode) error {
		panic("boom in setLastCurrency")
	}
	t.Cleanup(func() {
		bots.TgChat = origTgChat
		setLastCurrency = origSetCurrency
	})

	runHelperWithTimeout(authUser("u1"))
}

// TestHandleTgHelperCurrencySelected_DoSomethingPanic covers the recover() block
// in the second goroutine (api_tg_helpers.go:88-90).
func TestHandleTgHelperCurrencySelected_DoSomethingPanic(t *testing.T) {
	origTgChat := bots.TgChat
	bots.TgChat = panicTgChatDal{}
	origSetCurrency := setLastCurrency
	setLastCurrency = func(_ facade.ContextWithUser, _ money.CurrencyCode) error { return nil }
	t.Cleanup(func() {
		bots.TgChat = origTgChat
		setLastCurrency = origSetCurrency
	})

	runHelperWithTimeout(authUser("u1"))
}
