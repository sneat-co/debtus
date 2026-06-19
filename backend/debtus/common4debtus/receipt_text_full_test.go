package common4debtus

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func newReceiptTestTransfer() models4debtus.TransferEntry {
	data := models4debtus.NewTransferData(
		"12",
		false,
		money.Amount{Currency: "EUR", Value: 98765},
		&models4debtus.TransferCounterpartyInfo{
			ContactID:   "23",
			ContactName: "John Whites",
			Note:        "from note",
			Comment:     "from comment",
		},
		&models4debtus.TransferCounterpartyInfo{
			UserID:   "12",
			UserName: "Anna Blacks",
			Note:     "to note",
			Comment:  "to comment",
		},
	)
	return models4debtus.NewTransfer("123", data)
}

func newTestTranslator() i18n.SingleLocaleTranslator {
	return i18n.NewSingleMapTranslator(i18n.LocaleEnUS, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
}

func TestTextReceiptForTransfer(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator()
	utmParams := utm.Params{Source: "BotIdUnitTest", Medium: "telegram", Campaign: "unit-test"}

	t.Run("for_counterparty_with_interest_due_and_partial_return", func(t *testing.T) {
		transfer := newReceiptTestTransfer()
		transfer.Data.TransferInterest = models4debtus.NewInterest("simple", 10, 7).WithMinimumPeriod(3)
		transfer.Data.DtDueOn = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		transfer.Data.AmountInCentsReturned = 100

		text := TextReceiptForTransfer(ctx, translator, transfer, "", ShowReceiptToCounterparty, utmParams)
		// Creator() is the to-side (UserID == CreatorUserID), Counterparty() is
		// the from-side, so the counterparty sees the from-side note.
		for _, want := range []string{
			"Anna Blacks",  // counterparty name
			"987.65 EUR",   // amount
			"2026-07-01",   // due date
			"from note",    // counterparty (from-side) note
			"from comment", // from-side comment
			"to comment",   // to-side comment
		} {
			if !strings.Contains(text, want) {
				t.Errorf("receipt text does not contain %q:\n%v", want, text)
			}
		}
		if strings.Contains(text, "to note") {
			t.Error("creator note should not be shown to counterparty")
		}
	})

	t.Run("autodetect_creator", func(t *testing.T) {
		transfer := newReceiptTestTransfer()
		text := TextReceiptForTransfer(ctx, translator, transfer, "12", ShowReceiptToAutodetect, utmParams)
		if !strings.Contains(text, "John Whites") {
			t.Errorf("receipt for creator should mention the counterparty contact:\n%v", text)
		}
		if !strings.Contains(text, "to note") {
			t.Error("creator (to-side) note should be shown to creator")
		}
	})

	t.Run("autodetect_counterparty_without_user", func(t *testing.T) {
		data := models4debtus.NewTransferData(
			"12",
			false,
			money.Amount{Currency: "EUR", Value: 500},
			&models4debtus.TransferCounterpartyInfo{ContactID: "23", ContactName: "John Whites"},
			&models4debtus.TransferCounterpartyInfo{UserID: "12", UserName: "Anna Blacks"},
		)
		transfer := models4debtus.NewTransfer("124", data)
		text := TextReceiptForTransfer(ctx, translator, transfer, "999", ShowReceiptToAutodetect, utmParams)
		if text == "" {
			t.Error("expected non-empty receipt text")
		}
	})

	t.Run("panics_on_missing_id", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		TextReceiptForTransfer(ctx, translator, models4debtus.TransferEntry{}, "", ShowReceiptToCreator, utmParams)
	})
}

func TestWriteTransferInterestAndDays(t *testing.T) {
	translator := newTestTranslator()
	transfer := newReceiptTestTransfer()
	// period of 1 day covers the trans.DAY branch; minimum period of 4 days
	// covers the trans.DAYS_234 branch and the min-period suffix.
	transfer.Data.TransferInterest = models4debtus.NewInterest("simple", 10, 1).WithMinimumPeriod(4)

	var buffer bytes.Buffer
	WriteTransferInterest(&buffer, transfer, translator)
	if buffer.Len() == 0 {
		t.Error("expected non-empty interest text")
	}

	transfer.Data.InterestPeriod = 7 // covers the trans.DAYS branch
	transfer.Data.InterestMinimumPeriod = 0
	buffer.Reset()
	WriteTransferInterest(&buffer, transfer, translator)
	if buffer.Len() == 0 {
		t.Error("expected non-empty interest text")
	}
}
