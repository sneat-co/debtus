package debtus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sneat-co/sneat-go-core/extension"
	"github.com/strongo/delaying"
)

func TestModule(t *testing.T) {
	m := Extension()
	extension.AssertExtension(t, m, extension.Expected{
		ExtID:         extensionID,
		HandlersCount: 10,
		DelayersCount: 25,
	})
}

// TestExtension_handleWithContext exercises the handleWithContext closure
// that wraps strongoapp.HttpHandlerWithContext into a plain http.HandlerFunc.
func TestExtension_handleWithContext(t *testing.T) {
	m := Extension()

	// Capture the first http.HandlerFunc registered (receipt-ack-accept is a
	// trivial handler that just writes "ok"; it is always the 5th route but we
	// just grab whichever comes first and invoke it).
	var captured http.HandlerFunc
	mustRegister := func(key string, i any) delaying.Delayer {
		return delaying.NewDelayer(key, i,
			func(c context.Context, p delaying.Params, args ...any) error { return nil },
			func(c context.Context, p delaying.Params, args ...[]any) error { return nil },
		)
	}
	m.Register(extension.NewModuleRegistrationArgs(
		func(method, path string, handler http.HandlerFunc) {
			if captured == nil {
				captured = handler
			}
		},
		mustRegister,
	))

	if captured == nil {
		t.Fatal("no handler was registered")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	captured(w, r) // exercises the handleWithContext closure body
}
