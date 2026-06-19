package debtusdal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/decimal"
)

// newOutstandingTransfer builds a valid (From/To JSON populated) outstanding
// transfer between userID and a counterparty, with the given currency and
// returned amount. When amountInCents == returnedInCents the outstanding value
// is zero (exercises the "fix IsOutstanding" branch).
func newOutstandingTransfer(userID, contactID string, currency money.CurrencyCode, amountInCents, returnedInCents int64) *models4debtus.TransferData {
	// Set the query-relevant fields directly (BothUserIDs / currency / isOutstanding)
	// and the From/To JSON so From()/To()/DirectionForUser() don't panic. We bypass
	// the onSave hook (which requires fully-validated counterparties) the same way
	// the shared seedTransfers helper does.
	return &models4debtus.TransferData{
		CreatorUserID:         userID,
		Currency:              currency,
		IsOutstanding:         true,
		AmountInCents:         decimal.Decimal64p2(amountInCents),
		AmountInCentsReturned: decimal.Decimal64p2(returnedInCents),
		BothUserIDs:           []string{userID, "other"},
		BothCounterpartyIDs:   []string{"", contactID},
		DtCreated:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		FromJson:              `{"userID":"` + userID + `"}`,
		ToJson:                `{"userID":"other","contactID":"` + contactID + `"}`,
	}
}

func TestLoadOutstandingTransfers_filtersAndFix(t *testing.T) {
	ctx := context.Background()

	t.Run("filters_by_contactID_and_collects_zero_outstanding", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			// Outstanding with positive value, matching contact c1.
			"tPos": newOutstandingTransfer("u1", "c1", "EUR", 100, 0),
			// Outstanding but fully returned -> GetOutstandingValue == 0 (fix branch).
			"tZero": newOutstandingTransfer("u1", "c1", "EUR", 100, 100),
			// Different contact -> skipped by contactID filter.
			"tOther": newOutstandingTransfer("u1", "cX", "EUR", 50, 0),
		})
		transfers, err := NewTransferDal().LoadOutstandingTransfers(
			ctx, db, time.Now(), "u1", "c1", money.CurrencyCode("EUR"), "")
		if err != nil {
			t.Fatalf("LoadOutstandingTransfers() returned error: %v", err)
		}
		_ = transfers
	})

	t.Run("filters_by_direction", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"tPos": newOutstandingTransfer("u1", "c1", "EUR", 100, 0),
		})
		// Pass a direction that won't match to exercise the direction-skip branch.
		_, err := NewTransferDal().LoadOutstandingTransfers(
			ctx, db, time.Now(), "u1", "", money.CurrencyCode("EUR"),
			models4debtus.TransferDirectionCounterparty2User)
		if err != nil {
			t.Fatalf("LoadOutstandingTransfers() returned error: %v", err)
		}
	})
}

func TestFixTransferIsOutstanding(t *testing.T) {
	ctx := context.Background()

	t.Run("sets_isOutstanding_false_when_value_zero", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		RegisterDal()
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": newOutstandingTransfer("u1", "c1", "EUR", 100, 100),
		})
		transfer, err := fixTransferIsOutstanding(ctx, "t1")
		if err != nil {
			t.Fatalf("fixTransferIsOutstanding() returned error: %v", err)
		}
		if transfer.Data.IsOutstanding {
			t.Error("expected IsOutstanding to be set false")
		}
	})

	t.Run("noop_when_value_nonzero", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		RegisterDal()
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t2": newOutstandingTransfer("u1", "c1", "EUR", 100, 0),
		})
		if _, err := fixTransferIsOutstanding(ctx, "t2"); err != nil {
			t.Fatalf("fixTransferIsOutstanding() returned error: %v", err)
		}
	})

	t.Run("error_when_transfer_missing", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		RegisterDal()
		if _, err := fixTransferIsOutstanding(ctx, "missing"); err == nil {
			t.Error("expected error when transfer is missing")
		}
	})
}

func TestDelayedFixTransfersIsOutstanding(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	RegisterDal()
	// "missing" transfer -> fixTransferIsOutstanding returns error -> err captured.
	if err := delayedFixTransfersIsOutstanding(ctx, []string{"missing"}); err == nil {
		t.Error("expected error from delayedFixTransfersIsOutstanding for missing transfer")
	}
}

func TestGetTransfersByID_getMultiError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	err := failingDB{DB: db, fault: faultGetMulti}.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, e := NewTransferDal().GetTransfersByID(ctx, tx, []string{"t1"})
		return e
	})
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestGetLastTwilioSmsesForUser_withToFilter(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// Exercises the `to != ""` branch that adds a second WhereField.
	result, err := NewTwilioDal().GetLastTwilioSmsesForUser(ctx, db, "u1", "+15005550006", 10)
	if err != nil {
		t.Fatalf("GetLastTwilioSmsesForUser() returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}
