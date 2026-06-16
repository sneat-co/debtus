package dtb_transfer

import (
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
)

const acknowledgeReceiptCallbackCommandCode = "ack_receipt"

var acknowledgeReceiptCallbackCommand = botsfw.NewCallbackCommand(acknowledgeReceiptCallbackCommandCode, func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	query := callbackUrl.Query()
	receiptID := query.Get("id")
	if receiptID == "" {
		return m, fmt.Errorf("receiptID is empty")
	}

	return AcknowledgeReceipt(whc, receiptID, query.Get("do"))
})
