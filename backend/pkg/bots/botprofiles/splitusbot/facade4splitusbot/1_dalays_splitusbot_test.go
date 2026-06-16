package facade4splitusbot

import (
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/facade4splitus"
	"github.com/strongo/delaying"
)

// TestInitDelayingFotSplitusBot guards against "orphan" delayer vars: a delayer
// var that no registrar assigns stays nil and panics on the first EnqueueWork
// in production. InitDelayingFotSplitusBot is wired into splitus.Module() via
// extension.RegisterDelays.
func TestInitDelayingFotSplitusBot(t *testing.T) {
	registered := make(map[string]int)
	mustRegisterFunc := func(key string, delayedFunc any) delaying.Delayer {
		registered[key]++
		return delaying.VoidWithLog(key, delayedFunc)
	}

	InitDelayingFotSplitusBot(mustRegisterFunc)

	for key, count := range registered {
		if count > 1 {
			t.Errorf("delayed func key %q registered %d times", key, count)
		}
	}

	delayers := map[string]delaying.Delayer{
		"DelayerUpdateUserWithGroups":                   DelayerUpdateUserWithGroups,
		"facade4splitus.DelayerUpdateGroupUsers":        facade4splitus.DelayerUpdateGroupUsers,
		"facade4splitus.DelayerUpdateContactWithGroups": facade4splitus.DelayerUpdateContactWithGroups,
	}
	for name, delayer := range delayers {
		if delayer == nil {
			t.Errorf("%s is nil after InitDelayingFotSplitusBot", name)
		}
	}
}
