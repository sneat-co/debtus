package facade4splitus

import (
	"context"
	"errors"
	"reflect"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
)

// billsListLimit caps how many bills a single ListBillsBySpace call returns.
// Splitus does not yet page the splits list — this keeps the read cheap and
// bounded until pagination is needed.
const billsListLimit = 200

// ListBillsBySpace returns spaceID's bills, most recently created first. Used
// by the read-only GET /api4splitus/splits list endpoint
// (splitus#ac:participants-from-contactus-membership-enforced): callers are
// expected to have already verified space membership before calling this.
func ListBillsBySpace(ctx context.Context, spaceID coretypes.SpaceID) (bills []models4splitus.BillEntry, err error) {
	if spaceID == "" {
		return nil, errors.New("spaceID is required")
	}

	query := dal.From(dal.NewRootCollectionRef(models4splitus.BillKind, "")).
		NewQuery().
		WhereField("spaceID", dal.Equal, string(spaceID)).
		OrderBy(dal.DescendingField("createdAt")).
		Limit(billsListLimit).
		SelectIntoRecord(newBillQueryRecord)

	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return nil, err
	}
	var reader dal.RecordsReader
	if reader, err = db.ExecuteQueryToRecordsReader(ctx, query); err != nil {
		return nil, err
	}

	for {
		var r dal.Record
		if r, err = reader.Next(); err != nil {
			if errors.Is(err, dal.ErrNoMoreRecords) {
				err = nil
			}
			return
		}
		bills = append(bills, record.NewDataWithID[string, *models4splitus.BillDbo](
			r.Key().ID.(string), r.Key(), r.Data().(*models4splitus.BillDbo),
		))
	}
}

// newBillQueryRecord builds an empty dal.Record with an incomplete BillEntry
// key/data pair for the query executor to populate — mirrors
// models4debtus.NewTransferRecord's role for the Transfers collection.
func newBillQueryRecord() dal.Record {
	key := dal.NewIncompleteKey(models4splitus.BillKind, reflect.String, nil)
	return dal.NewRecordWithData(key, new(models4splitus.BillDbo))
}
