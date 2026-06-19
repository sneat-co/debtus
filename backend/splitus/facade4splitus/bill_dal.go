package facade4splitus

import (
	"context"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/strongo/logus"
)

func SaveBill(ctx context.Context, tx dal.ReadwriteTransaction, bill models4splitus.BillEntry) (err error) {
	if err = tx.Set(ctx, bill.Record); err != nil {
		return
	}
	if err = DelayUpdateUsersWithBill(ctx, bill.ID, bill.Data.UserIDs); err != nil {
		return
	}
	return
}

func delayedUpdateBillDependencies(ctx context.Context, billID string) (err error) {
	logus.Debugf(ctx, "delayerUpdateBillDependencies(billID=%s)", billID)
	var bill models4splitus.BillEntry
	if bill, err = GetBillByID(ctx, nil, billID); err != nil {
		if dal.IsNotFound(err) {
			logus.Warningf(ctx, err.Error())
			err = nil
		}
		return
	}
	if userGroupID := bill.Data.GetUserGroupID(); userGroupID != "" {
		if err = DelayUpdateGroupWithBill(ctx, userGroupID, bill.ID); err != nil {
			return
		}
	}
	for _, member := range bill.Data.GetBillMembers() {
		if member.UserID != "" {
			if err = DelayUpdateSpaceWithBill(ctx, member.UserID, bill.ID); err != nil {
				return
			}
		}
	}
	return
}
