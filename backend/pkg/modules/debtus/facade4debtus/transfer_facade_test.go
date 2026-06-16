package facade4debtus

import (
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
)

func Test_removeClosedTransfersFromOutstandingWithInterest(t *testing.T) {
	transfersWithInterest := []models4debtus.TransferWithInterestJson{
		{TransferID: "1"},
		{TransferID: "2"},
		{TransferID: "3"},
		{TransferID: "4"},
		{TransferID: "5"},
	}
	transfersWithInterest = removeClosedTransfersFromOutstandingWithInterest(transfersWithInterest, []string{"2", "3"})
	if len(transfersWithInterest) != 3 {
		t.Fatalf("len(transfersWithInterest) != 3: %v", transfersWithInterest)
	}
	for i, transferID := range []string{"1", "4", "5"} {
		if transfersWithInterest[i].TransferID != transferID {
			t.Fatalf("transfersWithInterest[%v].TransferID: %v != %v", i, transfersWithInterest[i].TransferID, transferID)
		}
	}
}
