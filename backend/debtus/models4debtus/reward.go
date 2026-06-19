package models4debtus

import (
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
)

const RewardKind = "Reward"

type Reward = record.DataWithID[string, *RewardDbo]

func NewReward(id string, data *RewardDbo) Reward {
	key := dal.NewKeyWithID(RewardKind, id)
	if data == nil {
		data = new(RewardDbo)
	}
	return Reward{
		RecordWithID: record.NewWithID(id, key, data),
		Data:         data,
	}
}

func NewRewardWithIncompleteKey(data *RewardDbo) Reward {
	key := dal.NewIncompleteKey(RewardKind, reflect.Int, nil)
	if data == nil {
		data = new(RewardDbo)
	}
	return Reward{
		RecordWithID: record.NewWithID("", key, data),
		Data:         data,
	}
}

type RewardReason string

const (
	RewardReasonInvitedUserJoined         RewardReason = "InvitedUserJoined"
	RewardReasonFriendOfInvitedUserJoined RewardReason = "FriendOfInvitedUserJoined"
)

type RewardDbo struct {
	UserID       int64
	DtCreated    time.Time
	Reason       RewardReason `firestore:",omitempty"`
	JoinedUserID int64        `firestore:",omitempty"`
	Points       int          `firestore:",omitempty"`
}
