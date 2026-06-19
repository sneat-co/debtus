package apimapping

import (
	"context"
	"net/http"
	"testing"

	"github.com/strongo/strongoapp"
)

func TestInitApi(t *testing.T) {
	var registered []string
	handle := func(method, path string, handler strongoapp.HttpHandlerWithContext) {
		registered = append(registered, method+":"+path)
	}
	InitApi(handle)
	if len(registered) == 0 {
		t.Error("expected routes to be registered")
	}
}

// Ensure the handle func type compiles correctly
var _ strongoapp.HandleHttpWithContext = func(method, path string, handler strongoapp.HttpHandlerWithContext) {}

// Unused but satisfies compiler for context/http imports used in handle signature
var _ = context.Background
var _ = http.MethodGet
