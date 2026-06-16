package dtb_general

import (
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const mainMenuCommandCode = "main_menu"

var MainMenuCommand = botsfw.Command{
	Code:       mainMenuCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Icon:       emoji.MAIN_MENU_ICON,
	Commands:   trans.Commands(trans.COMMAND_MENU, emoji.MAIN_MENU_ICON),
	Title:      trans.COMMAND_TEXT_MAIN_MENU_TITLE,
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return MainMenuAction(whc, "", true)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return MainMenuAction(whc, "", true)
	},
}

func MainMenuAction(whc botsfw.WebhookContext, messageText string, showHint bool) (m botmsg.MessageFromBot, err error) {
	if messageText == "" {
		//if whc.BotPlatform().ContactID() != fbm.PlatformID {
		if showHint {
			messageText = fmt.Sprintf("%v\n\n%v", whc.Translate(trans.MESSAGE_TEXT_WHATS_NEXT), whc.Translate(trans.MESSAGE_TEXT_DEBTUS_COMMANDS))
		} else {
			messageText = whc.Translate(trans.MESSAGE_TEXT_WHATS_NEXT)
		}
		//}
	}
	logus.Infof(whc.Context(), "MainMenuCommand.Action()")
	whc.ChatData().SetAwaitingReplyTo("")
	m = whc.NewMessage(messageText)
	m.Format = botmsg.FormatHTML
	if err = SetMainMenuKeyboard(whc, &m); err != nil {
		return m, err
	}
	return m, nil
}
