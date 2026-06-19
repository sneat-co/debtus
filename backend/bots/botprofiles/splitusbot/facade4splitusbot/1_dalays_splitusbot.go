package facade4splitusbot

import (
	"github.com/sneat-co/debtus/backend/splitus/facade4splitus"
	"github.com/strongo/delaying"
)

func InitDelayingFotSplitusBot(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	DelayerUpdateUserWithGroups = mustRegisterFunc("delayedUpdateUserWithGroups", delayedUpdateUserWithGroups)
	facade4splitus.DelayerUpdateGroupUsers = mustRegisterFunc("delayedUpdateGroupUsers", delayedUpdateGroupUsers)
	facade4splitus.DelayerUpdateContactWithGroups = mustRegisterFunc("delayedUpdateContactWithGroup", delayedUpdateContactWithGroup)
}
