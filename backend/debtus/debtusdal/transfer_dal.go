package debtusdal

import (
	"bytes"
	"fmt"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"

	"context"
	"errors"
)

type TransferDal struct {
}

func NewTransferDal() TransferDal {
	return TransferDal{}
}

var _ dal4debtus.TransferDal = (*TransferDal)(nil)

func _loadDueOnTransfers(ctx context.Context, tx dal.ReadSession, userID string, limit int, filter func(q dal.IQueryBuilder) dal.IQueryBuilder) (transfers []models4debtus.TransferEntry, err error) {
	qb := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BothUserIDs", userID).
		WhereField("isOutstanding", dal.Equal, true).OrderBy(dal.AscendingField("dtDueOn"))
	qb = filter(qb).Limit(limit)
	query := qb.SelectIntoRecord(models4debtus.NewTransferRecord)
	return models4debtus.TransfersFromQuery(ctx, query, tx)
}

func (transferDal TransferDal) LoadOverdueTransfers(ctx context.Context, tx dal.ReadSession, userID string, limit int) ([]models4debtus.TransferEntry, error) {
	return _loadDueOnTransfers(ctx, tx, userID, limit, func(q dal.IQueryBuilder) dal.IQueryBuilder {
		return q.WhereField("dtDueOn", dal.GreaterThen, time.Time{}).WhereField("dtDueOn", dal.LessThen, time.Now())
	})
}

func (transferDal TransferDal) LoadDueTransfers(ctx context.Context, tx dal.ReadSession, userID string, limit int) ([]models4debtus.TransferEntry, error) {
	return _loadDueOnTransfers(ctx, tx, userID, limit, func(q dal.IQueryBuilder) dal.IQueryBuilder {
		return q.WhereField("dtDueOn", dal.GreaterThen, time.Now())
	})
}

func (transferDal TransferDal) GetTransfersByID(ctx context.Context, tx dal.ReadSession, transferIDs []string) (transfers []models4debtus.TransferEntry, err error) {
	transfers = make([]models4debtus.TransferEntry, len(transferIDs))
	records := make([]dal.Record, len(transferIDs))
	for i, transferID := range transferIDs {
		transfers[i] = models4debtus.NewTransfer(transferID, nil)
		records[i] = transfers[i].Record
	}
	if err = tx.GetMulti(ctx, records); err != nil {
		return
	}
	return
}

func (transferDal TransferDal) LoadOutstandingTransfers(ctx context.Context, tx dal.ReadSession, periodEnds time.Time, userID, contactID string, currency money.CurrencyCode, direction models4debtus.TransferDirection) (transfers []models4debtus.TransferEntry, err error) {
	logus.Debugf(ctx, "TransferDal.LoadOutstandingTransfers(periodEnds=%v, userID=%v, contactID=%v currency=%v, direction=%v)", periodEnds, userID, contactID, currency, direction)
	const limit = 100

	// TODO: Load outstanding transfer just for the specific contact & specific direction
	q := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BothUserIDs", userID).
		WhereField("currency", dal.Equal, string(currency)).
		WhereField("isOutstanding", dal.Equal, true).
		OrderBy(dal.AscendingField("DtCreated")).
		Limit(limit).
		SelectIntoRecord(models4debtus.NewTransferRecord)

	if transfers, err = models4debtus.TransfersFromQuery(ctx, q, tx); err != nil {
		return nil, err
	}

	var errorMessages, warnings, debugs bytes.Buffer
	var transfersIDsToFixIsOutstanding []string
	for _, transfer := range transfers {
		if contactID != "" {
			if cpContactID := transfer.Data.CounterpartyInfoByUserID(userID).ContactID; cpContactID != contactID {
				fmt.Fprintf(&debugs, "Skipped outstanding TransferEntry(id=%v) as counterpartyContactID != contactID: %v != %v\n", transfer.ID, cpContactID, contactID)
				continue
			}
		}
		if direction != "" {
			if d := transfer.Data.DirectionForUser(userID); d != direction {
				fmt.Fprintf(&debugs, "Skipped outstanding TransferEntry(id=%v) as DirectionForUser(): %v\n", transfer.ID, d)
				continue
			}
		}

		if outstandingValue := transfer.Data.GetOutstandingValue(periodEnds); outstandingValue > 0 {
			transfers = append(transfers, transfer)
		} else if outstandingValue == 0 {
			_, _ = fmt.Fprintf(&warnings, "TransferEntry(id=%v) => GetOutstandingValue() == 0 && IsOutstanding==true\n", transfer.ID)
			transfersIDsToFixIsOutstanding = append(transfersIDsToFixIsOutstanding, transfer.ID)
		} else { // outstandingValue < 0
			_, _ = fmt.Fprintf(&errorMessages, "TransferEntry(id=%v) => IsOutstanding==true && GetOutstandingValue() < 0: %v\n", transfer.ID, outstandingValue)
		}
	}
	if len(transfersIDsToFixIsOutstanding) > 0 {
		if err = delayer4debtus.FixTransfersIsOutstanding.EnqueueWork(ctx, delaying.With(const4debtus.QueueTransfers, "fix-api4transfers-is-outstanding", 0), transfersIDsToFixIsOutstanding); err != nil {
			logus.Errorf(ctx, "failed to delay task to fix api4transfers IsOutstanding")
			err = nil
		}
	}
	if errorMessages.Len() > 0 {
		logus.Errorf(ctx, errorMessages.String())
	}
	if warnings.Len() > 0 {
		logus.Warningf(ctx, warnings.String())
	}
	if debugs.Len() > 0 {
		logus.Debugf(ctx, debugs.String())
	}
	return
}

func delayedFixTransfersIsOutstanding(ctx context.Context, transferIDs []string) (err error) {
	logus.Debugf(ctx, "delayedFixTransfersIsOutstanding(%v)", transferIDs)
	for _, transferID := range transferIDs {
		if _, transferErr := fixTransferIsOutstanding(ctx, transferID); transferErr != nil {
			logus.Errorf(ctx, "Failed to fix transfer %v: %v", transferID, err)
			err = transferErr
		}
	}
	return
}

func fixTransferIsOutstanding(ctx context.Context, transferID string) (transfer models4debtus.TransferEntry, err error) {
	err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		if transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID); err != nil {
			return err
		}
		if transfer.Data.GetOutstandingValue(time.Now()) == 0 {
			transfer.Data.IsOutstanding = false
			return facade4debtus.Transfers.SaveTransfer(ctx, tx, transfer)
		}
		return nil
	})
	if err == nil {
		logus.Warningf(ctx, "Fixed IsOutstanding (set to false) for transfer %v", transferID)
	} else {
		logus.Errorf(ctx, "Failed to fix IsOutstanding for transfer %v", transferID)
	}
	return
}

func (transferDal TransferDal) LoadTransfersByUserID(ctx context.Context, userID string, offset, limit int) (transfers []models4debtus.TransferEntry, hasMore bool, err error) {
	if limit == 0 {
		err = errors.New("limit == 0")
		return
	}
	if userID == "" {
		err = errors.New("userID == 0")
		return
	}
	q := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BothUserIDs", userID).
		OrderBy(dal.DescendingField("DtCreated")).
		SelectIntoRecord(models4debtus.NewTransferRecord)

	if transfers, err = transferDal.loadTransfers(ctx, q); err != nil {
		return
	}
	hasMore = len(transfers) > limit
	return
}

func (transferDal TransferDal) LoadTransferIDsByContactID(ctx context.Context, contactID string, limit int, startCursor string) (transferIDs []string, endCursor string, err error) {
	if limit == 0 {
		err = errors.New("LoadTransferIDsByContactID(): limit == 0")
		return
	} else if limit > 1000 {
		err = errors.New("LoadTransferIDsByContactID(): limit > 1000")
		return
	}
	if contactID == "" {
		err = errors.New("LoadTransferIDsByContactID(): contactID == 0")
		return
	}
	q := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BothCounterpartyIDs", contactID).
		Limit(limit).
		StartFrom(dal.Cursor(startCursor)).
		SelectIntoRecord(models4debtus.NewTransferRecord)

	//if startCursor != "" {
	//	var decodedCursor datastore.Cursor
	//	if decodedCursor, err = datastore.DecodeCursor(startCursor); err != nil {
	//		return
	//	} else {
	//		q = q.Start(decodedCursor)
	//	}
	//}

	transferIDs = make([]string, 0, limit)
	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return
	}
	var reader dal.RecordsReader
	if reader, err = db.ExecuteQueryToRecordsReader(ctx, q); err != nil {
		return
	}
	var record dal.Record
	for {
		if record, err = reader.Next(); err != nil {
			if errors.Is(err, dal.ErrNoMoreRecords) {
				endCursor, err = reader.Cursor()
			}
			return
		}
		transferIDs = append(transferIDs, record.Key().ID.(string))
	}
}

func (transferDal TransferDal) LoadTransfersByContactID(ctx context.Context, contactID string, offset, limit int) (transfers []models4debtus.TransferEntry, hasMore bool, err error) {
	if limit == 0 {
		err = errors.New("LoadTransfersByContactID(): limit == 0")
		return
	}
	if contactID == "" {
		err = errors.New("LoadTransfersByContactID(): contactID == 0")
		return
	}
	q := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BothCounterpartyIDs", contactID).
		OrderBy(dal.DescendingField("DtCreated")).
		Limit(limit).
		Offset(offset).
		SelectIntoRecord(models4debtus.NewTransferRecord)

	if transfers, err = transferDal.loadTransfers(ctx, q); err != nil {
		return
	}
	hasMore = len(transfers) > limit
	return
}

func (transferDal TransferDal) LoadLatestTransfers(ctx context.Context, offset, limit int) ([]models4debtus.TransferEntry, error) {
	q := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		OrderBy(dal.DescendingField("DtCreated")).
		Limit(limit).
		Offset(offset).
		SelectIntoRecord(models4debtus.NewTransferRecord)
	return transferDal.loadTransfers(ctx, q)
}

func (transferDal TransferDal) loadTransfers(ctx context.Context, q dal.Query) (transfers []models4debtus.TransferEntry, err error) {
	return models4debtus.TransfersFromQuery(ctx, q, nil)
}

func (TransferDal) DelayUpdateTransferWithCreatorReceiptTgMessageID(ctx context.Context, botCode string, transferID string, creatorTgChatID, creatorTgReceiptMessageID int64) error {
	// logus.Debugf(ctx, "delayerUpdateTransferWithCreatorReceiptTgMessageID(botCode=%v, transferID=%v, creatorTgChatID=%v, creatorTgReceiptMessageID=%v)", botCode, transferID, creatorTgChatID, creatorTgReceiptMessageID)

	if err := delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID.EnqueueWork(
		ctx, delaying.With(const4debtus.QueueTransfers, "update-transfer-with-creator-receipt-tg-message-id", 0),
		botCode, transferID, creatorTgChatID, creatorTgReceiptMessageID); err != nil {
		return fmt.Errorf("failed to create delayed task update-transfer-with-creator-receipt-tg-message-id: %w", err)
	}
	return nil
}
