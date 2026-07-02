package facade4splitus

import (
	"context"
	"errors"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-go-core/coretypes"
)

// billsQueryReader is a dal.RecordsReader yielding a fixed set of bill
// records, used to test ListBillsBySpace's record-to-BillEntry conversion in
// isolation from the query engine's own WHERE-filter behavior (exercised
// separately/implicitly the same way the rest of this package tests query
// facades — see settle2members_test.go's settleQueryDB/fakeRecordsReader).
type billsQueryReader struct {
	records []dal.Record
	i       int
	nextErr error
}

func (r *billsQueryReader) Next() (dal.Record, error) {
	if r.i >= len(r.records) {
		if r.nextErr != nil {
			return nil, r.nextErr
		}
		return nil, dal.ErrNoMoreRecords
	}
	rec := r.records[r.i]
	r.i++
	return rec, nil
}

func (r *billsQueryReader) Cursor() (string, error) { return "", nil }
func (r *billsQueryReader) Close() error            { return nil }

func billRecord(id string, forSpaceID coretypes.SpaceID, name string) dal.Record {
	billEntity := models4splitus.NewBillEntity(models4splitus.BillCommon{
		SpaceID:       forSpaceID,
		Status:        models4splitus.BillStatusOutstanding,
		SplitMode:     models4splitus.SplitModeEqually,
		CreatorUserID: "u1",
		Name:          name,
		Currency:      "EUR",
		AmountTotal:   100,
	})
	return dal.NewRecordWithData(dal.NewKeyWithID(models4splitus.BillKind, id), billEntity).SetError(nil)
}

func TestListBillsBySpace_MissingSpaceID(t *testing.T) {
	if _, err := ListBillsBySpace(context.Background(), ""); err == nil {
		t.Fatal("expected error for missing spaceID")
	}
}

func TestListBillsBySpace_GetSneatDBError(t *testing.T) {
	wantErr := errors.New("no db")
	failSneatDB(t, wantErr)
	if _, err := ListBillsBySpace(context.Background(), spaceID); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestListBillsBySpace_QueryError(t *testing.T) {
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("query failed")
	overrideSneatDB(t, &fakeDB{DB: memDB, useQuery: true, queryErr: wantErr})
	if _, err := ListBillsBySpace(context.Background(), spaceID); !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// TestListBillsBySpace_ReaderError verifies a non-ErrNoMoreRecords error from
// the reader mid-stream is surfaced rather than swallowed.
func TestListBillsBySpace_ReaderError(t *testing.T) {
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("reader failed")
	reader := &billsQueryReader{
		records: []dal.Record{billRecord("bill1", spaceID, "Dinner")},
		nextErr: wantErr,
	}
	overrideSneatDB(t, &fakeDB{DB: memDB, useQuery: true, queryReader: reader})
	bills, err := ListBillsBySpace(context.Background(), spaceID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
	if len(bills) != 1 {
		t.Errorf("expected the 1 successfully-read bill despite the later error, got %d", len(bills))
	}
}

// TestListBillsBySpace_ReturnsSpaceBills verifies the reader-to-BillEntry
// conversion: IDs and data round-trip, and reading stops cleanly at
// dal.ErrNoMoreRecords with no error surfaced.
func TestListBillsBySpace_ReturnsSpaceBills(t *testing.T) {
	memDB := sneattesting.SetupMemoryDB(t)
	reader := &billsQueryReader{records: []dal.Record{
		billRecord("bill1", spaceID, "Dinner"),
		billRecord("bill2", spaceID, "Taxi"),
	}}
	overrideSneatDB(t, &fakeDB{DB: memDB, useQuery: true, queryReader: reader})

	bills, err := ListBillsBySpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ListBillsBySpace() error: %v", err)
	}
	if len(bills) != 2 {
		t.Fatalf("expected 2 bills, got %d: %+v", len(bills), bills)
	}
	if bills[0].ID != "bill1" || bills[0].Data.Name != "Dinner" {
		t.Errorf("bills[0] = %+v, want id=bill1 name=Dinner", bills[0])
	}
	if bills[1].ID != "bill2" || bills[1].Data.Name != "Taxi" {
		t.Errorf("bills[1] = %+v, want id=bill2 name=Taxi", bills[1])
	}
}

// TestListBillsBySpace_NoBills verifies an empty (but non-error) result
// against the real in-memory query engine for a space with no bills.
func TestListBillsBySpace_NoBills(t *testing.T) {
	sneattesting.SetupMemoryDB(t)
	bills, err := ListBillsBySpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ListBillsBySpace() error: %v", err)
	}
	if len(bills) != 0 {
		t.Errorf("expected 0 bills, got %d", len(bills))
	}
}
