package analytics4bots

import (
	"strings"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/strongo/analytics"
)

func ReceiptSentFromBot(whc botsfw.WebhookContext, channel string) {
	whc.Analytics().Enqueue(analytics.NewEvent("receipt-sent", "receipt-sent", "update-receipt-sent").SetLabel(strings.ToLower(channel)))
}
