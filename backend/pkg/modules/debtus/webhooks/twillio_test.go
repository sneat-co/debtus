package webhooks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

// txErrDB wraps a dal.DB and always returns a fixed error from RunReadwriteTransaction.
type txErrDB struct {
	dal.DB
	err error
}

func (d txErrDB) RunReadwriteTransaction(_ context.Context, _ dal.RWTxWorker, _ ...dal.TransactionOption) error {
	return d.err
}

func overrideMemDB(t *testing.T) dal.DB {
	t.Helper()
	db := dalgo2memory.NewDB()
	original := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return db, nil
	}
	t.Cleanup(func() { facade.GetSneatDB = original })
	return db
}

func postTwilio(smsSid, messageStatus string) (*httptest.ResponseRecorder, *http.Request) {
	form := url.Values{"SmsSid": {smsSid}, "MessageStatus": {messageStatus}}
	r := httptest.NewRequest(http.MethodPost, "/webhooks/twilio/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	return w, r
}

func seedSms(t *testing.T, db dal.DB, smsSid, status string) {
	t.Helper()
	ctx := context.Background()
	data := &models4debtus.TwilioSmsDbo{}
	data.Status = status
	sms := models4debtus.NewTwilioSms(smsSid, data)
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, sms.Record)
	})
	if err != nil {
		t.Fatalf("seedSms: %v", err)
	}
}

func TestTwilioWebhook_Success_StatusChange(t *testing.T) {
	db := overrideMemDB(t)
	seedSms(t, db, "SM1", "queued")

	w, r := postTwilio("SM1", "sent")
	TwilioWebhook(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTwilioWebhook_Success_SameStatus(t *testing.T) {
	db := overrideMemDB(t)
	seedSms(t, db, "SM2", "sent")

	w, r := postTwilio("SM2", "sent")
	TwilioWebhook(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTwilioWebhook_Success_DeliveredStatus(t *testing.T) {
	db := overrideMemDB(t)
	seedSms(t, db, "SM3", "sent")

	w, r := postTwilio("SM3", "delivered")
	TwilioWebhook(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTwilioWebhook_NotFound(t *testing.T) {
	overrideMemDB(t)

	w, r := postTwilio("unknown_sid", "sent")
	TwilioWebhook(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTwilioWebhook_ParseFormError(t *testing.T) {
	// Invalid percent encoding in URL causes ParseForm to return an error.
	r := httptest.NewRequest(http.MethodPost, "/?a=%zz", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	TwilioWebhook(w, r)
	// ParseForm error causes early return; no status code is written so default is 200.
}

func TestTwilioWebhook_TxError(t *testing.T) {
	// Seed a real in-memory DB so facade.GetSneatDB returns a txErrDB wrapper
	// that fails RunReadwriteTransaction with a non-NotFound error, exercising
	// the else branch at the bottom of TwilioWebhook.
	base := dalgo2memory.NewDB()
	txErr := errors.New("simulated tx failure")
	original := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return txErrDB{DB: base, err: txErr}, nil
	}
	t.Cleanup(func() { facade.GetSneatDB = original })

	w, r := postTwilio("SM6", "sent")
	TwilioWebhook(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTwilioWebhook_GetDBError(t *testing.T) {
	original := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return nil, context.DeadlineExceeded
	}
	t.Cleanup(func() { facade.GetSneatDB = original })

	w, r := postTwilio("SM5", "sent")
	TwilioWebhook(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
