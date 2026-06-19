package reminders

import (
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/strongo/delaying"
)

func InitDelaying(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delaySetChatIsForbidden = mustRegisterFunc("SetChatIsForbidden", SetChatIsForbidden)
	// The delayed func lives here (it needs sendReminder), but the enqueueing
	// side lives in debtusdal — assign the delayer var there.
	debtusdal.DelayerSendReminder = mustRegisterFunc("sendReminder", sendReminder)
}

var (
	delaySetChatIsForbidden delaying.Delayer
)
