package debtusdal

import (
	"context"
	"testing"

	"github.com/strongo/decimal"

	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestDelayedUpdateTransferWithCounterparty(t *testing.T) {
	ctx := context.Background()

	t.Run("noop_when_transferID_empty", func(t *testing.T) {
		if err := delayedUpdateTransferWithCounterparty(ctx, "space1", "", "ccp"); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("noop_when_counterpartyCounterpartyID_empty", func(t *testing.T) {
		if err := delayedUpdateTransferWithCounterparty(ctx, "space1", "t1", ""); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("contact_not_found_returns_nil", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		// The counterparty contact does not exist -> GetContact returns not-found
		// -> the function logs and returns nil.
		if err := delayedUpdateTransferWithCounterparty(ctx, "space1", "t1", "ccp1"); err != nil {
			t.Errorf("expected nil when contact missing, got %v", err)
		}
	})
}

func TestDelayedUpdateTransferOnReturn(t *testing.T) {
	ctx := context.Background()

	t.Run("return_transfer_not_found_returns_nil", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setupDelayers(t)
		RegisterDal()
		// returnTransferID does not exist -> GetTransferByID returns not-found
		// -> err reset to nil and the transaction returns nil.
		if err := delayedUpdateTransferOnReturn(ctx, "no-return", "no-transfer", decimal.Decimal64p2(100)); err != nil {
			t.Errorf("expected nil when return transfer missing, got %v", err)
		}
	})
}
