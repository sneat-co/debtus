package dtb_admin

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/anybot/cmds4invites"
)

var adminCommand = botsfw.Command{
	Code:       "admin",
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/admin"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		m = whc.NewMessage("Admin menu")
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				{Text: "Create mass invite", CallbackData: cmds4invites.CreateMassInviteCommandCode},
			},
		)
		return
	},
}
