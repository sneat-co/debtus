package debtusdal

import (
	"context"
	"errors"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
)

func TestDelayer_enqueueErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("DelayCreateReminderForTransferUser", func(t *testing.T) {
		orig := delayer4debtus.CreateReminderForTransferUser
		delayer4debtus.CreateReminderForTransferUser = failingDelayer{id: "create-reminder"}
		t.Cleanup(func() { delayer4debtus.CreateReminderForTransferUser = orig })
		if err := NewReminderDal().DelayCreateReminderForTransferUser(ctx, "t1", "u1"); !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})

	t.Run("DelayUpdateTransfersWithCounterparty", func(t *testing.T) {
		orig := delayer4debtus.UpdateTransfersWithCounterparty
		delayer4debtus.UpdateTransfersWithCounterparty = failingDelayer{id: "update-transfers-with-cp"}
		t.Cleanup(func() { delayer4debtus.UpdateTransfersWithCounterparty = orig })
		if err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "cp2"); !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})

	t.Run("DelayUpdateTransferWithCreatorReceiptTgMessageID", func(t *testing.T) {
		orig := delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID
		delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID = failingDelayer{id: "tg-msg-id"}
		t.Cleanup(func() { delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID = orig })
		if err := NewTransferDal().DelayUpdateTransferWithCreatorReceiptTgMessageID(ctx, "bot", "t1", 1, 2); !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})

	t.Run("DelayUpdateTransfersOnReturn", func(t *testing.T) {
		orig := delayer4debtus.UpdateTransfersOnReturn
		delayer4debtus.UpdateTransfersOnReturn = failingDelayer{id: "on-return"}
		t.Cleanup(func() { delayer4debtus.UpdateTransfersOnReturn = orig })
		err := NewTransferDal().DelayUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
			{TransferID: "t1", ReturnedAmount: 100},
		})
		if !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})

	t.Run("delayDeleteContactTransfers", func(t *testing.T) {
		orig := delayer4debtus.DeleteContactTransfersDelayFunc
		delayer4debtus.DeleteContactTransfersDelayFunc = failingDelayer{id: "delete-contact-transfers"}
		t.Cleanup(func() { delayer4debtus.DeleteContactTransfersDelayFunc = orig })
		if err := delayDeleteContactTransfers(ctx, "c1", ""); !errors.Is(err, errInjected) {
			t.Errorf("expected errInjected, got %v", err)
		}
	})
}
