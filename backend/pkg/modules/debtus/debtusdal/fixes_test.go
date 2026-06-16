package debtusdal

import (
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
)

func TestNewTransferFixer(t *testing.T) {
	data := &models4debtus.TransferData{}
	key := models4debtus.NewTransferKey("t1")
	fixer := NewTransferFixer(key, data)
	if fixer.transferKey != key {
		t.Error("NewTransferFixer: transferKey not set")
	}
	if fixer.transfer != data {
		t.Error("NewTransferFixer: transfer not set")
	}
	if fixer.Fixes == nil {
		t.Error("NewTransferFixer: Fixes slice should be initialised")
	}
}

// needFixCounterpartyCounterpartyName calls Creator() which in turn parses
// FromJson/ToJson and panics when they are empty or when CreatorUserID does
// not match either side. Exercising that path requires a fully populated
// TransferData with valid JSON counterparty fields — tested via integration
// paths only.
