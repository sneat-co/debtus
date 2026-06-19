package debtusdal

import (
	"context"
	"errors"

	"github.com/dal-go/dalgo/dal"
	"github.com/strongo/delaying"
)

// failingDelayer is a delaying.Delayer whose EnqueueWork always returns
// errInjected, used to cover the error branches after EnqueueWork calls.
type failingDelayer struct{ id string }

func (d failingDelayer) ID() string          { return d.id }
func (d failingDelayer) Implementation() any { return nil }
func (d failingDelayer) EnqueueWork(context.Context, delaying.Params, ...any) error {
	return errInjected
}
func (d failingDelayer) EnqueueWorkMulti(context.Context, delaying.Params, ...[]any) error {
	return errInjected
}

// errInjected is the sentinel error returned by the failing-DB / failing-tx
// wrappers below. Tests assert on it to confirm the production error branch
// propagated the underlying DB error.
var errInjected = errors.New("injected DB error")

// txFault selects which transactional operation the failing tx should fail on.
type txFault int

const (
	faultNone txFault = iota
	faultGet
	faultSet
	faultDelete
	faultDeleteMulti
	faultGetMulti
	faultInsert
	faultQuery
)

// failingDB embeds a real dal.DB (the dalgo2memory adapter) so all methods are
// inherited, and overrides only the transaction coordinators to run the worker
// with a failing tx. This lets tests cover error branches that are reachable
// only when a DB write inside a transaction fails — without re-implementing the
// whole dal.DB surface.
type failingDB struct {
	dal.DB
	fault txFault
	// failCoordinator, when set, makes RunReadwriteTransaction return errInjected
	// before invoking the worker at all (covers the outer "begin tx failed" path).
	failCoordinator bool
}

func (d failingDB) RunReadwriteTransaction(ctx context.Context, f dal.RWTxWorker, options ...dal.TransactionOption) error {
	if d.failCoordinator {
		return errInjected
	}
	return d.DB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return f(ctx, failingTx{ReadwriteTransaction: tx, fault: d.fault})
	}, options...)
}

func (d failingDB) ExecuteQueryToRecordsReader(ctx context.Context, query dal.Query) (dal.RecordsReader, error) {
	if d.fault == faultQuery {
		return nil, errInjected
	}
	return d.DB.ExecuteQueryToRecordsReader(ctx, query)
}

func (d failingDB) RunReadonlyTransaction(ctx context.Context, f dal.ROTxWorker, options ...dal.TransactionOption) error {
	return d.DB.RunReadonlyTransaction(ctx, func(ctx context.Context, tx dal.ReadTransaction) error {
		return f(ctx, failingROTx{ReadTransaction: tx, fault: d.fault})
	}, options...)
}

// failingTx embeds a real ReadwriteTransaction and fails the selected op.
type failingTx struct {
	dal.ReadwriteTransaction
	fault txFault
}

func (tx failingTx) Get(ctx context.Context, record dal.Record) error {
	if tx.fault == faultGet {
		return errInjected
	}
	return tx.ReadwriteTransaction.Get(ctx, record)
}

func (tx failingTx) GetMulti(ctx context.Context, records []dal.Record) error {
	if tx.fault == faultGetMulti {
		return errInjected
	}
	return tx.ReadwriteTransaction.GetMulti(ctx, records)
}

func (tx failingTx) Set(ctx context.Context, record dal.Record) error {
	if tx.fault == faultSet {
		return errInjected
	}
	return tx.ReadwriteTransaction.Set(ctx, record)
}

func (tx failingTx) Delete(ctx context.Context, key *dal.Key) error {
	if tx.fault == faultDelete {
		return errInjected
	}
	return tx.ReadwriteTransaction.Delete(ctx, key)
}

func (tx failingTx) DeleteMulti(ctx context.Context, keys []*dal.Key) error {
	if tx.fault == faultDeleteMulti {
		return errInjected
	}
	return tx.ReadwriteTransaction.DeleteMulti(ctx, keys)
}

func (tx failingTx) Insert(ctx context.Context, record dal.Record, opts ...dal.InsertOption) error {
	if tx.fault == faultInsert {
		return errInjected
	}
	return tx.ReadwriteTransaction.Insert(ctx, record, opts...)
}

// failingROTx is the readonly counterpart used by RunReadonlyTransaction.
type failingROTx struct {
	dal.ReadTransaction
	fault txFault
}

func (tx failingROTx) Get(ctx context.Context, record dal.Record) error {
	if tx.fault == faultGet {
		return errInjected
	}
	return tx.ReadTransaction.Get(ctx, record)
}

func (tx failingROTx) GetMulti(ctx context.Context, records []dal.Record) error {
	if tx.fault == faultGetMulti {
		return errInjected
	}
	return tx.ReadTransaction.GetMulti(ctx, records)
}
