package api4debtus

import (
	"net/http"

	"github.com/sneat-co/sneat-core-modules/auth/api4auth"
	"github.com/strongo/strongoapp"
)

func InitApiForDebtus(handle strongoapp.HandleHttpWithContext) {
	handle(http.MethodGet, "/api4debtus/receipt-get", HandleGetReceipt)
	handle(http.MethodPost, "/api4debtus/receipt-create", api4auth.AuthOnly(HandleCreateReceipt))
	handle(http.MethodPost, "/api4debtus/receipt-send", api4auth.AuthOnlyWithUser(HandleSendReceipt))
	handle(http.MethodPost, "/api4debtus/receipt-set-channel", HandleSetReceiptChannel)
	handle(http.MethodPost, "/api4debtus/receipt-ack-accept", HandleReceiptAccept)
	handle(http.MethodPost, "/api4debtus/receipt-ack-decline", HandleReceiptDecline)
	//handle(http.MethodPost, "/api4debtus/invite-friend", inviteFriend)
}
