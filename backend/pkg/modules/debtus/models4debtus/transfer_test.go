package models4debtus

import (
	"testing"

	"github.com/crediterra/money"
	"github.com/strongo/decimal"
)

// TestNewTransferData_Validate is the dalgo re-enablement of the old
// TestTransfer_LoadSaver: instead of datastore Save/Load round-trip it
// verifies that NewTransferData + Validate produce a persistable entity
// with the derived fields (BothUserIDs, BothCounterpartyIDs, From/To JSON)
// populated.
func TestNewTransferData_Validate(t *testing.T) {
	currency := money.CurrencyIRR
	creator := TransferCounterpartyInfo{
		UserID:      "1",
		ContactID:   "2",
		ContactName: "Test1",
	}
	counterparty := TransferCounterpartyInfo{
		ContactName: "Creator 1",
	}
	amount := money.NewAmount(currency, decimal.NewDecimal64p2FromFloat64(123.45))
	transfer := NewTransferData(creator.UserID, false, amount, &creator, &counterparty)

	if transfer.CreatorUserID != creator.UserID {
		t.Errorf("transfer.CreatorUserID = %q, want %q", transfer.CreatorUserID, creator.UserID)
	}
	if transfer.AmountInCents != amount.Value {
		t.Errorf("transfer.AmountInCents = %v, want %v", transfer.AmountInCents, amount.Value)
	}
	if transfer.Currency != currency {
		t.Errorf("transfer.Currency = %v, want %v", transfer.Currency, currency)
	}
	if !transfer.IsOutstanding {
		t.Error("a new non-return transfer should be outstanding")
	}
	if transfer.DtCreated.IsZero() {
		t.Error("transfer.DtCreated should be set")
	}

	if err := transfer.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(transfer.BothUserIDs) == 0 {
		t.Error("len(transfer.BothUserIDs) == 0")
	}
	if len(transfer.BothCounterpartyIDs) == 0 {
		t.Error("len(transfer.BothCounterpartyIDs) == 0")
	}
	if transfer.FromJson == "" {
		t.Error("transfer.FromJson is empty")
	}
	if transfer.ToJson == "" {
		t.Error("transfer.ToJson is empty")
	}
	if got := transfer.GetAmount(); got != amount {
		t.Errorf("transfer.GetAmount() = %v, want %v", got, amount)
	}
}

//func TestTransferDump(t *testing.T) {
//	now := time.Now()
//	litter.Config.HidePrivateFields = true
//	t.Log("litter.Config.HidePrivateFields = true: ", litter.Sdump(now))
//	litter.Config.HidePrivateFields = false
//	t.Log("litter.Config.HidePrivateFields = false: ", litter.Options{HidePrivateFields: false}.Sdump(now))
//}
