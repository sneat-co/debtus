package dtb_transfer

import (
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/strongo/analytics"
)

func reportReminderIsActed(whc botsfw.WebhookContext, action string) {
	whc.Analytics().Enqueue(analytics.NewEvent("reminder-acted", "reminders", action))
}
