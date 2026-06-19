package debtusdal

import (
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

func TestNewReceiptIncompleteKey(t *testing.T) {
	testIncompleteKey(t, models4debtus.NewReceiptIncompleteKey())
}

func TestNewReceiptKey(t *testing.T) {
	const receiptID = "234"
	testStrKey(t, receiptID, models4debtus.NewReceiptKey(receiptID))
}
