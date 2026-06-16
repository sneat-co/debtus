package models4debtus

import (
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
)

const ReferrersCollection = "referrers"

type RefererEntry = record.DataWithID[string, *ReferrerDbo]

func NewReferrerEntry(dbo *ReferrerDbo) RefererEntry {
	if dbo == nil {
		panic("NewReferrerEntry: dbo is nil")
	}
	key := dal.NewIncompleteKey(ReferrersCollection, reflect.String, nil)
	return record.NewDataWithID("", key, dbo)
}

type ReferrerDbo struct {
	Platform   string    `firestore:"p"`
	ReferredTo string    `firestore:"to"`
	ReferredBy string    `firestore:"by"`
	DtCreated  time.Time `firestore:"t"`
}
