package api4unsorted

import (
	"testing"

	"github.com/strongo/delaying"
)

// TestInitDelaying guards against "orphan" delayer vars: a delayer var that no
// registrar assigns stays nil and panics on the first EnqueueWork in
// production. InitDelaying is wired into debtus.Extension() via
// extension.RegisterDelays.
func TestInitDelaying(t *testing.T) {
	registered := make(map[string]int)
	mustRegisterFunc := func(key string, delayedFunc any) delaying.Delayer {
		registered[key]++
		return delaying.VoidWithLog(key, delayedFunc)
	}

	InitDelaying(mustRegisterFunc)

	for key, count := range registered {
		if count > 1 {
			t.Errorf("delayed func key %q registered %d times", key, count)
		}
	}

	delayers := map[string]delaying.Delayer{
		"delayChangeTransfersCounterparty": delayChangeTransfersCounterparty,
		"delayChangeTransferCounterparty":  delayChangeTransferCounterparty,
	}
	for name, delayer := range delayers {
		if delayer == nil {
			t.Errorf("%s is nil after InitDelaying", name)
		}
	}
}
