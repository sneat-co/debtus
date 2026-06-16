package apigaedepended

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/strongo/strongoapp"
)

// stubAppHost implements strongoapp.HttpAppHost for tests.
type stubAppHost struct{}

func (s stubAppHost) GetEnvironment(_ context.Context, _ *http.Request) string { return "test" }
func (s stubAppHost) HandleWithContext(handler strongoapp.HttpHandlerWithContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(context.Background(), w, r)
	}
}

func TestInitApiGaeDepended(t *testing.T) {
	orig := dal4debtus.HttpAppHost
	defer func() { dal4debtus.HttpAppHost = orig }()
	dal4debtus.HttpAppHost = stubAppHost{}

	registered := make(map[string]bool)
	origHandle := handleFunc
	defer func() { handleFunc = origHandle }()
	handleFunc = func(pattern string, handler func(http.ResponseWriter, *http.Request)) {
		registered[pattern] = true
	}

	InitApiGaeDepended()

	for _, path := range []string{"/auth/google/signin", "/auth/google/signed"} {
		if !registered[path] {
			t.Errorf("expected %v to be registered", path)
		}
	}
}

func TestHandleSigninWithGoogle(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/google/signin", nil)
	handleSigninWithGoogle(nil, w, r) //nolint:staticcheck
	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}
}

func TestHandleSignedWithGoogle(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/google/signed", nil)
	handleSignedWithGoogle(nil, w, r) //nolint:staticcheck
	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}
}
