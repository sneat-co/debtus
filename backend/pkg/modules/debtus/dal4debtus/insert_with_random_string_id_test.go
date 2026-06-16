package dal4debtus

import (
	"context"
	"reflect"
	"testing"

	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
)

type testDbo struct {
	Name string `json:"name"`
}

func TestInsertWithRandomStringID(t *testing.T) {
	ctx := context.Background()
	db := dalgo2memory.NewDB()

	t.Run("generates_id_when_incomplete", func(t *testing.T) {
		var key *dal.Key
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			record := dal.NewRecordWithIncompleteKey("tests", reflect.String, &testDbo{Name: "first"})
			if err := InsertWithRandomStringID(ctx, tx, record); err != nil {
				return err
			}
			key = record.Key()
			return nil
		})
		if err != nil {
			t.Fatalf("InsertWithRandomStringID() returned error: %v", err)
		}
		id, ok := key.ID.(string)
		if !ok || id == "" {
			t.Fatalf("expected non-empty generated string ID, got %#v", key.ID)
		}
		data := new(testDbo)
		record := dal.NewRecordWithData(dal.NewKeyWithID("tests", id), data)
		if err = db.Get(ctx, record); err != nil {
			t.Fatalf("inserted record not found by generated ID %q: %v", id, err)
		}
		if data.Name != "first" {
			t.Errorf("expected Name=first, got %q", data.Name)
		}
	})

	t.Run("keeps_preset_id", func(t *testing.T) {
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			record := dal.NewRecordWithData(dal.NewKeyWithID("tests", "preset"), &testDbo{Name: "second"})
			return InsertWithRandomStringID(ctx, tx, record)
		})
		if err != nil {
			t.Fatalf("InsertWithRandomStringID() returned error: %v", err)
		}
		data := new(testDbo)
		record := dal.NewRecordWithData(dal.NewKeyWithID("tests", "preset"), data)
		if err = db.Get(ctx, record); err != nil {
			t.Fatalf("inserted record not found by preset ID: %v", err)
		}
		if data.Name != "second" {
			t.Errorf("expected Name=second, got %q", data.Name)
		}
	})
}
