package delayer4debtus

import "github.com/strongo/delaying"

var (
	DiscardReminderForTransfer delaying.Delayer
	SetReminderIsSent          delaying.Delayer
)
