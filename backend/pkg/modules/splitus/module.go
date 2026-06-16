package splitus

import (
	"net/http"

	"github.com/sneat-co/sneat-go-core/extension"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/splitusbot/cmds4splitusbot"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/splitusbot/facade4splitusbot"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/api4splitusbot"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/const4splitus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/facade4splitus"
	"github.com/strongo/delaying"
	"github.com/strongo/strongoapp"
)

const moduleID = const4splitus.ModuleID

func Module() extension.Config {
	return extension.NewExtension(moduleID,
		extension.RegisterRoutes(func(handle extension.HTTPHandleFunc) {
			// TODO: This should be unified with the rest of APIs
			api4splitusbot.InitApiForSplitus(func(method, path string, handler strongoapp.HttpHandlerWithContext) {
				handle(method, path, func(writer http.ResponseWriter, request *http.Request) {
					handler(request.Context(), writer, request)
				})
			})
		}),
		// IMPORTANT: extension.RegisterRoutes/RegisterDelays OVERWRITE on repeat use
		// (last option wins), so each must be passed exactly once with a combined closure.
		extension.RegisterDelays(func(mustRegisterFunc func(key string, i any) delaying.Delayer) {
			facade4splitusbot.InitDelayingFotSplitusBot(mustRegisterFunc)
			facade4splitus.InitDelaying(mustRegisterFunc)
			cmds4splitusbot.InitDelaying(mustRegisterFunc)
		}),
	)
}
