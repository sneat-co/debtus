package debtusdal

import (
	"reflect"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/strongo/delaying"
)

func TestRegisterDelayers4Debtus(t *testing.T) {
	registered := make(map[string]any)
	mustRegisterFunc := func(key string, delayedFunc any) delaying.Delayer {
		if _, duplicate := registered[key]; duplicate {
			t.Errorf("duplicate registration of delayed func key=%q", key)
		}
		registered[key] = delayedFunc
		return delaying.VoidWithLog(key, delayedFunc)
	}

	RegisterDelayers4Debtus(mustRegisterFunc)

	expected := map[string]any{
		DeleteContactTransfersFuncKey:     delayedDeleteContactTransfers,
		"UpdateInviteClaimedCount":        delayedUpdateInviteClaimedCount,
		"MarkReceiptAsSent":               delayedMarkReceiptAsSent,
		"FixTransfersIsOutstanding":       delayedFixTransfersIsOutstanding,
		"UpdateTransferOnReturn":          delayedUpdateTransferOnReturn,
		"UpdateTransfersOnReturn":         delayedUpdateTransfersOnReturn,
		"UpdateTransfersWithCounterparty": delayedUpdateTransfersWithCounterparty,
		"UpdateTransferWithCounterparty":  delayedUpdateTransferWithCounterparty,
		"UpdateTransfersWithCreatorName":  delayedUpdateTransfersWithCreatorName,
	}
	for key, expectedFunc := range expected {
		actualFunc, ok := registered[key]
		if !ok {
			t.Errorf("delayed func not registered: key=%q", key)
			continue
		}
		if reflect.ValueOf(actualFunc).Pointer() != reflect.ValueOf(expectedFunc).Pointer() {
			t.Errorf("delayed func key=%q registered with unexpected handler", key)
		}
	}
	for key := range registered {
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected delayed func registered: key=%q", key)
		}
	}

	delayers := map[string]delaying.Delayer{
		DeleteContactTransfersFuncKey:     delayer4debtus.DeleteContactTransfersDelayFunc,
		"UpdateInviteClaimedCount":        delayer4debtus.UpdateInviteClaimedCount,
		"MarkReceiptAsSent":               delayer4debtus.MarkReceiptAsSent,
		"FixTransfersIsOutstanding":       delayer4debtus.FixTransfersIsOutstanding,
		"UpdateTransferOnReturn":          delayer4debtus.UpdateTransferOnReturn,
		"UpdateTransfersOnReturn":         delayer4debtus.UpdateTransfersOnReturn,
		"UpdateTransfersWithCounterparty": delayer4debtus.UpdateTransfersWithCounterparty,
		"UpdateTransferWithCounterparty":  delayer4debtus.UpdateTransferWithCounterparty,
		"UpdateTransfersWithCreatorName":  delayer4debtus.UpdateTransfersWithCreatorName,
	}
	for expectedKey, delayer := range delayers {
		if delayer == nil {
			t.Errorf("delayer var not assigned for key=%q", expectedKey)
			continue
		}
		if delayer.ID() != expectedKey {
			t.Errorf("delayer var for key=%q assigned delayer with ID=%q", expectedKey, delayer.ID())
		}
	}
}
