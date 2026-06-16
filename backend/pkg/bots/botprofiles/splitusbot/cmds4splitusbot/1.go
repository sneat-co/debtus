package cmds4splitusbot

import "github.com/strongo/delaying"

func InitDelaying(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayUpdateBillCards = mustRegisterFunc("UpdateBillCards", delayedUpdateBillCards)
	delayUpdateBillTgChatCard = mustRegisterFunc("UpdateBillTgChatCard", delayedUpdateBillTgChartCard)
}

var (
	delayUpdateBillCards      delaying.Delayer
	delayUpdateBillTgChatCard delaying.Delayer
)
