package admin

import (
	"context"
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

func SendFeedbackToAdmins(ctx context.Context, botToken string, feedback models4debtus.Feedback) (err error) {
	bot := tgbotapi.NewBotAPIWithClient(botToken, dal4debtus.Default.HttpClient(ctx))
	text := fmt.Sprintf("%v user #%s @%v (rate=%v):\n%v", feedback.CreatedOnPlatform, feedback.UserStrID, feedback.CreatedOnID, feedback.Rate, feedback.Text)
	message := tgbotapi.NewMessageToChannel("-1001128307094", text)
	message.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{Text: "Reply to feedback", URL: fmt.Sprintf("https://debtus.app/app/#/reply-to-feedback/%s", feedback.ID)},
		},
	)
	_, err = bot.Send(message)
	return
}
