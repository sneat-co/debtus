package splitus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sneat-co/sneat-go-core/extension"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/const4splitus"
	"github.com/strongo/delaying"
)

func TestModule(t *testing.T) {
	m := Module()
	extension.AssertExtension(t, m, extension.Expected{
		ExtID:         const4splitus.ModuleID,
		HandlersCount: 2,
		DelayersCount: 9,
	})
}

func TestModule_RouteAdapterInvokesHandler(t *testing.T) {
	m := Module()
	if m == nil {
		t.Fatal("Module() must not return nil")
	}

	// Capture the first registered http.HandlerFunc via the handle func.
	var capturedHandler http.HandlerFunc
	handle := func(method, path string, handler http.HandlerFunc) {
		if capturedHandler == nil {
			capturedHandler = handler
		}
	}
	mustRegister := func(key string, i any) delaying.Delayer {
		enqueueWork := func(_ context.Context, _ delaying.Params, _ ...any) error { return nil }
		enqueueWorkMulti := func(_ context.Context, _ delaying.Params, _ ...[]any) error { return nil }
		return delaying.NewDelayer(key, func() {}, enqueueWork, enqueueWorkMulti)
	}
	delaying.Init(mustRegister)
	args := extension.NewModuleRegistrationArgs(handle, mustRegister)
	m.Register(args)

	if capturedHandler == nil {
		t.Fatal("expected at least one handler to be registered")
	}

	// Invoke the adapter closure — this covers the `handler(request.Context(), writer, request)` line.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	capturedHandler(rec, req)
}
