package common4debtus

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

// ---- SignStrWithExpiry ----

func TestSignStrWithExpiry(t *testing.T) {
	_, err := SignStrWithExpiry(context.Background(), "test", time.Now())
	if err == nil {
		t.Error("expected error from SignStrWithExpiry")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' in error, got: %v", err)
	}
}

// ---- WriteCounterpartyUrl error branch ----

func TestWriteCounterpartyUrl_tokenError(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("token error")
	}

	var buf bytes.Buffer
	err := WriteCounterpartyUrl(context.Background(), &buf, "cp1", "user1", i18n.LocaleEnUS, utm.Params{})
	if err == nil {
		t.Error("expected error from WriteCounterpartyUrl when token fails")
	}
}

// ---- getUrlForUser default branch ----

func TestGetUrlForUser_defaultBranch(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "tok", nil
	}
	// Pass a page that is neither "history" nor "debts" to hit the default branch.
	result := getUrlForUser(context.Background(), 42, i18n.LocaleEnUS, "unknown_page", "platform", "id")
	if !strings.Contains(result, "page=unknown_page") {
		t.Errorf("expected 'page=unknown_page' in URL, got: %v", result)
	}
}

// ---- newReceiptTextBuilder panic branches ----

func newTestTransfer() models4debtus.TransferEntry {
	return models4debtus.NewTransfer("t1", models4debtus.NewTransferData(
		"u1",
		false,
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
}

func TestNewReceiptTextBuilder_panicOnEmptyID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty transferID")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	newReceiptTextBuilder(translator, models4debtus.TransferEntry{}, ShowReceiptToCreator)
}

func TestNewReceiptTextBuilder_showToCounterparty(t *testing.T) {
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// ShowReceiptToCounterparty sets viewerUserID to Counterparty().UserID
	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCounterparty)
	if r.viewerUserID != "u2" {
		t.Errorf("viewerUserID = %q, want u2", r.viewerUserID)
	}
}

func TestNewReceiptTextBuilder_unknownShowReceiptTo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unknown ShowReceiptTo")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	newReceiptTextBuilder(translator, transfer, ShowReceiptToAutodetect) // autodetect panics
}

func TestNewReceiptTextBuilder_invalidDirection(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on invalid direction")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	// Build builder directly with ShowReceiptToCreator but set internal partyAction to
	// an invalid value, then call WriteReceiptText to hit the panic.
	transfer := newTestTransfer()
	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	r.partyAction = ReceiptPartyAction(99) // invalid
	var buf bytes.Buffer
	_ = r.WriteReceiptText(context.Background(), &buf, utm.Params{Medium: "telegram"})
}

// TestNewReceiptTextBuilder_3dPartyDirection covers receipt_text.go:73-77 (else branch when Direction is 3d-party).
func TestNewReceiptTextBuilder_3dPartyDirection(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on 3d-party direction")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	// Creator "u3" is neither from "u1" nor to "u2" → Direction() returns TransferDirection3dParty
	transfer := models4debtus.NewTransfer("t-3d", models4debtus.NewTransferData(
		"u3",
		false,
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
	// ShowReceiptToCreator with 3d-party direction → hits else branch at line 73 → panics
	newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
}

// ---- getReceiptCounterparty default panic branch ----

func TestGetReceiptCounterparty_defaultPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unknown ShowReceiptTo")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// Build a valid builder first, then mutate showReceiptTo to invalid value
	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	r.showReceiptTo = ShowReceiptTo(99)
	r.getReceiptCounterparty()
}

// ---- TextReceiptForTransfer panic branches ----

func TestTextReceiptForTransfer_creatorMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on creator user ID mismatch")
		}
	}()
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// ShowReceiptToCreator with mismatched showToUserID panics
	TextReceiptForTransfer(ctx, translator, transfer, "wrong_user", ShowReceiptToCreator, utm.Params{Medium: "telegram"})
}

func TestTextReceiptForTransfer_counterpartyMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on counterparty user ID mismatch")
		}
	}()
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// ShowReceiptToCounterparty with wrong userID panics
	TextReceiptForTransfer(ctx, translator, transfer, "wrong_user", ShowReceiptToCounterparty, utm.Params{Medium: "telegram"})
}

func TestTextReceiptForTransfer_autodetectNobodyPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on autodetect with unrelated userID and non-empty counterparty.UserID")
		}
	}()
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	// Transfer where both from.UserID and to.UserID are set
	transfer := models4debtus.NewTransfer("t2", models4debtus.NewTransferData(
		"u1",
		false,
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
	// showToUserID doesn't match either user → panic
	TextReceiptForTransfer(ctx, translator, transfer, "u999", ShowReceiptToAutodetect, utm.Params{Medium: "telegram"})
}

// ---- WriteReceiptText — IsReturn branches ----

func TestWriteReceiptText_isReturn(t *testing.T) {
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := models4debtus.NewTransfer("t3", models4debtus.NewTransferData(
		"u1",
		true, // IsReturn
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
	// ShowReceiptToCreator with IsReturn=true covers MESSAGE_TEXT_RECEIPT_RETURN_FROM_USER
	result := TextReceiptForTransfer(ctx, translator, transfer, "u1", ShowReceiptToCreator, utm.Params{Medium: "telegram"})
	if result == "" {
		t.Error("expected non-empty receipt text")
	}
}

func TestWriteReceiptText_isReturn_toUser(t *testing.T) {
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := models4debtus.NewTransfer("t4", models4debtus.NewTransferData(
		"u1",
		true, // IsReturn
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
	// ShowReceiptToCounterparty with IsReturn=true covers MESSAGE_TEXT_RECEIPT_RETURN_TO_USER
	result := TextReceiptForTransfer(ctx, translator, transfer, "u2", ShowReceiptToCounterparty, utm.Params{Medium: "telegram"})
	if result == "" {
		t.Error("expected non-empty receipt text")
	}
}

// ---- WriteReceiptText — amountReturned branch ----

func TestWriteReceiptText_amountReturned(t *testing.T) {
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// Set partial return so amountReturned > 0 and != AmountInCents
	transfer.Data.AmountInCentsReturned = 10 // partial: 10 of 100 returned
	result := TextReceiptForTransfer(ctx, translator, transfer, "u1", ShowReceiptToCreator, utm.Params{Medium: "telegram"})
	if result == "" {
		t.Error("expected non-empty receipt text")
	}
}

// ---- WriteReceiptText — IsReturn=true with invalid partyAction panics ----

func TestWriteReceiptText_isReturn_unknownPartyAction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unknown partyAction with IsReturn=true")
		}
	}()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := models4debtus.NewTransfer("t5", models4debtus.NewTransferData(
		"u1",
		true, // IsReturn
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	))
	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	r.partyAction = ReceiptPartyAction(99) // invalid
	var buf bytes.Buffer
	_ = r.WriteReceiptText(context.Background(), &buf, utm.Params{Medium: "telegram"})
}

// ---- TextReceiptForTransfer — nil Data panics ----

func TestTextReceiptForTransfer_nilData(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil transfer.Data")
		}
	}()
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	// Transfer with non-empty ID but nil Data
	transfer := models4debtus.TransferEntry{}
	transfer.ID = "t-nil-data"
	// Data is nil by default
	TextReceiptForTransfer(ctx, translator, transfer, "", ShowReceiptToCreator, utm.Params{Medium: "telegram"})
}

// ---- TextReceiptForTransfer autodetect — match Counterparty().UserID ----

func TestTextReceiptForTransfer_autodetect_matchesCounterparty(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "tok", nil
	}
	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// showToUserID == Counterparty().UserID ("u2") → autodetect sets ShowReceiptToCounterparty
	result := TextReceiptForTransfer(ctx, translator, transfer, "u2", ShowReceiptToAutodetect, utm.Params{Medium: "telegram"})
	if result == "" {
		t.Error("expected non-empty receipt text")
	}
}

// ---- translateAndFormatMessage — GetCounterpartyUrl error ----
// Also covers WriteReceiptText returned-amount and outstanding-amount error paths.

func TestTranslateAndFormatMessage_counterpartyUrlError(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("token error")
	}

	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	// Use empty utm.Params so ShortString() is never called (IsEmpty()==true skips that branch).
	// viewerUserID = "u1" (non-empty), medium == "" (not "telegram") → GetCounterpartyUrl is called → token fails → error returned → WriteReceiptText panics.
	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	var buf bytes.Buffer
	// translateAndFormatMessage returns error → WriteReceiptText panics at line 220
	defer func() { _ = recover() }()
	_ = r.WriteReceiptText(ctx, &buf, utm.Params{})
}

// ---- WriteReceiptText — partial return, translateAndFormatMessage error path ----

// TestWriteReceiptText_returnedAmount_translateError covers receipt_text.go:235 (return err on returned amount).
func TestWriteReceiptText_returnedAmount_translateError(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	callCount := 0
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		if callCount >= 2 {
			// Fail on 2nd call (returned-amount translateAndFormatMessage)
			return "", errors.New("token error on second call")
		}
		return "tok", nil
	}

	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	transfer.Data.AmountInCentsReturned = 10 // partial return → triggers returned-amount section

	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	// Empty utm so ShortString is not called; medium="" so GetCounterpartyUrl is called → uses token
	var buf bytes.Buffer
	err := r.WriteReceiptText(ctx, &buf, utm.Params{})
	if err == nil {
		t.Error("expected error from returned-amount translateAndFormatMessage")
	}
}

// TestWriteReceiptText_outstandingAmount_translateError covers receipt_text.go:243 (return err on outstanding amount).
func TestWriteReceiptText_outstandingAmount_translateError(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	callCount := 0
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		if callCount >= 3 {
			// Fail on 3rd call (outstanding-amount translateAndFormatMessage)
			return "", errors.New("token error on third call")
		}
		return "tok", nil
	}

	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	transfer.Data.AmountInCentsReturned = 10 // partial return → both returned-amount and outstanding-amount sections triggered

	r := newReceiptTextBuilder(translator, transfer, ShowReceiptToCreator)
	var buf bytes.Buffer
	err := r.WriteReceiptText(ctx, &buf, utm.Params{})
	if err == nil {
		t.Error("expected error from outstanding-amount translateAndFormatMessage")
	}
}

// TestTextReceiptForTransfer_writeReceiptError covers receipt_text.go:174 (panic(err) in TextReceiptForTransfer).
func TestTextReceiptForTransfer_writeReceiptError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from TextReceiptForTransfer when WriteReceiptText returns error")
		}
	}()
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	callCount := 0
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		if callCount >= 2 {
			return "", errors.New("token error")
		}
		return "tok", nil
	}

	ctx := context.Background()
	translator := i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))
	transfer := newTestTransfer()
	transfer.Data.AmountInCentsReturned = 10 // triggers secondary translateAndFormatMessage call that will fail
	// TextReceiptForTransfer calls WriteReceiptText which will return an error → panic at line 174
	TextReceiptForTransfer(ctx, translator, transfer, "u1", ShowReceiptToCreator, utm.Params{})
}

// ---- WriteCounterpartyUrl — token error covers models.go:37 ----

func TestWriteCounterpartyUrl_tokenErrorReturn(t *testing.T) {
	orig := token4auth.IssueBotToken
	defer func() { token4auth.IssueBotToken = orig }()
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("forced token error")
	}

	var w bytes.Buffer
	// currentUserID must be non-empty and not "0" to reach the token call
	err := WriteCounterpartyUrl(context.Background(), &w, "cp1", "realuser", i18n.LocaleEnUS, utm.Params{})
	if err == nil {
		t.Error("expected error from WriteCounterpartyUrl when token fails")
	}
}
