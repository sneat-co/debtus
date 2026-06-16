package debtusdal

import (
	"reflect"
	"testing"
)

// TestNewDAL_populatesEveryField guards the R1 invariant: NewDAL is total.
// A new field added to dal4debtus.DAL without a corresponding assignment in
// NewDAL fails here instead of nil-panicking at runtime.
func TestNewDAL_populatesEveryField(t *testing.T) {
	v := reflect.ValueOf(NewDAL())
	for f := range v.Type().Fields() {
		if v.FieldByIndex(f.Index).IsZero() {
			t.Errorf("dal4debtus.DAL.%s is not populated by NewDAL()", f.Name)
		}
	}
}
