package debtusdal

import (
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

func TestNewTransferKey(t *testing.T) {
	const transferID = "12345"
	testStrKey(t, transferID, models4debtus.NewTransferKey(transferID))
}
