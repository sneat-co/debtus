package debtusdal

import (
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/logus"

	//"errors"
	"sync"

	"context"
)

type TransferFixer struct {
	changed     bool
	Fixes       []string
	transferKey *dal.Key
	transfer    *models4debtus.TransferData
}

func NewTransferFixer(transferKey *dal.Key, transfer *models4debtus.TransferData) TransferFixer {
	return TransferFixer{transferKey: transferKey, transfer: transfer, Fixes: make([]string, 0)}
}

func (f *TransferFixer) needFixCounterpartyCounterpartyName() bool {
	return f.transfer.Creator().ContactName == ""
}

func (f *TransferFixer) needFixes(_ context.Context) bool {
	return f.needFixCounterpartyCounterpartyName()
	//logus.Debugf(c, "%v: needFixes=%v", f.transferKey.IntegerID(), result)
	//return result
}

func (f *TransferFixer) FixAllIfNeeded(ctx context.Context) (err error) {
	if f.needFixes(ctx) {
		err = facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) error {
			transfer, err := facade4debtus.Transfers.GetTransferByID(tctx, tx, f.transferKey.ID.(string))
			if err != nil {
				return err
			}
			f.transfer = transfer.Data
			//if err = f.fixCounterpartyCounterpartyName(ctx); err != nil {
			//	return err
			//}
			if f.changed {
				//logus.Debugf(ctx, "%v: changed", f.transferKey.IntegerID())
				err = tx.Set(tctx, transfer.Record)
				return err
				//} else {
				//	logus.Debugf(ctx, "%v: not changed", f.transferKey.IntegerID())
			}
			return nil
		}, nil)
	}
	return
}

func FixTransfers(ctx context.Context) (loadedCount int, fixedCount int, failedCount int, err error) {
	query := dal.From(models4debtus.TransfersCollectionRef).NewQuery().SelectIntoRecord(func() dal.Record {
		return models4debtus.NewTransferWithIncompleteKey(nil).Record
	})
	//query.Limit = 50
	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return
	}
	var reader dal.RecordsReader
	reader, err = db.ExecuteQueryToRecordsReader(ctx, query)
	if err != nil {
		return
	}
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}
	for {
		var record dal.Record
		if record, err = reader.Next(); err != nil {
			if err == dal.ErrNoMoreRecords {
				err = nil
				return
			}
			logus.Errorf(ctx, "Failed to get next transfer: %v", err.Error())
			return
		}
		loadedCount += 1
		wg.Add(1)
		go func(transferRecord dal.Record) {
			defer wg.Done()
			key := transferRecord.Key()
			fixer := NewTransferFixer(key, transferRecord.Data().(*models4debtus.TransferData))
			err2 := fixer.FixAllIfNeeded(ctx)
			if err2 != nil {
				logus.Errorf(ctx, "Failed to fix transfer=%v: %v", key.ID.(int), err2.Error())
				mutex.Lock()
				failedCount += 1
				err = err2
				mutex.Unlock()
			} else {
				if len(fixer.Fixes) > 0 {
					mutex.Lock()
					fixedCount += 1
					mutex.Unlock()
					logus.Infof(ctx, "Fixed transfer %v: %v", key.ID.(int), fixer.Fixes)
					//} else {
					//	logus.Debugf(ctx, "TransferEntry %v is OK: CounterpartyCounterpartyName: %v", transferKey.IntegerID(), fixer.transfer.Creator().ContactName)
				}
			}
		}(record)
		if err != nil {
			break
		}
	}
	wg.Wait()
	return
}
