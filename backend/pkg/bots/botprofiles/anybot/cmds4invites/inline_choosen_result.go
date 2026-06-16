package cmds4invites

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/strongo/logus"
)

var chosenInlineResultCommand = botsfw.Command{
	Code:                     "invite2join",
	ChosenInlineResultAction: chosenInlineResultAction,
}

func chosenInlineResultAction(whc botsfw.WebhookContext, chosenResult botinput.ChosenInlineResult, queryUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "cmds4invites.chosenInlineResultAction.Action(), queryUrl: %v", queryUrl)
	//m, err = whc.NewEditMessage(fmt.Sprintf("Chosen inline result: {%s, %s}", chosenResult.GetInlineMessageID(), chosenResult.GetQuery()), botmsg.FormatHTML)
	return
}

//func chosenResultQueryHandler(
//	whc botsfw.WebhookContext,
//	chosenInlineResult botinput.ChosenInlineResult,
//) (
//	handled bool, m botmsg.MessageFromBot, err error,
//) {
//
//}
