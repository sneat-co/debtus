package models4debtus

import (
	"errors"
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/general4debtus"
)

const (
	ReceiptKind = "ReceiptEntry"

	ReceiptStatusCreated      = "created"
	ReceiptStatusSending      = "sending"
	ReceiptStatusSent         = "sent"
	ReceiptStatusViewed       = "viewed"
	ReceiptStatusAcknowledged = "acknowledged"
)

var ReceiptStatuses = [4]string{
	ReceiptStatusCreated,
	ReceiptStatusSent,
	ReceiptStatusViewed,
	ReceiptStatusAcknowledged,
}

type ReceiptEntry = record.DataWithID[string, *ReceiptDbo]

func NewReceiptKey(id string) *dal.Key {
	if id == "" {
		return NewReceiptIncompleteKey()
	}
	return dal.NewKeyWithID(ReceiptKind, id)
}

func NewReceiptWithoutID(data *ReceiptDbo) ReceiptEntry {
	key := NewReceiptIncompleteKey()
	if data == nil {
		data = new(ReceiptDbo)
	}
	return record.NewDataWithID("", key, data)
}

func NewReceipt(id string, data *ReceiptDbo) ReceiptEntry {
	key := NewReceiptKey(id)
	if data == nil {
		data = new(ReceiptDbo)
	}
	return record.NewDataWithID(id, key, data)
}

const (
	ReceiptForFrom = "from"
	ReceiptForTo   = "to"
)

type ReceiptFor string

type ReceiptDbo struct {
	Status               string            `json:"status" firestore:"status"`
	SpaceID              coretypes.SpaceID `json:"spaceID" firestore:"spaceID"`
	CounterpartySpaceID  string            `json:"counterpartySpaceID,omitempty" firestore:"counterpartySpaceID,omitempty"`
	TransferID           string            `json:"transferID" firestore:"transferID"`
	CreatorUserID        string            `json:"creatorUserID" firestore:"creatorUserID"` // IMPORTANT: Can be different from transfer.CreatorUserID (usually same). Think of 3d party bills
	For                  ReceiptFor        `json:"for" firestore:"for"`                     // TODO: always fill. If receipt.CreatorUserID != transfer.CreatorUserID then receipt.For must be set to either "from" or "to"
	ViewedByUserIDs      []string          `json:"viewedByUserIDs,omitempty" firestore:"viewedByUserIDs,omitempty"`
	CounterpartyUserID   string            `json:"counterpartyUserID" firestore:"counterpartyUserID"`                         // TODO: Is it always equal to AcknowledgedByUserID?
	AcknowledgedByUserID string            `json:"acknowledgedByUserID,omitempty" firestore:"acknowledgedByUserID,omitempty"` // TODO: Is it always equal to CounterpartyUserID?
	general4debtus.CreatedOn
	TgInlineMsgID  string    `firestore:"tgInlineMsgID,omitempty"`
	DtCreated      time.Time `json:"dtCreated" firestore:"dtCreated"`
	DtSent         time.Time `json:"dtSent,omitempty" firestore:"dtSent,omitempty"`
	DtFailed       time.Time `json:"dtFailed,omitempty" firestore:"dtFailed,omitempty"`
	DtViewed       time.Time `json:"dtViewed,omitempty" firestore:"dtViewed,omitempty"`
	DtAcknowledged time.Time `json:"dtAcknowledged,omitempty" firestore:"dtAcknowledged,omitempty"`
	SentVia        string    `json:"sentVia,omitempty" firestore:"sentVia,omitempty"`
	SentTo         string    `json:"sentTo,omitempty" firestore:"sentTo,omitempty"`
	Lang           string    `json:"lang" firestore:"lang"`
	Error          string    `json:"error" firestore:"error,omitempty"` //TODO: Need a comment on when it is used
}

func NewReceiptIncompleteKey() *dal.Key {
	return dal.NewIncompleteKey(ReceiptKind, reflect.String, nil)
}

func NewReceiptEntity(creatorUserID, transferID, counterpartyUserID, lang, sentVia, sentTo string, createdOn general4debtus.CreatedOn) *ReceiptDbo {
	if creatorUserID == counterpartyUserID {
		panic("creatorUserID == counterpartyUserID")
	}
	if transferID == "" {
		panic("transferID == 0")
	}
	if createdOn.CreatedOnID == "" {
		panic("CreatedOnID is empty")
	}
	if createdOn.CreatedOnPlatform == "" {
		panic("CreatedOnPlatform is empty")
	}
	return &ReceiptDbo{
		CreatorUserID:      creatorUserID,
		CounterpartyUserID: counterpartyUserID,
		TransferID:         transferID,
		CreatedOn:          createdOn,
		DtCreated:          time.Now(),
		Lang:               lang,
		SentVia:            sentVia,
		SentTo:             sentTo,
		Status:             ReceiptStatusCreated,
	}
}

func (r *ReceiptDbo) Validate() (err error) {
	if r.TransferID == "" {
		return errors.New("receipt.TransferID == 0")
	}
	if err = ValidateString("Unknown receipt.Status", r.Status, ReceiptStatuses[:]); err != nil {
		return err
	}
	if r.CreatorUserID == "" {
		err = errors.New("ReceiptDbo.CreatorUserID == 0")
		return
	}
	if r.CounterpartyUserID == r.CreatorUserID {
		err = errors.New("ReceiptDbo.CounterpartyUserID == ReceiptDbo.CreatorUserID")
		return
	}
	if r.CreatedOnID == "" {
		err = errors.New("ReceiptDbo.CreatedOnID is empty")
		return
	}
	if r.CreatedOnPlatform == "" {
		err = errors.New("ReceiptDbo.CreatedOnPlatform is empty")
		return
	}
	if r.Lang == "" {
		err = errors.New("ReceiptDbo.Lang is empty")
		return
	}
	if r.Status == "" {
		err = errors.New("ReceiptDbo.Status is empty")
		return
	}

	if r.DtCreated.IsZero() {
		r.DtCreated = time.Now()
	}

	return
}
