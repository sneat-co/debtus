package models4debtus

import (
	"reflect"
	"strings"
	"testing"
)

// TestWithCounterpartyFields_uniqueFirestoreTags guards against the
// copy-paste bug where CounterpartySpaceID was tagged with the same
// firestore property name as CounterpartyUserID, so the two fields
// overwrote each other in storage.
func TestWithCounterpartyFields_uniqueFirestoreTags(t *testing.T) {
	tt := reflect.TypeFor[WithCounterpartyFields]()
	seen := map[string]string{}
	for f := range tt.Fields() {
		name, _, _ := strings.Cut(f.Tag.Get("firestore"), ",")
		if name == "" {
			name = f.Name
		}
		if prev, ok := seen[name]; ok {
			t.Errorf("duplicate firestore property %q on fields %s and %s", name, prev, f.Name)
		}
		seen[name] = f.Name
	}
}
