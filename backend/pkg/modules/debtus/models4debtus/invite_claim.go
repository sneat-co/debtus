package models4debtus

import (
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
)

const InviteClaimKind = "InviteClaim"

type InviteClaim struct {
	record.WithID[string]
	Data *InviteClaimData
}

func NewInviteClaimWithoutID(data *InviteClaimData) InviteClaim {
	if data == nil {
		data = new(InviteClaimData)
	}
	return InviteClaim{
		WithID: record.NewWithID("", NewInviteClaimIncompleteKey(), data),
		Data:   data,
	}
}

func NewInviteClaim(id string, data *InviteClaimData) InviteClaim {
	if data == nil {
		data = new(InviteClaimData)
	}
	return InviteClaim{
		WithID: record.NewWithID(id, NewInviteClaimKey(id), data),
		Data:   data,
	}
}

type InviteClaimData struct {
	InviteCode string // We don't use it as parent key as can be a bottleneck for public invites
	UserID     string
	DtClaimed  time.Time
	ClaimedOn  string // For example: "Telegram"
	ClaimedVia string // For the Telegram it would be bot name
}

func NewInviteClaimIncompleteKey() *dal.Key {
	return dal.NewIncompleteKey(InviteClaimKind, reflect.String, nil)
}

func NewInviteClaimKey(claimID string) *dal.Key {
	if claimID == "" {
		return dal.NewIncompleteKey(InviteClaimKind, reflect.String, nil)
	}
	return dal.NewKeyWithID(InviteClaimKind, claimID)
}

func NewInviteClaimData(inviteCode string, userID string, claimedOn, claimedVia string) *InviteClaimData {
	return &InviteClaimData{
		InviteCode: inviteCode,
		UserID:     userID,
		ClaimedOn:  claimedOn,
		ClaimedVia: claimedVia,
		DtClaimed:  time.Now(),
	}
}
