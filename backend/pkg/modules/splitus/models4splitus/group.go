package models4splitus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
	"github.com/strongo/validation"
)

const GroupKind = "Group"

type GroupEntry = record.DataWithID[string, *GroupDbo]

func NewGroupEntry(id string, data *GroupDbo) GroupEntry {
	key := NewGroupKey(id)
	if data == nil {
		data = new(GroupDbo)
	}
	return GroupEntry{
		RecordWithID: record.WithID[string]{
			ID:     id,
			Key:    key,
			Record: dal.NewRecordWithData(key, data),
		},
		Data: data,
	}
}

func NewGroupKey(id string) *dal.Key {
	if id == "" {
		key, err := dal.NewKeyWithOptions(GroupKind, dal.WithRandomStringID(dal.RandomLength(const4debtus.GroupIdLen)))
		if err != nil {
			panic(err.Error())
		}
		return key
	}
	return dal.NewKeyWithID(GroupKind, id)
}

type GroupDbo struct {
	CreatorUserID string
	//IsUser2User         bool   `firestore:",omitempty"`
	Name            string             `firestore:"name"`
	Note            string             `firestore:"note,omitempty"`
	DefaultCurrency money.CurrencyCode `firestore:"defaultCurrency,omitempty"`
	//
	telegramGroups      []briefs4splitus.GroupTgChatJson
	TelegramGroupsCount int    `firestore:"TgGroupsCount,omitempty"`
	TelegramGroupsJson  string `firestore:"TgGroupsJson,omitempty"`
	//
	BillsHolder
}

func (entity *GroupDbo) GetTelegramGroups() (tgGroups []briefs4splitus.GroupTgChatJson, err error) {
	if entity.telegramGroups != nil {
		return entity.telegramGroups, nil
	}
	if entity.TelegramGroupsJson != "" {
		if err = json.Unmarshal([]byte(entity.TelegramGroupsJson), &tgGroups); err != nil {
			return
		} else if len(tgGroups) != entity.TelegramGroupsCount {
			err = fmt.Errorf("len([]GroupTgChatJson) != entity.TelegramGroupsCount: %d != %d", len(tgGroups), entity.TelegramGroupsCount)
			return
		}
		entity.telegramGroups = tgGroups
	}
	return
}

func (entity *GroupDbo) SetTelegramGroups(tgGroups []briefs4splitus.GroupTgChatJson) (changed bool) {
	if data, err := json.Marshal(tgGroups); err != nil {
		panic(err.Error())
	} else {
		if s := string(data); s != entity.TelegramGroupsJson {
			entity.TelegramGroupsJson = s
			changed = true
		}
		if l := len(tgGroups); l != entity.TelegramGroupsCount {
			entity.TelegramGroupsCount = l
			changed = true
		}
	}
	return
}

func (entity *GroupDbo) Validate() error {
	if entity.CreatorUserID == "" {
		return validation.NewErrRecordIsMissingRequiredField("CreatorUserID")
	}
	if strings.TrimSpace(entity.Name) == "" {
		return validation.NewErrRecordIsMissingRequiredField("UserTitle")
	}
	return nil
}
