package analytics2debtus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/strongo/gamp"
)

// stubBuffer implements gamp.Buffer and records calls.
type stubBuffer struct {
	queued   []gamp.Message
	queueErr error
	flushErr error
}

func (s *stubBuffer) Queue(m gamp.Message) error {
	if s.queueErr != nil {
		return s.queueErr
	}
	s.queued = append(s.queued, m)
	return nil
}

func (s *stubBuffer) Flush() error {
	return s.flushErr
}

func withStubBuffer(t *testing.T, buf *stubBuffer) func() {
	orig := newGaBuffer
	newGaBuffer = func(_ context.Context) gamp.Buffer { return buf }
	return func() { newGaBuffer = orig }
}

func TestSendSingleMessage_nilCtx(t *testing.T) {
	var ctx context.Context
	if err := SendSingleMessage(ctx, nil); err == nil {
		t.Error("expected error on nil context")
	}
}

func TestSendSingleMessage_success(t *testing.T) {
	buf := &stubBuffer{}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	gaCommon := getGaCommon(nil, "user1", "en-US", "api")
	msg := gamp.NewEvent("cat", "act", gaCommon)
	if err := SendSingleMessage(ctx, msg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(buf.queued) != 1 {
		t.Errorf("expected 1 queued message, got %d", len(buf.queued))
	}
}

func TestSendSingleMessage_flushError(t *testing.T) {
	buf := &stubBuffer{flushErr: errors.New("flush failed")}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	gaCommon := getGaCommon(nil, "user1", "en-US", "api")
	msg := gamp.NewEvent("cat", "act", gaCommon)
	if err := SendSingleMessage(ctx, msg); err == nil {
		t.Error("expected error from flush")
	}
}

func TestGetGaCommon_nilRequest(t *testing.T) {
	c := getGaCommon(nil, "u1", "en-US", "platform")
	if c.UserAgent != "appengine" {
		t.Errorf("expected userAgent=appengine, got %q", c.UserAgent)
	}
}

func TestGetGaCommon_withRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("User-Agent", "TestBrowser/1.0")
	c := getGaCommon(r, "u1", "en-US", "platform")
	if c.UserAgent != "TestBrowser/1.0" {
		t.Errorf("expected userAgent=TestBrowser/1.0, got %q", c.UserAgent)
	}
}

func TestSendSingleMessage_queueError(t *testing.T) {
	buf := &stubBuffer{queueErr: errors.New("queue failed")}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	gaCommon := getGaCommon(nil, "user1", "en-US", "api")
	msg := gamp.NewEvent("cat", "act", gaCommon)
	if err := SendSingleMessage(ctx, msg); err == nil {
		t.Error("expected error from queue")
	}
}

func TestReminderSent(t *testing.T) {
	buf := &stubBuffer{}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	ReminderSent(ctx, "user1", "en-US", "api")
	if len(buf.queued) != 1 {
		t.Errorf("expected 1 queued message, got %d", len(buf.queued))
	}
}

func TestReminderSent_sendError(t *testing.T) {
	buf := &stubBuffer{flushErr: errors.New("send failed")}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	// Should not panic — error is logged internally
	ReminderSent(ctx, "user1", "en-US", "api")
}

func TestReceiptSentFromApi(t *testing.T) {
	buf := &stubBuffer{}
	defer withStubBuffer(t, buf)()

	ctx := context.Background()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("User-Agent", "TestClient/1.0")
	ReceiptSentFromApi(ctx, r, "user1", "en-US", "api4debtus", "email")
	if len(buf.queued) != 1 {
		t.Errorf("expected 1 queued message, got %d", len(buf.queued))
	}
}

func TestNewGaBuffer_defaultBody(t *testing.T) {
	// Exercise the default body of the newGaBuffer seam var.
	// gamp.NewBufferedClient is a pure constructor (no network I/O at construction time),
	// so calling the default body is safe without a live GA endpoint.
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return &http.Client{}
	}
	t.Cleanup(func() { dal4debtus.Default.HttpClient = nil })

	origNewGaBuffer := newGaBuffer
	buf := origNewGaBuffer(context.Background())
	if buf == nil {
		t.Error("expected non-nil gamp.Buffer from default newGaBuffer")
	}
}
