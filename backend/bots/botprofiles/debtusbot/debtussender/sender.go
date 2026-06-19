package debtussender

import (
	"errors"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

func SendRefreshOrNothingChanged(whc botsfw.WebhookContext, m botmsg.MessageFromBot) (m2 botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	if _, err = whc.Responder().SendMessage(ctx, m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
		logus.Debugf(ctx, "error type: %T", err)
		var apiResponse tgbotapi.APIResponse
		if errors.As(err, &apiResponse) && apiResponse.ErrorCode == 400 {
			m.BotMessage = telegram.CallbackAnswer(tgbotapi.NewCallback("", whc.Translate(trans.ALERT_TEXT_NOTHING_CHANGED)))
			err = nil
		}
	}
	return m, err
}
