package debtusdal

import (
	"testing"

	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

func TestNewReceiptIncompleteKey(t *testing.T) {
	testIncompleteKey(t, models4debtus.NewReceiptIncompleteKey())
}

func TestNewReceiptKey(t *testing.T) {
	const receiptID = "234"
	testStrKey(t, receiptID, models4debtus.NewReceiptKey(receiptID))
}
