package api4reminders

import (
	"net/http"

	"github.com/strongo/strongoapp"
)

func InitApiForReminder(handle strongoapp.HandleHttpWithContext) {
	handle(http.MethodPost, "/api4debtus/send-receipt", sendReceipt)
	handle(http.MethodGet, "/api4debtus/test/email", testEmail)
}
