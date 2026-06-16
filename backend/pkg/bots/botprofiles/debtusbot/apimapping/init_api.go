package apimapping

import (
	"github.com/sneat-co/sneat-core-modules/auth/api4auth"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/api4unsorted"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/api4reminders"
	"github.com/strongo/strongoapp"
)

func InitApi(handle strongoapp.HandleHttpWithContext) {
	api4auth.InitApiForAuth(handle)
	api4unsorted.InitApiForUnsorted(handle)
	api4reminders.InitApiForReminder(handle)
}
