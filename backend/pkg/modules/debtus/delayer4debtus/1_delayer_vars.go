package delayer4debtus

import "github.com/strongo/delaying"

var (
	FixTransfersIsOutstanding                    delaying.Delayer
	CreateAndSendReceiptToCounterpartyByTelegram delaying.Delayer
	DeleteContactTransfersDelayFunc              delaying.Delayer
	DiscardRemindersForTransfer                  delaying.Delayer
	UpdateTransferWithCreatorReceiptTgMessageID  delaying.Delayer
	SendReceiptToCounterpartyByTelegram          delaying.Delayer
	UpdateInviteClaimedCount                     delaying.Delayer
	UpdateTransfersWithCounterparty              delaying.Delayer
	UpdateTransferWithCounterparty               delaying.Delayer
	MarkReceiptAsSent                            delaying.Delayer
	UpdateTransfersWithCreatorName               delaying.Delayer
	UpdateTransfersOnReturn                      delaying.Delayer
	UpdateTransferOnReturn                       delaying.Delayer
	OnReceiptSendFail                            delaying.Delayer
	CreateReminderForTransferUser                delaying.Delayer
	OnReceiptSentSuccess                         delaying.Delayer
	DiscardRemindersForTransfers                 delaying.Delayer
)
