package debtusbot

import (
	"reflect"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
)

func NewDebtusTelegramChatRecord() dal.Record {
	return dal.NewRecordWithIncompleteKey(botsfwtgmodels.TgChatCollection, reflect.String, new(anybot.SneatAppTgChatDbo))
}
