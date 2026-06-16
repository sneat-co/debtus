package dtb_general

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

const debtusHomeCommandCode = "debtus"

var debtusHomeCommand = botsfw.Command{
	Code:       debtusHomeCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/debts"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		m.Format = botmsg.FormatHTML
		m.Text = `<b>Debtus home</b>

Choose what you want to do:
`
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(
					whc.CommandText(trans.COMMAND_TEXT_GAVE, emoji.GIVE_ICON),
					"balance",
				),
				tgbotapi.NewInlineKeyboardButtonData(
					whc.CommandText(trans.COMMAND_TEXT_GOT, emoji.TAKE_ICON),
					"balance",
				),
			},
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(
					whc.CommandText(trans.COMMAND_TEXT_BALANCE, emoji.BALANCE_ICON),
					"balance",
				),
				tgbotapi.NewInlineKeyboardButtonData(
					whc.CommandText(trans.COMMAND_TEXT_HISTORY, emoji.HISTORY_ICON),
					"balance",
				),
			},
		)
		return
	},
}
