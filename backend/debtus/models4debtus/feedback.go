package models4debtus

import (
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
)

const FeedbackKind = "Feedback"

type FeedbackData struct {
	general4debtus.CreatedOn
	// Deprecated: use UserStrID instead
	UserID    int64
	UserStrID string
	Created   time.Time
	Rate      string
	Text      string `firestore:",omitempty"`
}

//var _ db.EntityHolder = (*Feedback)(nil)

type Feedback struct {
	record.WithID[string]
	*FeedbackData
}

func NewFeedbackKey(feedbackID string) *dal.Key {
	return dal.NewKeyWithID(FeedbackKind, feedbackID)
}

func NewFeedbackWithIncompleteKey(data *FeedbackData) Feedback {
	if data == nil {
		data = new(FeedbackData)
	}
	return Feedback{
		WithID:       record.NewWithID[string]("", dal.NewIncompleteKey(FeedbackKind, reflect.String, nil), data),
		FeedbackData: data,
	}
}

func NewFeedback(id string, data *FeedbackData) Feedback {
	key := NewFeedbackKey(id)
	if data == nil {
		data = new(FeedbackData)
	}
	return Feedback{
		WithID:       record.NewWithID(id, key, data),
		FeedbackData: data,
	}
}

//func (o *Feedback) Kind() string {
//	return FeedbackKind
//}
//
//func (o Feedback) Entity() interface{} {
//	return o.FeedbackData
//}
//
//func (Feedback) NewEntity() interface{} {
//	return new(FeedbackData)
//}
//
//func (o *Feedback) SetEntity(entity interface{}) {
//	o.FeedbackData = entity.(*FeedbackData)
//}
