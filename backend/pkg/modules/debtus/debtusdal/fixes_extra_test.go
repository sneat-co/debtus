package debtusdal

import (
	"context"
	"testing"

	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

// validTransferData returns a TransferData with the minimal valid From/To JSON
// so From()/To()/Direction()/Creator() don't panic. ContactName is left empty
// on the creator side so needFixCounterpartyCounterpartyName() reports true.
func validTransferData() *models4debtus.TransferData {
	return &models4debtus.TransferData{
		CreatorUserID: "u1",
		FromJson:      `{"userID":"u1","contactID":"c2"}`,
		ToJson:        `{"userID":"u2","contactID":"c3"}`,
	}
}

func TestTransferFixer_FixAllIfNeeded(t *testing.T) {
	ctx := context.Background()

	t.Run("no_fixes_needed_when_creator_has_contact_name", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		data := &models4debtus.TransferData{
			CreatorUserID: "u1",
			FromJson:      `{"userID":"u1","contactID":"c2","contactName":"Alice"}`,
			ToJson:        `{"userID":"u2","contactID":"c3"}`,
		}
		fixer := NewTransferFixer(models4debtus.NewTransferKey("t1"), data)
		if err := fixer.FixAllIfNeeded(ctx); err != nil {
			t.Errorf("FixAllIfNeeded() returned error: %v", err)
		}
	})

	t.Run("runs_transaction_when_fix_needed", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		RegisterDal()
		// Seed the transfer so GetTransferByID inside the transaction succeeds.
		seedTransfers(t, db, map[string]*models4debtus.TransferData{
			"t1": validTransferData(),
		})
		fixer := NewTransferFixer(models4debtus.NewTransferKey("t1"), validTransferData())
		if err := fixer.FixAllIfNeeded(ctx); err != nil {
			t.Errorf("FixAllIfNeeded() returned error: %v", err)
		}
	})

	t.Run("error_when_transfer_missing", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		RegisterDal()
		fixer := NewTransferFixer(models4debtus.NewTransferKey("missing"), validTransferData())
		if err := fixer.FixAllIfNeeded(ctx); err == nil {
			t.Error("expected error when transfer is missing, got nil")
		}
	})
}

func TestFixTransfers_emptyDB(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	loaded, fixed, failed, err := FixTransfers(ctx)
	if err != nil {
		t.Fatalf("FixTransfers() returned error: %v", err)
	}
	if loaded != 0 || fixed != 0 || failed != 0 {
		t.Errorf("FixTransfers() = (loaded=%d, fixed=%d, failed=%d), want all zero", loaded, fixed, failed)
	}
}

func TestFixTransfers_facadeDBError(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	withErroringFacadeDB(t)
	if _, _, _, err := FixTransfers(ctx); err == nil {
		t.Error("expected error from FixTransfers when facade DB fails")
	}
}
