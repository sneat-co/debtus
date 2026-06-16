package debtusdal

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestReceiptDalGae_MarkReceiptAsSent(t *testing.T) {
	ctx := context.Background()
	sentTime := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	setup := func(t *testing.T, receiptData *models4debtus.ReceiptDbo, transferData *models4debtus.TransferData) dal.DB {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			if receiptData != nil {
				if err := tx.Set(ctx, models4debtus.NewReceipt("r1", receiptData).Record); err != nil {
					return err
				}
			}
			if transferData != nil {
				if err := tx.Set(ctx, models4debtus.NewTransfer("t1", transferData).Record); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to seed records: %v", err)
		}
		return db
	}

	getReceipt := func(t *testing.T, db dal.DB) *models4debtus.ReceiptDbo {
		receipt := models4debtus.NewReceipt("r1", nil)
		if err := db.Get(ctx, receipt.Record); err != nil {
			t.Fatalf("failed to get receipt: %v", err)
		}
		return receipt.Data
	}
	getTransfer := func(t *testing.T, db dal.DB) *models4debtus.TransferData {
		transfer := models4debtus.NewTransfer("t1", nil)
		if err := db.Get(ctx, transfer.Record); err != nil {
			t.Fatalf("failed to get transfer: %v", err)
		}
		return transfer.Data
	}

	t.Run("marks_sent_and_links_receipt_to_transfer", func(t *testing.T) {
		db := setup(t, &models4debtus.ReceiptDbo{TransferID: "t1"}, &models4debtus.TransferData{})
		if err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "t1", sentTime); err != nil {
			t.Fatalf("MarkReceiptAsSent() returned error: %v", err)
		}
		if dtSent := getReceipt(t, db).DtSent; !dtSent.Equal(sentTime) {
			t.Errorf("receipt.DtSent = %v, want %v", dtSent, sentTime)
		}
		transferData := getTransfer(t, db)
		if got := transferData.ReceiptIDs; len(got) != 1 || got[0] != "r1" {
			t.Errorf("transfer.ReceiptIDs = %v, want [r1]", got)
		}
		if transferData.ReceiptsSentCount != 1 {
			t.Errorf("transfer.ReceiptsSentCount = %d, want 1", transferData.ReceiptsSentCount)
		}
	})

	t.Run("resolves_transfer_id_from_receipt_when_empty", func(t *testing.T) {
		db := setup(t, &models4debtus.ReceiptDbo{TransferID: "t1"}, &models4debtus.TransferData{})
		if err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "", sentTime); err != nil {
			t.Fatalf("MarkReceiptAsSent() returned error: %v", err)
		}
		if got := getTransfer(t, db).ReceiptIDs; len(got) != 1 || got[0] != "r1" {
			t.Errorf("transfer.ReceiptIDs = %v, want [r1]", got)
		}
	})

	t.Run("noop_when_already_sent", func(t *testing.T) {
		alreadySent := sentTime.Add(-time.Hour)
		db := setup(t,
			&models4debtus.ReceiptDbo{TransferID: "t1", DtSent: alreadySent},
			&models4debtus.TransferData{ReceiptIDs: []string{"r1"}, ReceiptsSentCount: 1},
		)
		if err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "t1", sentTime); err != nil {
			t.Fatalf("MarkReceiptAsSent() returned error: %v", err)
		}
		if dtSent := getReceipt(t, db).DtSent; !dtSent.Equal(alreadySent) {
			t.Errorf("receipt.DtSent = %v, want unchanged %v", dtSent, alreadySent)
		}
		if got := getTransfer(t, db).ReceiptsSentCount; got != 1 {
			t.Errorf("transfer.ReceiptsSentCount = %d, want unchanged 1", got)
		}
	})

	t.Run("does_not_duplicate_receipt_id_in_transfer", func(t *testing.T) {
		db := setup(t,
			&models4debtus.ReceiptDbo{TransferID: "t1"},
			&models4debtus.TransferData{ReceiptIDs: []string{"r1"}, ReceiptsSentCount: 1},
		)
		if err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "t1", sentTime); err != nil {
			t.Fatalf("MarkReceiptAsSent() returned error: %v", err)
		}
		if dtSent := getReceipt(t, db).DtSent; !dtSent.Equal(sentTime) {
			t.Errorf("receipt.DtSent = %v, want %v", dtSent, sentTime)
		}
		transferData := getTransfer(t, db)
		if got := transferData.ReceiptIDs; len(got) != 1 {
			t.Errorf("transfer.ReceiptIDs = %v, want exactly one r1", got)
		}
		if transferData.ReceiptsSentCount != 1 {
			t.Errorf("transfer.ReceiptsSentCount = %d, want unchanged 1", transferData.ReceiptsSentCount)
		}
	})

	t.Run("error_when_receipt_missing", func(t *testing.T) {
		setup(t, nil, &models4debtus.TransferData{})
		err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "t1", sentTime)
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}
