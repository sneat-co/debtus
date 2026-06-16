package cmds4debtus

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_transfer"
	"github.com/strongo/logus"
)

// urlParse is a seam for url.Parse so tests can inject errors.
var urlParse = url.Parse

// onInlineChosenCreateReceipt is a seam for dtb_transfer.OnInlineChosenCreateReceipt.
var onInlineChosenCreateReceipt = dtb_transfer.OnInlineChosenCreateReceipt

//var chosenInlineResultCommand = botsfw.Command{
//	Code:       "inline-create-invite",
//	InputTypes: []botinput.Type{botinput.TypeChosenInlineResult},
//	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
//
//	},
//}

func chosenResultQueryHandler(whc botsfw.WebhookContext, inlineQuery botinput.ChosenInlineResult) (handled bool, m botmsg.MessageFromBot, err error) {
	_ = inlineQuery
	ctx := whc.Context()
	chosenResult := whc.Input().(botinput.ChosenInlineResult)
	query := chosenResult.GetQuery()
	logus.Debugf(ctx, "chosenInlineResultCommand.Action() => query: %v", query)

	var queryUrl *url.URL
	if queryUrl, err = urlParse(query); err != nil {
		return
	}

	switch queryUrl.Path {
	case "receipt":
		m, err = onInlineChosenCreateReceipt(whc, chosenResult.GetInlineMessageID(), queryUrl)
		handled = true
	default:
		logus.Warningf(ctx, "Unknown chosen inline query: "+query)
	}
	return
}
