package dbo4reminders

import (
	"errors"
	"strings"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/validation"
)

const (
	ReminderStatusCreated           = "created"
	ReminderStatusSending           = "sending"
	ReminderStatusFailed            = "failed"
	ReminderStatusSent              = "sent"
	ReminderStatusViewed            = "viewed"
	ReminderStatusRescheduled       = "rescheduled"
	ReminderStatusUsed              = "used"
	ReminderStatusDiscarded         = "discarded"
	ReminderStatusInvalidNoTransfer = "invalid:no-transfer"
)

var ReminderStatuses = []string{
	ReminderStatusCreated,
	ReminderStatusSending,
	ReminderStatusFailed,
	ReminderStatusSent,
	ReminderStatusViewed,
	ReminderStatusRescheduled,
	ReminderStatusUsed,
	ReminderStatusDiscarded,
	ReminderStatusInvalidNoTransfer,
}

const ReminderKind = "Reminder"

type Reminder = record.DataWithID[string, *ReminderDbo]

//var _ db.EntityHolder = (*Reminder)(nil)

func NewReminder(id string, entity *ReminderDbo) Reminder {
	if entity == nil {
		entity = new(ReminderDbo)
	}
	key := dal.NewKeyWithID(ReminderKind, id)
	return record.NewDataWithID(id, key, entity)
}

type ReminderDbo struct {
	ParentReminderID int  `json:"parentReminderID,omitempty" firestore:"parentReminderID,omitempty"`
	IsAutomatic      bool `json:"isAutomatic,omitempty" firestore:"isAutomatic,omitempty"`
	IsRescheduled    bool `json:"isRescheduled,omitempty" firestore:"isRescheduled,omitempty"`
	//
	DtNext      time.Time `json:"dtNext" firestore:"dtNext"`
	DtScheduled time.Time `json:"dtScheduled,omitzero" firestore:"dtScheduled,omitempty"` // DtNext moves here once sent
	DtCreated   time.Time `json:"dtCreated,omitzero" firestore:"dtCreated,omitempty"`
	DtUpdated   time.Time `json:"dtUpdated,omitzero" firestore:"dtUpdated,omitempty"`
	DtSent      time.Time `json:"dtSent,omitzero" firestore:"dtSent,omitempty"`
	DtUsed      time.Time `json:"dtUsed,omitzero" firestore:"dtUsed,omitempty"` // When a user clicks "Yes/no returned"
	DtViewed    time.Time `json:"dtViewed,omitzero" firestore:"dtViewed,omitempty"`
	DtDiscarded time.Time `json:"dtDiscarded,omitzero" firestore:"dtDiscarded,omitempty"`
	//
	Locale         string `json:"locale,omitempty" firestore:"locale,omitempty"`
	SentVia        string `json:"sentVia,omitempty" firestore:"sentVia,omitempty"`
	Status         string `json:"status" firestore:"status"`
	UserID         string `json:"userID" firestore:"userID"`
	CounterpartyID string `json:"counterpartyID,omitempty" firestore:"counterpartyID,omitempty"` // If this field is not empty then the reminder is to a counterparty
	BotID          string `json:"botID,omitempty" firestore:"botID,omitempty"`
	ChatIntID      int64  `json:"chatIntID,omitempty" firestore:"chatIntID,omitempty"`
	MessageIntID   int64  `json:"messageIntID,omitempty" firestore:"messageIntID,omitempty"`
	MessageStrID   string `json:"messageStrID,omitempty" firestore:"messageStrID,omitempty"`
	ErrDetails     string `json:"errDetails,omitempty" firestore:"errDetails,omitempty"`

	//
	TargetKind string `json:"targetKind,omitempty" firestore:"targetKind,omitempty"`
	TargetID   string `json:"targetID,omitempty" firestore:"targetID,omitempty"`
}

func (r *ReminderDbo) Validate() (err error) {
	if err = models4debtus.ValidateString("Unknown reminder.Status", r.Status, ReminderStatuses); err != nil {
		return err
	}
	if targetID := strings.TrimSpace(r.TargetID); targetID != r.TargetID {
		return validation.NewErrBadRecordFieldValue("targetID", "trailing spaces not allowed")
	}
	if targetKind := strings.TrimSpace(r.TargetKind); targetKind != r.TargetKind {
		return validation.NewErrBadRecordFieldValue("targetKind", "trailing spaces not allowed")
	}
	if r.SentVia == "" {
		return errors.New("reminder.SentVia is empty")
	}
	if r.DtCreated.IsZero() {
		return errors.New("reminder.DtCreated.IsZero()")
	}
	if !r.DtSent.IsZero() && r.DtSent.Before(r.DtCreated) {
		return errors.New("reminder.DtSent.Before(n.DtCreated)")
	}
	if !r.DtViewed.IsZero() && r.DtViewed.Before(r.DtSent) {
		return errors.New("reminder.DtViewed.Before(n.DtSent)")
	}
	if r.ChatIntID != 0 && r.BotID == "" || r.ChatIntID == 0 && r.BotID != "" {
		return errors.New("r.TgChatID != 0 && r.TgBot == '' || r.TgChatID == 0 && r.TgBot != ''")
	}
	return nil
}

func NewReminderViaTelegram(botID string, chatID int64, userID, transferID string, isAutomatic bool, next time.Time) (reminder *ReminderDbo) {
	return &ReminderDbo{
		Status:      ReminderStatusCreated,
		SentVia:     "telegram",
		BotID:       botID,
		ChatIntID:   chatID,
		UserID:      userID,
		TargetKind:  "transfer",
		TargetID:    transferID,
		DtCreated:   time.Now(),
		IsAutomatic: isAutomatic,
		DtNext:      next,
	}
}

func (r *ReminderDbo) ScheduleNextReminder(parentReminderID int, next time.Time) *ReminderDbo {
	reminder := *r
	reminder.ParentReminderID = parentReminderID
	reminder.Status = ReminderStatusRescheduled

	reminder.DtCreated = time.Now()
	reminder.DtNext = next
	reminder.Status = ReminderStatusCreated
	zero := time.Time{}
	reminder.DtSent = zero
	reminder.DtDiscarded = zero
	reminder.DtViewed = zero
	reminder.MessageStrID = ""
	reminder.MessageIntID = 0

	r.IsRescheduled = true
	return &reminder
}
