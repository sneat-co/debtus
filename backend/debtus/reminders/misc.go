package reminders

type TransferReminderTo int

const (
	TransferReminderToCreator TransferReminderTo = iota
	TransferReminderToCounterparty
)
