package cmds4debtus

import (
	"strings"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_inline"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_transfer"
	"github.com/strongo/logus"
)

// inlineSendReceipt is a seam for dtb_transfer.InlineSendReceipt.
var inlineSendReceipt = dtb_transfer.InlineSendReceipt

// inlineNewRecord is a seam for dtb_inline.InlineNewRecord.
var inlineNewRecord = dtb_inline.InlineNewRecord

func inlineQueryHandler(whc botsfw.WebhookContext, inlineQuery botinput.InlineQuery) (handled bool, m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()

	query := inlineQuery.GetQuery()
	logus.Debugf(ctx, "inlineQueryCommand.Action(query=%v)", query)
	switch {
	case strings.HasPrefix(query, "receipt?id="):
		m, err = inlineSendReceipt(whc)
		handled = true
	//case strings.HasPrefix(query, "accept?transfer="):
	//	m, err = dtb_transfer.InlineAcceptTransfer(whc)
	default:
		amountMatches := dtb_inline.ReInlineQueryAmount.FindStringSubmatch(query)
		if amountMatches != nil {
			if m, err = inlineNewRecord(whc, amountMatches); err != nil {
				return
			}
			handled = true
		}
		logus.Debugf(ctx, "Inline query not matched to any action: [%v]", query)
	}
	return
}
