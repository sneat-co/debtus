package debtus

import (
	"net/http"

	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/api4unsorted"
	"github.com/sneat-co/debtus/backend/bots/delayers4debtusbot"
	"github.com/sneat-co/debtus/backend/debtus/api/api4debtus"
	"github.com/sneat-co/debtus/backend/debtus/api/api4transfers"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/reminders"
	"github.com/sneat-co/sneat-go-core/extension"
	"github.com/strongo/delaying"
	"github.com/strongo/strongoapp"
)

const extensionID = const4debtus.ModuleID

func Extension() extension.Config {
	debtusdal.RegisterDal() // TODO: This does not feels right, should be done in some module's init?
	// IMPORTANT: extension.RegisterRoutes/RegisterDelays OVERWRITE on repeat use
	// (last option wins), so each must be passed exactly once with a combined
	// closure. Passing api4debtus & api4transfers as separate options silently
	// dropped the api4debtus routes.
	return extension.NewExtension(extensionID,
		extension.RegisterRoutes(func(handle extension.HTTPHandleFunc) {
			// TODO: This should be unified with the rest of APIs
			handleWithContext := func(method, path string, handler strongoapp.HttpHandlerWithContext) {
				handle(method, path, func(writer http.ResponseWriter, request *http.Request) {
					handler(request.Context(), writer, request)
				})
			}
			api4debtus.InitApiForDebtus(handleWithContext)
			api4transfers.InitApiForTransfers(handleWithContext)
		}),
		extension.RegisterDelays(func(mustRegisterFunc func(key string, i any) delaying.Delayer) {
			facade4debtus.InitDelays4debtus(mustRegisterFunc)
			debtusdal.RegisterDelayers4Debtus(mustRegisterFunc)
			delayers4debtusbot.InitDelayers(mustRegisterFunc)
			api4unsorted.InitDelaying(mustRegisterFunc)
			reminders.InitDelaying(mustRegisterFunc)
		}),
	)
}
