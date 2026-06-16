package dal4reminders

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
)

func GetDueReminderIDs(ctx context.Context, db dal.QueryExecutor) (reminderIDs []string, err error) {
	query := dal.From(dal.NewRootCollectionRef(dbo4reminders.ReminderKind, "")).
		NewQuery().
		WhereField("status", dal.Equal, dbo4reminders.ReminderStatusCreated).
		WhereField("dtNext", dal.GreaterThen, time.Time{}).
		WhereField("dtNext", dal.LessThen, time.Now()).
		OrderBy(dal.AscendingField("dtNext")).
		Limit(100).
		SelectKeysOnly(reflect.Int)

	var reader dal.RecordsReader
	if reader, err = db.ExecuteQueryToRecordsReader(ctx, query); err != nil {
		err = fmt.Errorf("failed to execute query to get due reminder IDs: %w", err)
		return
	}
	if reminderIDs, err = dal.SelectAllIDs[string](ctx, reader, dal.WithLimit(query.Limit())); err != nil {
		err = fmt.Errorf("failed to read due reminder IDs: %w", err)
		return
	}
	return
}
