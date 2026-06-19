package cmds4splitusbot

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

const menuCommandCode = "menu"

var menuCommand = botsfw.Command{
	Code:       menuCommandCode,
	Commands:   []string{"/" + menuCommandCode},
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		m.Text = whc.Translate(trans.SPLITUS_TG_COMMANDS)
		m.Format = botmsg.FormatHTML
		SetMainMenu(whc, &m)
		return
	},
}

func SetMainMenu(whc botsfw.WebhookContext, m *botmsg.MessageFromBot) {
	m.Keyboard = tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{
			{Text: groupsCommand.TitleByKey(botsfw.DefaultTitle, whc)},
			{Text: billsCommand.TitleByKey(botsfw.DefaultTitle, whc)},
		},
		[]tgbotapi.KeyboardButton{
			{Text: whc.CommandText(trans.SettingsButtonText, cmds4anybot.SettingsEmoji)},
			{Text: emoji.HELP_ICON + " " + whc.Translate(trans.COMMAND_TEXT_HELP)},
		},
	)
}
