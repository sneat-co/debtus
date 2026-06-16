package email4debtus

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/sneat-co/debtus/backend/internal/testutil"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp"
)

var (
	errSubj = errors.New("subj error")
	errText = errors.New("text error")
	errHtml = errors.New("html error")
	errSend = errors.New("send error")
)

func restoreSeams(orig struct {
	gt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	gh func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	se func(context.Context, emails.Email) (string, error)
	rt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	rh func(context.Context, *bytes.Buffer, i18n.SingleLocaleTranslator, string, any) error
}) {
	getEmailText = orig.gt
	getEmailHtml = orig.gh
	sendEmail = orig.se
	renderText = orig.rt
	renderHtml = orig.rh
}

func saveSeams() struct {
	gt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	gh func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	se func(context.Context, emails.Email) (string, error)
	rt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
	rh func(context.Context, *bytes.Buffer, i18n.SingleLocaleTranslator, string, any) error
} {
	return struct {
		gt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
		gh func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
		se func(context.Context, emails.Email) (string, error)
		rt func(context.Context, i18n.SingleLocaleTranslator, string, any) (string, error)
		rh func(context.Context, *bytes.Buffer, i18n.SingleLocaleTranslator, string, any) error
	}{getEmailText, getEmailHtml, sendEmail, renderText, renderHtml}
}

func TestSendInviteByEmail_Success(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	getEmailText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "ok", nil
	}
	getEmailHtml = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "<p>ok</p>", nil
	}
	sendEmail = func(_ context.Context, _ emails.Email) (string, error) {
		return "msg-1", nil
	}

	translator := testutil.NewPassthroughTranslator("en-US")
	ec := strongoapp.NewExecutionContext(context.Background())
	id, err := SendInviteByEmail(ec, translator, "Alice", "bob@example.com", "Bob", "code1", "sneat_bot", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "msg-1" {
		t.Fatalf("expected msg-1, got %q", id)
	}
}

func TestSendInviteByEmail_SubjError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	getEmailText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "", errSubj
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	ec := strongoapp.NewExecutionContext(context.Background())
	_, err := SendInviteByEmail(ec, translator, "Alice", "bob@example.com", "Bob", "code1", "sneat_bot", "test")
	if !errors.Is(err, errSubj) {
		t.Fatalf("expected errSubj, got %v", err)
	}
}

func TestSendInviteByEmail_BodyTextError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	call := 0
	getEmailText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		call++
		if call == 1 {
			return "subj", nil
		}
		return "", errText
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	ec := strongoapp.NewExecutionContext(context.Background())
	_, err := SendInviteByEmail(ec, translator, "Alice", "bob@example.com", "Bob", "code1", "sneat_bot", "test")
	if !errors.Is(err, errText) {
		t.Fatalf("expected errText, got %v", err)
	}
}

func TestSendInviteByEmail_HtmlError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	getEmailText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "ok", nil
	}
	getEmailHtml = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "", errHtml
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	ec := strongoapp.NewExecutionContext(context.Background())
	_, err := SendInviteByEmail(ec, translator, "Alice", "bob@example.com", "Bob", "code1", "sneat_bot", "test")
	if !errors.Is(err, errHtml) {
		t.Fatalf("expected errHtml, got %v", err)
	}
}

func TestSendReceiptByEmail_Success(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	renderText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "ok", nil
	}
	renderHtml = func(_ context.Context, buf *bytes.Buffer, _ i18n.SingleLocaleTranslator, _ string, _ any) error {
		buf.WriteString("<p>ok</p>")
		return nil
	}
	sendEmail = func(_ context.Context, _ emails.Email) (string, error) {
		return "msg-2", nil
	}

	translator := testutil.NewPassthroughTranslator("en-US")
	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		CreatedOn: general4debtus.CreatedOn{CreatedOnID: "bot1"},
	})
	id, err := SendReceiptByEmail(context.Background(), translator, receipt, "Alice", "Bob", "bob@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "msg-2" {
		t.Fatalf("expected msg-2, got %q", id)
	}
}

func TestSendReceiptByEmail_SubjError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	renderText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "", errSubj
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{})
	_, err := SendReceiptByEmail(context.Background(), translator, receipt, "Alice", "Bob", "bob@example.com")
	if !errors.Is(err, errSubj) {
		t.Fatalf("expected errSubj, got %v", err)
	}
}

func TestSendReceiptByEmail_BodyTextError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	call := 0
	renderText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		call++
		if call == 1 {
			return "subj", nil
		}
		return "", errText
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{})
	_, err := SendReceiptByEmail(context.Background(), translator, receipt, "Alice", "Bob", "bob@example.com")
	if !errors.Is(err, errText) {
		t.Fatalf("expected errText, got %v", err)
	}
}

func TestSendReceiptByEmail_HtmlError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	renderText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "ok", nil
	}
	renderHtml = func(_ context.Context, _ *bytes.Buffer, _ i18n.SingleLocaleTranslator, _ string, _ any) error {
		return errHtml
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{})
	_, err := SendReceiptByEmail(context.Background(), translator, receipt, "Alice", "Bob", "bob@example.com")
	if !errors.Is(err, errHtml) {
		t.Fatalf("expected errHtml, got %v", err)
	}
}

func TestSendReceiptByEmail_SendEmailError(t *testing.T) {
	orig := saveSeams()
	t.Cleanup(func() { restoreSeams(orig) })
	renderText = func(_ context.Context, _ i18n.SingleLocaleTranslator, _ string, _ any) (string, error) {
		return "ok", nil
	}
	renderHtml = func(_ context.Context, buf *bytes.Buffer, _ i18n.SingleLocaleTranslator, _ string, _ any) error {
		buf.WriteString("<p>ok</p>")
		return nil
	}
	sendEmail = func(_ context.Context, _ emails.Email) (string, error) {
		return "", errSend
	}
	translator := testutil.NewPassthroughTranslator("en-US")
	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		CreatedOn: general4debtus.CreatedOn{CreatedOnID: "bot1"},
	})
	_, err := SendReceiptByEmail(context.Background(), translator, receipt, "Alice", "Bob", "bob@example.com")
	if !errors.Is(err, errSend) {
		t.Fatalf("expected errSend, got %v", err)
	}
}
