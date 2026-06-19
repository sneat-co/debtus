package debtusdal

import (
	"context"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/strongo/decimal"
	"github.com/strongo/delaying"
)

func setupDelayers(t *testing.T) {
	t.Helper()
	RegisterDelayers4Debtus(delaying.VoidWithLog)
}

func TestDelayUpdateTransfersOnReturn_panics(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)

	t.Run("panics_when_returnTransferID_empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when returnTransferID is empty")
			}
		}()
		_ = NewTransferDal().DelayUpdateTransfersOnReturn(ctx, "", []dal4debtus.TransferReturnUpdate{
			{TransferID: "t1", ReturnedAmount: 100},
		})
	})

	t.Run("panics_when_updates_empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when transferReturnsUpdate is empty")
			}
		}()
		_ = NewTransferDal().DelayUpdateTransfersOnReturn(ctx, "rt1", nil)
	})

	t.Run("panics_when_update_transferID_empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when update TransferID is empty")
			}
		}()
		_ = NewTransferDal().DelayUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
			{TransferID: "", ReturnedAmount: 100},
		})
	})

	t.Run("panics_when_returned_amount_zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when ReturnedAmount <= 0")
			}
		}()
		_ = NewTransferDal().DelayUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
			{TransferID: "t1", ReturnedAmount: 0},
		})
	})
}

func TestDelayUpdateTransferOnReturn_enqueues(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)

	err := DelayUpdateTransferOnReturn(ctx, "rt1", "t1", decimal.Decimal64p2(100))
	if err != nil {
		t.Errorf("DelayUpdateTransferOnReturn() returned error: %v", err)
	}
}

func TestDelayedUpdateTransfersOnReturn_panics(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)

	t.Run("panics_when_update_transferID_empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when TransferID is empty in delayedUpdateTransfersOnReturn")
			}
		}()
		_ = delayedUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
			{TransferID: "", ReturnedAmount: 100},
		})
	})

	t.Run("panics_when_returned_amount_zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when ReturnedAmount <= 0 in delayedUpdateTransfersOnReturn")
			}
		}()
		_ = delayedUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
			{TransferID: "t1", ReturnedAmount: 0},
		})
	})
}
