package dal4debtus

import (
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
)

func TestNewTransferSourceBot_panics_on_empty_botID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when botID is empty")
		}
	}()
	NewTransferSourceBot("telegram", "", "12345")
}

func TestNewTransferSourceBot_panics_on_empty_chatID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when chatID is empty")
		}
	}()
	NewTransferSourceBot("telegram", "DebtusBot", "")
}

// PopulateTransfer for platform=="telegram" calls t.Creator() which panics
// unless CreatorUserID and FromJson/ToJson are fully populated.
// That path is exercised via integration tests; only the non-telegram branch
// is unit-testable with a zero-value TransferData.

func TestTransferSourceBot_PopulateTransfer_non_telegram(t *testing.T) {
	src := NewTransferSourceBot("viber", "ViberBot", "11111")
	data := &models4debtus.TransferData{}
	src.PopulateTransfer(data)

	if data.CreatedOnPlatform != "viber" {
		t.Errorf("CreatedOnPlatform = %q, want viber", data.CreatedOnPlatform)
	}
	if data.CreatedOnID != "ViberBot" {
		t.Errorf("CreatedOnID = %q, want ViberBot", data.CreatedOnID)
	}
}
