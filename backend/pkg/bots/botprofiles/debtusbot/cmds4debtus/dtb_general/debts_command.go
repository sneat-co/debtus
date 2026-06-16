package dtb_general

import (
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-bots/pkg/bots/sneatbots/tghelpers"
	"github.com/sneat-co/sneat-translations/trans"
)

var getCurrentSpaceRef = bothelper.GetCurrentSpaceRef
var getMessageUID = telegram.GetMessageUID

var debtsCommand = botsfw.Command{
	Code:     "debts",
	Commands: []string{"/debts"},
	InputTypes: []botinput.Type{
		botinput.TypeText,
		botinput.TypeCallbackQuery,
	},
	CallbackAction: debtsCallbackAction,
	Action:         debtsAction,
}

func debtsCallbackAction(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	if m, err = debtsAction(whc); err != nil {
		return
	}

	keyboard := m.Keyboard.(*tgbotapi.InlineKeyboardMarkup)
	spaceRef := bothelper.GetSpaceRef(callbackUrl)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
		tghelpers.BackToSpaceMenuButton(spaceRef, whc),
	})
	if m, err = whc.NewEditMessage(m.Text, m.Format); err != nil {
		return
	}
	m.Keyboard = keyboard

	m.EditMessageUID, err = getMessageUID(whc)
	return
}

func debtsAction(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	m.Format = botmsg.FormatHTML
	m.Text = fmt.Sprintf("<b>%s</b>", whc.Translate(trans.FamilyDebts))
	//m.Text += "\n\n<i>Not implemented yet</i>"
	var spaceRef coretypes.SpaceRef
	if spaceRef, err = getCurrentSpaceRef(whc); err != nil {
		return
	}
	keyboard := debtsMainMenuInlineKeyboard(whc, spaceRef)
	m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	return
}
