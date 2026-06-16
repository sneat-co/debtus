package cmds4invites

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
)

const (
	CreateMassInviteCommandCode = "create_mass_invite"
)

var createMassInviteCommand = botsfw.Command{
	Code:       CreateMassInviteCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/massinvite"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		m = whc.NewMessage("Admin menu")

		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				{Text: "Create mass invite", URL: CreateMassInviteCommandCode},
			},
		)
		return
	},
}
