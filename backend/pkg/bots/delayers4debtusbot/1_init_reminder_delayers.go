package delayers4debtusbot

import (
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/debtusdal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/delayer4debtus"
	"github.com/strongo/delaying"
)

func InitReminderDelayers(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayer4debtus.SetReminderIsSent = mustRegisterFunc("SetReminderIsSent", debtusdal.DelayedSetReminderIsSent)
}
