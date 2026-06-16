package delayers4debtusbot

import (
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/delayed4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/delayer4debtus"
	"github.com/strongo/delaying"
)

func InitDelayers(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	// TODO(help-wanted): Check if we can register unexported functions?
	delayer4debtus.OnReceiptSentSuccess = mustRegisterFunc("OnReceiptSentSuccess", delayed4debtus.DelayedOnReceiptSentSuccess)
	delayer4debtus.OnReceiptSendFail = mustRegisterFunc("OnReceiptSendFail", delayed4debtus.DelayedOnReceiptSendFail)
	delayer4debtus.CreateReminderForTransferUser = mustRegisterFunc("CreateReminderForTransferUser", delayed4debtus.DelayedCreateReminderForTransferUser)
	delayer4debtus.SendReceiptToCounterpartyByTelegram = mustRegisterFunc("SendReceiptToCounterpartyByTelegram", delayed4debtus.DelayedSendReceiptToCounterpartyByTelegram)
	delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID = mustRegisterFunc("UpdateTransferWithCreatorReceiptTgMessageID", delayed4debtus.DelayedUpdateTransferWithCreatorReceiptTgMessageID)
	//
	delayer4debtus.DiscardReminderForTransfer = mustRegisterFunc("DiscardReminderForTransfer", delayed4debtus.DelayedDiscardReminderForTransfer)
	delayer4debtus.DiscardRemindersForTransfer = mustRegisterFunc("DiscardRemindersForTransfer", delayed4debtus.DelayedDiscardRemindersForTransfer)
	//
	delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram = mustRegisterFunc("CreateAndSendReceiptToCounterpartyByTelegram", delayed4debtus.DelayedCreateAndSendReceiptToCounterpartyByTelegram)
	delayer4debtus.DiscardRemindersForTransfers = mustRegisterFunc("DiscardRemindersForTransfers", delayed4debtus.DelayedDiscardRemindersForTransfers)

	InitReminderDelayers(mustRegisterFunc)
}
