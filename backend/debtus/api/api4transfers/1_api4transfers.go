package api4transfers

import (
	"net/http"

	"github.com/sneat-co/sneat-core-modules/auth/api4auth"
	"github.com/strongo/strongoapp"
)

func InitApiForTransfers(handle strongoapp.HandleHttpWithContext) {
	handle(http.MethodGet, "/api4debtus/transfer", api4auth.AuthOnly(HandleGetTransfer))
	handle(http.MethodPost, "/api4debtus/create-transfer", api4auth.AuthOnly(HandleCreateTransfer))
	handle(http.MethodGet, "/api4debtus/user/api4transfers", api4auth.AuthOnlyWithUser(HandleUserTransfers))
	handle(http.MethodGet, "/api4debtus/admin/latest/api4transfers", api4auth.AdminOnly(HandleAdminLatestTransfers))
}
