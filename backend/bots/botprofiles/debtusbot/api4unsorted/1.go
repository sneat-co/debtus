package api4unsorted

import (
	"github.com/strongo/delaying"
)

func InitDelaying(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayChangeTransfersCounterparty = mustRegisterFunc("changeTransfersCounterparty", DelayedChangeTransfersCounterparty)
	delayChangeTransferCounterparty = mustRegisterFunc("changeTransferCounterparty", DelayedChangeTransferCounterparty)
}

var (
	delayChangeTransfersCounterparty,
	delayChangeTransferCounterparty delaying.Delayer
)
