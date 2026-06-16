package debtusdal

import (
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/delayer4debtus"
	"github.com/strongo/delaying"
)

// RegisterDelayers4Debtus registers the delayed funcs implemented in this
// package. Wired into the extension via debtus.Extension() so the
// delayer4debtus vars are assigned before any EnqueueWork call.
func RegisterDelayers4Debtus(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayer4debtus.DeleteContactTransfersDelayFunc = mustRegisterFunc(DeleteContactTransfersFuncKey, delayedDeleteContactTransfers)
	delayer4debtus.UpdateInviteClaimedCount = mustRegisterFunc("UpdateInviteClaimedCount", delayedUpdateInviteClaimedCount)
	delayer4debtus.MarkReceiptAsSent = mustRegisterFunc("MarkReceiptAsSent", delayedMarkReceiptAsSent)
	delayer4debtus.FixTransfersIsOutstanding = mustRegisterFunc("FixTransfersIsOutstanding", delayedFixTransfersIsOutstanding)
	delayer4debtus.UpdateTransferOnReturn = mustRegisterFunc("UpdateTransferOnReturn", delayedUpdateTransferOnReturn)
	delayer4debtus.UpdateTransfersWithCounterparty = mustRegisterFunc("UpdateTransfersWithCounterparty", delayedUpdateTransfersWithCounterparty)
	delayer4debtus.UpdateTransferWithCounterparty = mustRegisterFunc("UpdateTransferWithCounterparty", delayedUpdateTransferWithCounterparty)
	delayer4debtus.UpdateTransfersWithCreatorName = mustRegisterFunc("UpdateTransfersWithCreatorName", delayedUpdateTransfersWithCreatorName)
	delayer4debtus.UpdateTransfersOnReturn = mustRegisterFunc("UpdateTransfersOnReturn", delayedUpdateTransfersOnReturn)
}
