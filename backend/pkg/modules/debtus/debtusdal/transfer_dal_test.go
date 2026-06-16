package debtusdal

import (
	"testing"

	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

func TestNewTransferKey(t *testing.T) {
	const transferID = "12345"
	testStrKey(t, transferID, models4debtus.NewTransferKey(transferID))
}
