package debtusdal

import (
	"context"
	"fmt"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/decimal"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
)

func (TransferDal) DelayUpdateTransfersOnReturn(ctx context.Context, returnTransferID string, transferReturnsUpdate []dal4debtus.TransferReturnUpdate) (err error) {
	logus.Debugf(ctx, "DelayUpdateTransfersOnReturn(returnTransferID=%v, transferReturnsUpdate=%v)", returnTransferID, transferReturnsUpdate)
	if returnTransferID == "" {
		panic("returnTransferID == 0")
	}
	if len(transferReturnsUpdate) == 0 {
		panic("len(transferReturnsUpdate) == 0")
	}
	for i, transferReturnUpdate := range transferReturnsUpdate {
		if transferReturnUpdate.TransferID == "" {
			panic(fmt.Sprintf("transferReturnsUpdates[%d].TransferID == 0", i))
		}
		if transferReturnUpdate.ReturnedAmount <= 0 {
			panic(fmt.Sprintf("transferReturnsUpdates[%d].Amount <= 0: %v", i, transferReturnUpdate.ReturnedAmount))
		}
	}
	return delayer4debtus.UpdateTransfersOnReturn.EnqueueWork(ctx, delaying.With(const4debtus.QueueTransfers, "update-api4transfers-on-return", 0), returnTransferID, transferReturnsUpdate)
}

func delayedUpdateTransfersOnReturn(ctx context.Context, returnTransferID string, transferReturnsUpdate []dal4debtus.TransferReturnUpdate) (err error) {
	logus.Debugf(ctx, "delayedUpdateTransfersOnReturn(returnTransferID=%v, transferReturnsUpdate=%+v)", returnTransferID, transferReturnsUpdate)
	for i, transferReturnUpdate := range transferReturnsUpdate {
		if transferReturnUpdate.TransferID == "" {
			panic(fmt.Sprintf("transferReturnsUpdates[%d].TransferID == 0", i))
		}
		if transferReturnUpdate.ReturnedAmount <= 0 {
			panic(fmt.Sprintf("transferReturnsUpdates[%d].Amount <= 0: %v", i, transferReturnUpdate.ReturnedAmount))
		}
		if err = DelayUpdateTransferOnReturn(ctx, returnTransferID, transferReturnUpdate.TransferID, transferReturnUpdate.ReturnedAmount); err != nil {
			return
		}
	}
	return
}

func DelayUpdateTransferOnReturn(ctx context.Context, returnTransferID, transferID string, returnedAmount decimal.Decimal64p2) error {
	return delayer4debtus.UpdateTransferOnReturn.EnqueueWork(ctx, delaying.With(const4debtus.QueueTransfers, "update-transfer-on-return", 0), returnTransferID, transferID, returnedAmount)
}

func delayedUpdateTransferOnReturn(ctx context.Context, returnTransferID, transferID string, returnedAmount decimal.Decimal64p2) (err error) {
	logus.Debugf(ctx, "delayedUpdateTransferOnReturn(returnTransferID=%v, transferID=%v, returnedAmount=%v)", returnTransferID, transferID, returnedAmount)

	var transfer, returnTransfer models4debtus.TransferEntry

	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if returnTransfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, returnTransferID); err != nil {
			if dal.IsNotFound(err) {
				logus.Errorf(ctx, fmt.Errorf("return transfer not found: %w", err).Error())
				err = nil
			}
			return
		}

		if transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID); err != nil {
			if dal.IsNotFound(err) {
				logus.Errorf(ctx, err.Error())
				err = nil
			}
			return
		}
		if err = facade4debtus.Transfers.UpdateTransferOnReturn(ctx, tx, returnTransfer, transfer, returnedAmount); err != nil {
			return
		}
		if transfer.Data.HasInterest() && !transfer.Data.IsOutstanding {
			if err = removeFromOutstandingWithInterest(ctx, tx, transfer); err != nil {
				return
			}
		}
		return
	}, dal.TxWithCrossGroup())
}

func removeFromOutstandingWithInterest(ctx context.Context, tx dal.ReadwriteTransaction, transfer models4debtus.TransferEntry) (err error) {
	removeFromOutstanding := func(spaceID coretypes.SpaceID, contactID string) (err error) {
		if spaceID == "" && contactID == "" {
			return
		} else if spaceID == "" {
			panic("removeFromOutstandingWithInterest(): spaceID == 0")
		} else if contactID == "" {
			panic("removeFromOutstandingWithInterest(): contactID == 0")
		}
		removeFromUser := func() (err error) {
			debtusSpace := models4debtus.NewDebtusSpaceEntry(spaceID)

			if err = models4debtus.GetDebtusSpace(ctx, tx, debtusSpace); err != nil {
				return
			}
			for _, debtusContactBrief := range debtusSpace.Data.Contacts {
				for i, outstanding := range debtusContactBrief.Transfers.OutstandingWithInterest {
					if outstanding.TransferID == transfer.ID {
						// https://github.com/golang/go/wiki/SliceTricks
						a := debtusContactBrief.Transfers.OutstandingWithInterest
						debtusContactBrief.Transfers.OutstandingWithInterest = append(a[:i], a[i+1:]...)
						debtusSpace.Data.TransfersWithInterestCount -= 1
						if err = tx.Set(ctx, debtusSpace.Record); err != nil {
							return err
						}
					}
				}
			}
			return
		}
		removeFromContact := func() (err error) {
			var (
				contact models4debtus.DebtusSpaceContactEntry
			)
			if contact, err = facade4debtus.GetDebtusSpaceContactByID(ctx, tx, spaceID, contactID); err != nil {
				return
			}
			transfersInfo := *contact.Data.GetTransfersInfo()
			for i, outstanding := range transfersInfo.OutstandingWithInterest {
				if outstanding.TransferID == transfer.ID {
					// https://github.com/golang/go/wiki/SliceTricks
					a := transfersInfo.OutstandingWithInterest
					transfersInfo.OutstandingWithInterest = append(a[:i], a[i+1:]...)
					if err = contact.Data.SetTransfersInfo(transfersInfo); err != nil {
						return
					}
					return facade4debtus.SaveContact(ctx, contact)
				}
			}
			return
		}
		if err = removeFromUser(); err != nil {
			return
		}
		if err = removeFromContact(); err != nil {
			return
		}
		return
	}
	from, to := transfer.Data.From(), transfer.Data.To()

	if err = removeFromOutstanding(coretypes.SpaceID(from.UserID), to.ContactID); err != nil {
		return
	}
	if err = removeFromOutstanding(coretypes.SpaceID(to.UserID), from.ContactID); err != nil {
		return
	}
	return
}
