package facade4splitus

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
)

// ---- test doubles for dal.DB / dal.ReadwriteTransaction ----

// fakeDB wraps a real dal.DB (usually the dalgo2memory one) and lets tests
// override individual operations. Nil fields delegate to the embedded DB.
type fakeDB struct {
	dal.DB
	get         func(ctx context.Context, rec dal.Record) error
	wrapTx      func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction
	queryReader dal.RecordsReader
	queryErr    error
	useQuery    bool
}

func (f fakeDB) Get(ctx context.Context, rec dal.Record) error {
	if f.get != nil {
		return f.get(ctx, rec)
	}
	return f.DB.Get(ctx, rec)
}

func (f fakeDB) RunReadwriteTransaction(ctx context.Context, worker func(ctx context.Context, tx dal.ReadwriteTransaction) error, opts ...dal.TransactionOption) error {
	if f.wrapTx == nil {
		return f.DB.RunReadwriteTransaction(ctx, worker, opts...)
	}
	return f.DB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return worker(ctx, f.wrapTx(tx))
	}, opts...)
}

func (f fakeDB) ExecuteQueryToRecordsReader(ctx context.Context, query dal.Query) (dal.RecordsReader, error) {
	if f.useQuery {
		return f.queryReader, f.queryErr
	}
	return f.DB.ExecuteQueryToRecordsReader(ctx, query)
}

// overrideSneatDB points facade.GetSneatDB at the given DB for the duration of
// the test and restores the previous value afterwards.
func overrideSneatDB(t *testing.T, db dal.DB) {
	t.Helper()
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return db, nil
	}
	t.Cleanup(func() { facade.GetSneatDB = orig })
}

// failSneatDB makes facade.GetSneatDB return the given error.
func failSneatDB(t *testing.T, err error) {
	t.Helper()
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return nil, err
	}
	t.Cleanup(func() { facade.GetSneatDB = orig })
}

// txWrap wraps a dal.ReadwriteTransaction allowing tests to inject errors into
// individual operations. Nil fields delegate to the embedded transaction.
type txWrap struct {
	dal.ReadwriteTransaction
	get      func(rec dal.Record) error // returning non-nil error short-circuits
	set      func(rec dal.Record) error
	insert   func(rec dal.Record) error
	setMulti func(recs []dal.Record) error
	getMulti func(recs []dal.Record) error
}

func (t txWrap) Get(ctx context.Context, rec dal.Record) error {
	if t.get != nil {
		if err := t.get(rec); err != nil {
			return err
		}
	}
	return t.ReadwriteTransaction.Get(ctx, rec)
}

func (t txWrap) Set(ctx context.Context, rec dal.Record) error {
	if t.set != nil {
		if err := t.set(rec); err != nil {
			return err
		}
	}
	return t.ReadwriteTransaction.Set(ctx, rec)
}

func (t txWrap) Insert(ctx context.Context, rec dal.Record, opts ...dal.InsertOption) error {
	if t.insert != nil {
		if err := t.insert(rec); err != nil {
			return err
		}
	}
	return t.ReadwriteTransaction.Insert(ctx, rec, opts...)
}

func (t txWrap) SetMulti(ctx context.Context, recs []dal.Record) error {
	if t.setMulti != nil {
		if err := t.setMulti(recs); err != nil {
			return err
		}
	}
	return t.ReadwriteTransaction.SetMulti(ctx, recs)
}

func (t txWrap) GetMulti(ctx context.Context, recs []dal.Record) error {
	if t.getMulti != nil {
		if err := t.getMulti(recs); err != nil {
			return err
		}
	}
	return t.ReadwriteTransaction.GetMulti(ctx, recs)
}

// fakeRecordsReader is a dal.RecordsReader serving bill keys with the given
// string IDs. If nextErr is set it is returned by the first call to Next.
type fakeRecordsReader struct {
	keys    []*dal.Key
	i       int
	nextErr error
}

func (r *fakeRecordsReader) Next() (dal.Record, error) {
	if r.nextErr != nil {
		return nil, r.nextErr
	}
	if r.i >= len(r.keys) {
		return nil, dal.ErrNoMoreRecords
	}
	rec := dal.NewRecordWithData(r.keys[r.i], &struct{}{})
	r.i++
	return rec, nil
}

func (r *fakeRecordsReader) Cursor() (string, error) { return "", nil }
func (r *fakeRecordsReader) Close() error            { return nil }
