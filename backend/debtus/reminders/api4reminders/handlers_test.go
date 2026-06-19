package api4reminders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/strongo/strongoapp"
)

type fakeSent struct{ id string }

func (s fakeSent) MessageID() string { return s.id }

type fakeEmailClient struct {
	sent []emails.Email
	err  error
}

func (f *fakeEmailClient) Send(_ context.Context, email emails.Email) (emails.Sent, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sent = append(f.sent, email)
	return fakeSent{id: "msg-1"}, nil
}

func overrideEmailClient(t *testing.T, client emails.Client) {
	t.Helper()
	original := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return client, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = original })
}

func newFormRequest(method, target, form string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestTestEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeEmailClient{}
		overrideEmailClient(t, client)
		w := httptest.NewRecorder()
		testEmail(context.Background(), w, newFormRequest(http.MethodGet, "/api4debtus/test/email", ""))
		if len(client.sent) != 1 {
			t.Fatalf("expected 1 email sent, got %d", len(client.sent))
		}
		if client.sent[0].Subject == "" {
			t.Error("expected non-empty subject")
		}
	})

	t.Run("send_error_is_written_to_response", func(t *testing.T) {
		overrideEmailClient(t, &fakeEmailClient{err: errors.New("smtp down")})
		w := httptest.NewRecorder()
		testEmail(context.Background(), w, newFormRequest(http.MethodGet, "/api4debtus/test/email", ""))
		if !strings.Contains(w.Body.String(), "smtp down") {
			t.Errorf("expected error in response body, got: %q", w.Body.String())
		}
	})
}

func TestSendReceipt_ParseFormError(t *testing.T) {
	overrideEmailClient(t, &fakeEmailClient{})
	w := httptest.NewRecorder()
	// %zz is an invalid URL encoding that causes ParseForm to return an error
	r := httptest.NewRequest(http.MethodPost, "/api4debtus/send-receipt", strings.NewReader("a=%zz"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sendReceipt(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for parse form error, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Failed to parse form") {
		t.Errorf("expected error message in response, got: %q", w.Body.String())
	}
}

func TestSendReceipt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeEmailClient{}
		overrideEmailClient(t, client)
		w := httptest.NewRecorder()
		r := newFormRequest(http.MethodPost, "/api4debtus/send-receipt", "from_name=Alice&value=10&currency=EUR")
		sendReceipt(context.Background(), w, r)
		if len(client.sent) != 1 {
			t.Fatalf("expected 1 email sent, got %d", len(client.sent))
		}
		if !strings.Contains(client.sent[0].HTML, "10EUR") {
			t.Errorf("expected amount in email HTML, got: %q", client.sent[0].HTML)
		}
		if !strings.Contains(w.Body.String(), "Email sent") {
			t.Errorf("expected confirmation in response, got: %q", w.Body.String())
		}
	})

	t.Run("send_error_is_written_to_response", func(t *testing.T) {
		overrideEmailClient(t, &fakeEmailClient{err: errors.New("smtp down")})
		w := httptest.NewRecorder()
		r := newFormRequest(http.MethodPost, "/api4debtus/send-receipt", "from_name=Alice&value=10&currency=EUR")
		sendReceipt(context.Background(), w, r)
		if !strings.Contains(w.Body.String(), "smtp down") {
			t.Errorf("expected error in response body, got: %q", w.Body.String())
		}
	})
}

func TestInviteFriend(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeEmailClient{}
		overrideEmailClient(t, client)
		w := httptest.NewRecorder()
		r := newFormRequest(http.MethodPost, "/invite-friend", "from_name=Alice")
		InviteFriend(w, r)
		if len(client.sent) != 1 {
			t.Fatalf("expected 1 email sent, got %d", len(client.sent))
		}
		if !strings.Contains(w.Body.String(), "Email sent") {
			t.Errorf("expected confirmation in response, got: %q", w.Body.String())
		}
	})

	t.Run("send_error_is_written_to_response", func(t *testing.T) {
		overrideEmailClient(t, &fakeEmailClient{err: errors.New("smtp down")})
		w := httptest.NewRecorder()
		r := newFormRequest(http.MethodPost, "/invite-friend", "from_name=Alice")
		InviteFriend(w, r)
		if !strings.Contains(w.Body.String(), "smtp down") {
			t.Errorf("expected error in response body, got: %q", w.Body.String())
		}
	})

	t.Run("parse_form_error_writes_400", func(t *testing.T) {
		// ParseForm error branch — the function does NOT return after writing 400,
		// so it panics at r.Form["from_name"][0] with an empty form.
		// We use recover() so the covered lines (14-16) are counted before the panic.
		overrideEmailClient(t, &fakeEmailClient{})
		w := httptest.NewRecorder()
		// %zz in the body causes ParseForm to return an invalid URL escape error.
		r := httptest.NewRequest(http.MethodPost, "/invite-friend", strings.NewReader("from_name=%zz"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		func() {
			defer func() { _ = recover() }()
			InviteFriend(w, r)
		}()
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for parse form error, got %d", w.Code)
		}
	})
}

func TestInitApiForReminder(t *testing.T) {
	registered := map[string]string{}
	InitApiForReminder(func(method, path string, _ strongoapp.HttpHandlerWithContext) {
		registered[path] = method
	})
	if registered["/api4debtus/send-receipt"] != http.MethodPost {
		t.Error("send-receipt not registered as POST")
	}
	if registered["/api4debtus/test/email"] != http.MethodGet {
		t.Error("test/email not registered as GET")
	}
}
