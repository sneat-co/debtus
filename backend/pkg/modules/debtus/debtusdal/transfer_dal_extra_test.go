package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestTransferDalGae_LoadTransfersByUserID_validation(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)

	t.Run("error_when_limit_zero", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransfersByUserID(ctx, "u1", 0, 0)
		if err == nil {
			t.Error("expected error when limit==0, got nil")
		}
	})

	t.Run("error_when_userID_empty", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransfersByUserID(ctx, "", 0, 10)
		if err == nil {
			t.Error("expected error when userID is empty, got nil")
		}
	})
}

func TestTransferDalGae_LoadTransferIDsByContactID_validation(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)

	t.Run("error_when_limit_zero", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransferIDsByContactID(ctx, "c1", 0, "")
		if err == nil {
			t.Error("expected error when limit==0, got nil")
		}
	})

	t.Run("error_when_limit_over_1000", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransferIDsByContactID(ctx, "c1", 1001, "")
		if err == nil {
			t.Error("expected error when limit>1000, got nil")
		}
	})

	t.Run("error_when_contactID_empty", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransferIDsByContactID(ctx, "", 10, "")
		if err == nil {
			t.Error("expected error when contactID is empty, got nil")
		}
	})
}

func TestTransferDalGae_LoadTransfersByContactID_validation(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)

	t.Run("error_when_limit_zero", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransfersByContactID(ctx, "c1", 0, 0)
		if err == nil {
			t.Error("expected error when limit==0, got nil")
		}
	})

	t.Run("error_when_contactID_empty", func(t *testing.T) {
		_, _, err := NewTransferDal().LoadTransfersByContactID(ctx, "", 0, 10)
		if err == nil {
			t.Error("expected error when contactID is empty, got nil")
		}
	})
}

func TestTransferDalGae_GetTransfersByID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed two transfers
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		for _, id := range []string{"t1", "t2"} {
			if err := tx.Set(ctx, models4debtus.NewTransfer(id, &models4debtus.TransferData{}).Record); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	transfers, err := NewTransferDal().GetTransfersByID(ctx, db, []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("GetTransfersByID() returned error: %v", err)
	}
	if len(transfers) != 2 {
		t.Errorf("got %d transfers, want 2", len(transfers))
	}
}

func TestTransferDalGae_DelayUpdateTransfersWithCounterparty_validation(t *testing.T) {
	ctx := context.Background()

	t.Run("error_when_creatorCounterpartyID_empty", func(t *testing.T) {
		err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "space1", "", "cp2")
		if err == nil {
			t.Error("expected error when creatorCounterpartyID is empty, got nil")
		}
	})

	t.Run("error_when_counterpartyCounterpartyID_empty", func(t *testing.T) {
		err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "")
		if err == nil {
			t.Error("expected error when counterpartyCounterpartyID is empty, got nil")
		}
	})
}
