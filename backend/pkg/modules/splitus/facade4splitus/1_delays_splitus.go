package facade4splitus

import (
	"context"

	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/const4splitus"
	"github.com/strongo/delaying"
)

var DelayerUpdateGroupUsers delaying.Delayer
var DelayerUpdateContactWithGroups delaying.Delayer

// ------------------------------------------------------------
var delayerUpdateGroupWithBill delaying.Delayer

func DelayUpdateGroupWithBill(ctx context.Context, groupID, billID string) (err error) {
	if err = delayerUpdateGroupWithBill.EnqueueWork(ctx, delaying.With(const4splitus.QueueSplitus, "UpdateGroupWithBill", 0), groupID, billID); err != nil {
		return
	}
	return
}

// ------------------------------------------------------------
var delayerUpdateBillDependencies delaying.Delayer

func DelayUpdateBillDependencies(ctx context.Context, billID string) (err error) {
	if err = delayerUpdateBillDependencies.EnqueueWork(ctx, delaying.With(const4splitus.QueueSplitus, "UpdateBillDependencies", 0), billID); err != nil {
		return
	}
	return
}

var delayerUpdateUsersWithBill delaying.Delayer
var delayerUpdateUserWithBill delaying.Delayer

//------------------------------------------------------------

func InitDelaying(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayerUpdateBillDependencies = mustRegisterFunc("delayerUpdateBillDependencies", delayedUpdateBillDependencies)
	delayerUpdateUsersWithBill = mustRegisterFunc(updateUsersWithBillKeyName, delayedUpdateUsersWithBill)
	delayerUpdateGroupWithBill = mustRegisterFunc("delayedUpdateWithBill", delayedUpdateGroupWithBill)
	delayerUpdateUserWithBill = mustRegisterFunc("delayedUpdateUserWithBill", delayedUpdateUserWithBill)
}
