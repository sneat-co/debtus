package dtb_general

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/trans"
)

const PleaseWaitCommandCode = "please_wait"

var pleaseWaitCommand = botsfw.Command{
	Code: PleaseWaitCommandCode,
	CallbackAction: func(whc botsfw.WebhookContext, _ *url.URL) (botmsg.MessageFromBot, error) {
		return whc.NewMessageByCode(trans.MESSAGE_TEXT_PLEASE_WAIT), nil
	},
}
