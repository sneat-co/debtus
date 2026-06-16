package cmds4debtus

import (
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/strongo/logus"
)

const newChatMembersCommandCode = "new_chat_members"

var newChatMembersCommand = botsfw.Command{
	Code:       newChatMembersCommandCode,
	InputTypes: []botinput.Type{botinput.TypeNewChatMembers},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		var isInGroup bool
		if isInGroup, err = whc.IsInGroup(); err != nil {
			return
		} else if isInGroup {
			logus.Warningf(whc.Context(), "Leaving chat as @DebtusBot does not support group chats")
			m.BotMessage = telegram.LeaveChat{}
		}
		return
	},
}
