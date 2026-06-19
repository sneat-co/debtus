package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/strongo/gotwilio"
	"github.com/strongo/i18n"
)

// fakeTranslator returns the translation key unchanged.
type fakeTranslator struct{}

func (f fakeTranslator) Locale() i18n.Locale {
	return i18n.Locale{Code5: "en-US"}
}

func (f fakeTranslator) Translate(key string, _ ...any) string { return key }

func (f fakeTranslator) TranslateWithMap(key string, _ map[string]string) string { return key }

func (f fakeTranslator) TranslateNoWarning(key string, _ ...any) string { return key }

// roundTripFunc is an http.RoundTripper backed by a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// mockHTTPClient returns an *http.Client whose transport calls fn.
func mockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

// twilioSuccessResponse returns a 201 response with a minimal SmsResponse body.
func twilioSuccessResponse() *http.Response {
	body := `{"sid":"SM1","status":"queued","to":"+1234567890","from":"+0987654321","body":"test"}`
	return &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// twilioExceptionResponse returns a non-201 response with an Exception body.
func twilioExceptionResponse(code int) *http.Response {
	ex := gotwilio.Exception{Code: code, Message: "error"}
	b, _ := json.Marshal(ex)
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(string(b))),
	}
}

func TestTwilioExceptionToMessage_AllCodes(t *testing.T) {
	cases := []struct {
		code           int
		wantTryAnother bool
	}{
		{21211, true},
		{21614, true},
		{21612, true},
		{21408, true},
		{21610, true},
		{99999, false}, // unknown code → no message, no tryAnotherNumber
	}
	for _, tc := range cases {
		ex := &gotwilio.Exception{Code: tc.code, Message: "msg"}
		msg, tryAnother := TwilioExceptionToMessage(nil, fakeTranslator{}, ex)
		if tryAnother != tc.wantTryAnother {
			t.Errorf("code=%d: tryAnotherNumber=%v, want %v", tc.code, tryAnother, tc.wantTryAnother)
		}
		if tc.wantTryAnother && msg == "" {
			t.Errorf("code=%d: expected non-empty message", tc.code)
		}
	}
}

func TestSendSms_TransportError(t *testing.T) {
	// Covers the err != nil early-return after the first SendSMS call (send.go:41-43).
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		})
	}
	_, _, _, err := SendSms(context.Background(), false, "+12025550100", "hello")
	if err == nil {
		t.Error("expected error from transport failure")
	}
}

func TestSendSms_IsLiveFalse_Success(t *testing.T) {
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
			return twilioSuccessResponse(), nil
		})
	}
	_, _, exception, err := SendSms(context.Background(), false, "+12025550100", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exception != nil {
		t.Fatalf("unexpected exception: %+v", exception)
	}
}

func TestSendSms_IsLiveTrue_Success(t *testing.T) {
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
			return twilioSuccessResponse(), nil
		})
	}
	_, _, exception, err := SendSms(context.Background(), true, "+12025550100", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exception != nil {
		t.Fatalf("unexpected exception: %+v", exception)
	}
}

func TestSendSms_RussianPhoneCorrection(t *testing.T) {
	// Covers the branch where twilioException.Code==21211 and phone starts with "+8"
	// and length is 12. The second SMS attempt should be made with "+7" prefix.
	callCount := 0
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: return exception 21211 for a Russian-format number
				return twilioExceptionResponse(21211), nil
			}
			// Second call (corrected number): success
			return twilioSuccessResponse(), nil
		})
	}
	// "+79991234567" is 12 chars and starts with "+7" — but we need "+8" prefix.
	// "+89991234567" starts with "+8", length=12.
	_, smsResp, _, err := SendSms(context.Background(), false, "+89991234567", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls (original + corrected), got %d", callCount)
	}
	if smsResp == nil {
		t.Error("expected non-nil SmsResponse from corrected phone number call")
	}
}
