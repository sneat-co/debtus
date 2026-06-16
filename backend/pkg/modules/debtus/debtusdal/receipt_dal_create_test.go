package debtusdal

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

var zeroTime = time.Time{}

func TestReceiptDalGae_CreateReceipt(t *testing.T) {
	ctx := context.Background()

	seedDebtusUser := func(t *testing.T, db dal.DB, userID string) {
		t.Helper()
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewDebtusUserEntry(userID).Record)
		})
		if err != nil {
			t.Fatalf("seed debtus user: %v", err)
		}
	}

	t.Run("creates_receipt_and_increments_user_counter", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		const userID = "u1"
		seedDebtusUser(t, db, userID)

		receipt, err := NewReceiptDal().CreateReceipt(ctx, &models4debtus.ReceiptDbo{
			CreatorUserID: userID,
			TransferID:    "t1",
		})
		if err != nil {
			t.Fatalf("CreateReceipt() returned error: %v", err)
		}
		if receipt.ID == "" {
			t.Error("CreateReceipt() should assign a non-empty receipt ID")
		}

		// Verify the receipt is persisted
		loaded := models4debtus.NewReceipt(receipt.ID, nil)
		if err = db.Get(ctx, loaded.Record); err != nil {
			t.Fatalf("receipt not found after create: %v", err)
		}
		if loaded.Data.TransferID != "t1" {
			t.Errorf("receipt.TransferID = %q, want t1", loaded.Data.TransferID)
		}

		// Verify user counter was incremented
		debtusUser := models4debtus.NewDebtusUserEntry(userID)
		if err = db.Get(ctx, debtusUser.Record); err != nil {
			t.Fatalf("debtus user not found: %v", err)
		}
		if debtusUser.Data.CountOfReceiptsCreated != 1 {
			t.Errorf("CountOfReceiptsCreated = %d, want 1", debtusUser.Data.CountOfReceiptsCreated)
		}
	})

	t.Run("error_when_debtus_user_missing", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		_, err := NewReceiptDal().CreateReceipt(ctx, &models4debtus.ReceiptDbo{
			CreatorUserID: "missing-user",
		})
		if err == nil {
			t.Error("expected error when debtus user does not exist, got nil")
		}
	})
}

func TestReceiptDalGae_UpdateReceipt(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed a receipt
	initial := &models4debtus.ReceiptDbo{TransferID: "t1"}
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewReceipt("r1", initial).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Update it
	updated := &models4debtus.ReceiptDbo{TransferID: "t2"}
	err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return NewReceiptDal().UpdateReceipt(ctx, tx, models4debtus.NewReceipt("r1", updated))
	})
	if err != nil {
		t.Fatalf("UpdateReceipt() returned error: %v", err)
	}

	loaded := models4debtus.NewReceipt("r1", nil)
	if err = db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if loaded.Data.TransferID != "t2" {
		t.Errorf("after update TransferID = %q, want t2", loaded.Data.TransferID)
	}
}

func TestReceiptDalGae_GetReceiptByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_receipt_when_exists", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewReceipt("r42", &models4debtus.ReceiptDbo{TransferID: "t99"}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		receipt, err := NewReceiptDal().GetReceiptByID(ctx, db, "r42")
		if err != nil {
			t.Fatalf("GetReceiptByID() returned error: %v", err)
		}
		if receipt.ID != "r42" {
			t.Errorf("receipt.ID = %q, want r42", receipt.ID)
		}
		if receipt.Data.TransferID != "t99" {
			t.Errorf("receipt.TransferID = %q, want t99", receipt.Data.TransferID)
		}
	})

	t.Run("returns_not_found_for_missing_receipt", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		_, err := NewReceiptDal().GetReceiptByID(ctx, db, "no-such")
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}

func TestDelayedMarkReceiptAsSent_guards(t *testing.T) {
	ctx := context.Background()

	// These guard branches return nil before touching dal4debtus.Default.Receipt, so no DAL
	// registration is needed.
	t.Run("noop_when_receiptID_empty", func(t *testing.T) {
		if err := delayedMarkReceiptAsSent(ctx, "", "t1", zeroTime); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("noop_when_transferID_empty", func(t *testing.T) {
		if err := delayedMarkReceiptAsSent(ctx, "r1", "", zeroTime); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("noop_on_not_found", func(t *testing.T) {
		// Wire up dal4debtus.Default.Receipt so the not-found path can be exercised.
		sneattesting.SetupMemoryDB(t)
		RegisterDal()
		// Receipt and transfer don't exist — MarkReceiptAsSent returns
		// not-found, which delayedMarkReceiptAsSent should swallow.
		if err := delayedMarkReceiptAsSent(ctx, "no-receipt", "no-transfer", zeroTime); err != nil {
			t.Errorf("expected nil on not-found, got %v", err)
		}
	})
}
