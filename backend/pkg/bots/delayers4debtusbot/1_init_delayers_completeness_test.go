package delayers4debtusbot

import (
	"testing"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/delayer4debtus"
	"github.com/strongo/delaying"
)

// TestAllDelayerVarsAreRegistered guards against "orphan" delayer vars: a
// delayer4debtus var that no registrar assigns stays nil and panics on the
// first EnqueueWork in production. Both registrars are wired into
// debtus.Extension() via extension.RegisterDelays.
func TestAllDelayerVarsAreRegistered(t *testing.T) {
	registered := make(map[string]int)
	mustRegisterFunc := func(key string, delayedFunc any) delaying.Delayer {
		registered[key]++
		return delaying.VoidWithLog(key, delayedFunc)
	}

	debtusdal.RegisterDelayers4Debtus(mustRegisterFunc)
	InitDelayers(mustRegisterFunc)

	for key, count := range registered {
		if count > 1 {
			t.Errorf("delayed func key %q registered %d times", key, count)
		}
	}

	delayers := map[string]delaying.Delayer{
		"FixTransfersIsOutstanding":                    delayer4debtus.FixTransfersIsOutstanding,
		"CreateAndSendReceiptToCounterpartyByTelegram": delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram,
		"DeleteContactTransfersDelayFunc":              delayer4debtus.DeleteContactTransfersDelayFunc,
		"DiscardRemindersForTransfer":                  delayer4debtus.DiscardRemindersForTransfer,
		"UpdateTransferWithCreatorReceiptTgMessageID":  delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID,
		"SendReceiptToCounterpartyByTelegram":          delayer4debtus.SendReceiptToCounterpartyByTelegram,
		"UpdateInviteClaimedCount":                     delayer4debtus.UpdateInviteClaimedCount,
		"UpdateTransfersWithCounterparty":              delayer4debtus.UpdateTransfersWithCounterparty,
		"UpdateTransferWithCounterparty":               delayer4debtus.UpdateTransferWithCounterparty,
		"MarkReceiptAsSent":                            delayer4debtus.MarkReceiptAsSent,
		"UpdateTransfersWithCreatorName":               delayer4debtus.UpdateTransfersWithCreatorName,
		"UpdateTransfersOnReturn":                      delayer4debtus.UpdateTransfersOnReturn,
		"UpdateTransferOnReturn":                       delayer4debtus.UpdateTransferOnReturn,
		"OnReceiptSendFail":                            delayer4debtus.OnReceiptSendFail,
		"CreateReminderForTransferUser":                delayer4debtus.CreateReminderForTransferUser,
		"OnReceiptSentSuccess":                         delayer4debtus.OnReceiptSentSuccess,
		"DiscardRemindersForTransfers":                 delayer4debtus.DiscardRemindersForTransfers,
		"DiscardReminderForTransfer":                   delayer4debtus.DiscardReminderForTransfer,
		"SetReminderIsSent":                            delayer4debtus.SetReminderIsSent,
	}
	for name, delayer := range delayers {
		if delayer == nil {
			t.Errorf("delayer4debtus.%s is nil after registering all delayers", name)
		}
	}
}
