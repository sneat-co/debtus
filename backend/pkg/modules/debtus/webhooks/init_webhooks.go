package webhooks

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func InitWebhooks(router *httprouter.Router) {
	router.HandlerFunc(http.MethodPost, "/webhooks/twilio/", TwilioWebhook)
}
