package debtusdal

import (
	"context"
	"slices"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
)

type ReceiptDal struct {
}

func NewReceiptDal() ReceiptDal {
	return ReceiptDal{}
}

var _ dal4debtus.ReceiptDal = (*ReceiptDal)(nil)

var receiptCollection = dal.CollectionAt[string, models4debtus.ReceiptDbo](models4debtus.ReceiptKind)

func (ReceiptDal) DelayCreateAndSendReceiptToCounterpartyByTelegram(ctx context.Context, env string, transferID string, userID string) error {
	logus.Debugf(ctx, "delayerSendReceiptToCounterpartyByTelegram(env=%v, transferID=%v, userID=%v)", env, transferID, userID)
	return delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram.EnqueueWork(ctx, delaying.With(const4debtus.QueueReceipts, "create-and-send-receipt-for-counterparty-by-telegram", 0), env, transferID, userID)
}

func (ReceiptDal) UpdateReceipt(ctx context.Context, tx dal.ReadwriteTransaction, receipt models4debtus.ReceiptEntry) error {
	return tx.Set(ctx, receipt.Record)
}

func (receiptDal ReceiptDal) GetReceiptByID(ctx context.Context, tx dal.ReadSession, id string) (receipt models4debtus.ReceiptEntry, err error) {
	receipt = models4debtus.NewReceipt(id, nil)
	return receipt, tx.Get(ctx, receipt.Record)
}

func (receiptDal ReceiptDal) CreateReceipt(ctx context.Context, data *models4debtus.ReceiptDbo) (receipt models4debtus.ReceiptEntry, err error) { // TODO: Move to facade4debtus
	err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		debtusUser := models4debtus.NewDebtusUserEntry(data.CreatorUserID)
		if err = dal4debtus.GetDebtusUser(ctx, tx, debtusUser); err != nil {
			return err
		}
		debtusUser.Data.CountOfReceiptsCreated += 1
		if err = tx.Set(ctx, debtusUser.Record); err != nil {
			return err
		}
		var key *dal.Key
		// R5 ID policy: adapter-generated IDs (Collection.Insert defaults to
		// client-side WithRandomStringKey otherwise).
		if key, err = receiptCollection.Insert(ctx, tx, *data, dal.WithAdapterGeneratedID()); err != nil {
			return err
		}
		receipt = models4debtus.NewReceipt(key.ID.(string), data)
		return
	})
	return
}

func (receiptDal ReceiptDal) MarkReceiptAsSent(ctx context.Context, receiptID, transferID string, sentTime time.Time) error {
	return facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		var receipt models4debtus.ReceiptEntry
		if receipt, err = receiptDal.GetReceiptByID(ctx, tx, receiptID); err != nil {
			return err
		}
		if transferID == "" {
			transferID = receipt.Data.TransferID
		}
		transfer := models4debtus.NewTransfer(transferID, nil)
		if err = tx.Get(ctx, transfer.Record); err != nil {
			return err
		}
		if !receipt.Data.DtSent.IsZero() {
			return nil
		}
		receipt.Data.DtSent = sentTime
		if slices.Contains(transfer.Data.ReceiptIDs, receiptID) {
			return tx.Set(ctx, receipt.Record)
		}
		transfer.Data.ReceiptIDs = append(transfer.Data.ReceiptIDs, receiptID)
		transfer.Data.ReceiptsSentCount += 1
		return tx.SetMulti(ctx, []dal.Record{receipt.Record, transfer.Record})
	})
}

func (receiptDal ReceiptDal) DelayedMarkReceiptAsSent(ctx context.Context, receiptID, transferID string, sentTime time.Time) error {
	return delayer4debtus.MarkReceiptAsSent.EnqueueWork(ctx, delaying.With(const4debtus.QueueTransfers, "set-receipt-as-sent", 0), receiptID, transferID, sentTime)
}

func delayedMarkReceiptAsSent(ctx context.Context, receiptID, transferID string, sentTime time.Time) (err error) {
	logus.Debugf(ctx, "MarkReceiptAsSent(receiptID=%v, transferID=%v, sentTime=%v)", receiptID, transferID, sentTime)
	if receiptID == "" {
		logus.Errorf(ctx, "receiptID is empty")
		return nil
	}
	if transferID == "" {
		logus.Errorf(ctx, "transferID is empty")
		return nil
	}

	if err = dal4debtus.Default.Receipt.MarkReceiptAsSent(ctx, receiptID, transferID, sentTime); dal.IsNotFound(err) {
		logus.Errorf(ctx, err.Error())
		return nil
	}
	return
}
