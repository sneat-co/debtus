package api4unsorted

import (
	"net/http"

	"github.com/sneat-co/sneat-core-modules/auth/api4auth"
	"github.com/strongo/strongoapp"
)

func InitApiForUnsorted(handle strongoapp.HandleHttpWithContext) {

	handle(http.MethodPost, "/api4debtus/tg-helpers/currency-selected", api4auth.AuthOnly(HandleTgHelperCurrencySelected))

	handle(http.MethodGet, "/api4debtus/contact-get", api4auth.AuthOnly(HandleGetContact))
	handle(http.MethodPost, "/api4debtus/contact-create", api4auth.AuthOnly(HandleCreateCounterparty))
	handle(http.MethodPost, "/api4debtus/contact-update", api4auth.AuthOnly(HandleUpdateCounterparty))
	handle(http.MethodPost, "/api4debtus/contact-delete", api4auth.AuthOnly(HandleDeleteContact))
	handle(http.MethodPost, "/api4debtus/contact-archive", api4auth.AuthOnly(HandleArchiveCounterparty))
	handle(http.MethodPost, "/api4debtus/contact-activate", api4auth.AuthOnly(HandleActivateCounterparty))

	handle(http.MethodPost, "/api4debtus/group-create", api4auth.AuthOnlyWithUser(HandlerCreateGroup))
	handle(http.MethodPost, "/api4debtus/group-get", api4auth.AuthOnlyWithUser(HandlerGetGroup))
	handle(http.MethodPost, "/api4debtus/group-update", api4auth.AuthOnly(HandlerUpdateGroup))
	handle(http.MethodPost, "/api4debtus/group-delete", api4auth.AuthOnly(HandlerDeleteGroup))
	handle(http.MethodPost, "/api4debtus/group-set-contacts", api4auth.AuthOnlyWithUser(HandlerSetContactsToGroup))
	handle(http.MethodPost, "/api4debtus/join-groups", api4auth.AuthOnly(HandleJoinGroups))

	handle(http.MethodGet, "/api4debtus/user/data/*rest", api4auth.AuthOnly(HandleGetUserData))
	handle(http.MethodGet, "/api4debtus/user/currencies", api4auth.AuthOnlyWithUser(HandleGetUserCurrencies))
	handle(http.MethodGet, "/api4debtus/user", HandleUserInfo)

	handle(http.MethodGet, "/api4debtus/me", api4auth.AuthOnlyWithUser(HandleMe))
	handle(http.MethodPost, "/api4debtus/user-set-name", api4auth.AuthOnly(SetUserName))

	handle(http.MethodGet, "/api4debtus/admin/latest/users", api4auth.AdminOnly(HandleAdminLatestUsers))
	handle(http.MethodPost, "/api4debtus/admin/find-user", api4auth.AdminOnly(HandleAdminFindUser))
	handle(http.MethodGet, "/api4debtus/admin/merge-user-contacts", api4auth.AdminOnly(HandleAdminMergeUserContacts))

	handle(http.MethodPost, "/api4debtus/analytics/visitor", HandleSaveVisitorData)
	handle(http.MethodPost, "/api4debtus/invite/create", CreateInvite)
}
