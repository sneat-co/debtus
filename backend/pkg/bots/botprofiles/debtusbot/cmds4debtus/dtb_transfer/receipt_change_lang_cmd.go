package dtb_transfer

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"

	"errors"
)

const changeReceiptLangCommandCode = "change_lang_receipt"

var changeReceiptAnnouncementLangCommand = botsfw.NewCallbackCommand(
	changeReceiptLangCommandCode,
	func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		query := callbackUrl.Query()
		code5 := query.Get("locale")
		if len(code5) != 5 {
			return m, errors.New("changeReceiptAnnouncementLangCommand: len(code5) != 5")
		}
		if err = whc.SetLocale(code5); err != nil {
			return m, err
		}
		receiptID := query.Get("id")
		if err != nil {
			return m, err
		}
		return showReceiptAnnouncement(whc, receiptID, "")
	},
)
