package vkapp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubRouter captures registered handlers for test invocation.
type stubRouter struct {
	handlers map[string]http.HandlerFunc
}

func (s *stubRouter) HandlerFunc(method, path string, handler http.HandlerFunc) {
	s.handlers[method+path] = handler
}

func TestInitVkIFrameApp(t *testing.T) {
	r := &stubRouter{handlers: make(map[string]http.HandlerFunc)}
	InitVkIFrameApp(r)
	if _, ok := r.handlers["GET/apps/vk/iframe"]; !ok {
		t.Error("expected handler registered for GET /apps/vk/iframe")
	}
}

func TestIFrameHandler_panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected IFrameHandler to panic")
		}
	}()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/apps/vk/iframe", nil)
	IFrameHandler(w, r)
}
