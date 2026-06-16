package shared_splitus

import (
	"errors"
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
)

type SplitusSpaceAction func(whc botsfw.WebhookContext, splitusSpace models4splitus.SplitusSpaceEntry) (m botmsg.MessageFromBot, err error)

type SplitusSpaceCallbackAction func(whc botsfw.WebhookContext, callbackUrl *url.URL, splitusSpace models4splitus.SplitusSpaceEntry) (m botmsg.MessageFromBot, err error)

func NewSplitusSpaceAction(f SplitusSpaceAction) botsfw.CommandAction {
	return func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		var splitusSpace models4splitus.SplitusSpaceEntry
		if splitusSpace, err = GetSplitusSpaceEntryByCallbackUrl(whc, nil); err != nil {
			return
		}
		return f(whc, splitusSpace)
	}
}

func NewSplitusSpaceCallbackAction(f SplitusSpaceCallbackAction) botsfw.CallbackAction {
	return func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		var splitusSpace models4splitus.SplitusSpaceEntry
		if splitusSpace, err = GetSplitusSpaceEntryByCallbackUrl(whc, nil); err != nil {
			return
		}
		return f(whc, callbackUrl, splitusSpace)
	}
}

func GetSplitusSpaceEntryByCallbackUrl(whc botsfw.WebhookContext, callbackUrl *url.URL) (splitusSpace models4splitus.SplitusSpaceEntry, err error) {
	_, _ = whc, callbackUrl
	err = errors.New("func GetSplitusSpaceEntryByCallbackUrl() is not implemented yet")
	return
}
