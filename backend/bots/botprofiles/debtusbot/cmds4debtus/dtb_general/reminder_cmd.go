package dtb_general

import (
	"fmt"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/utm4bots"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/utmconsts"
	"github.com/sneat-co/sneat-translations/trans"
)

var textReceiptForTransfer = common4debtus.TextReceiptForTransfer

func EditReminderMessage(whc botsfw.WebhookContext, transfer models4debtus.TransferEntry, message string) (m botmsg.MessageFromBot, err error) {
	utm := utm4bots.NewParams(whc, utmconsts.UTM_CAMPAIGN_REMINDER)
	appUserID := whc.AppUserID()
	mt := fmt.Sprintf(
		"<b>%v</b>\n%v\n\n%v",
		whc.Translate(trans.MESSAGE_TEXT_REMINDER),
		textReceiptForTransfer(whc.Context(), whc, transfer, appUserID, common4debtus.ShowReceiptToAutodetect, utm),
		message,
	)
	if whc.Input().InputType() == botinput.TypeCallbackQuery {
		if m, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
			return
		}
	} else {
		m = whc.NewMessage(mt)
		m.Format = botmsg.FormatHTML
		if err = SetMainMenuKeyboard(whc, &m); err != nil {
			return
		}
	}

	return
}
