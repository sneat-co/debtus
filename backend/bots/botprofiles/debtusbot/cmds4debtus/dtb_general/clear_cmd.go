package dtb_general

import (
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

var clearCommand = botsfw.Command{
	Code:       "clear",
	Commands:   trans.Commands(trans.COMMAND_CLEAR),
	InputTypes: []botinput.Type{botinput.TypeText},
	//Title:    trans.COMMAND_TEXT_MAIN_MENU_TITLE,
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		logus.Warningf(whc.Context(), "User called /clear command (not implemented yet)")
		return MainMenuAction(whc, whc.Translate(trans.MESSAGE_TEXT_NOT_IMPLEMENTED_YET), false)
	},
}
