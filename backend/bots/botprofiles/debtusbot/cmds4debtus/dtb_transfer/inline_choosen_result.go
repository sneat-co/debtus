package dtb_transfer

import (
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/anybot/inlinekeyboards"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/sneat-translations/trans"
)

func showReceiptAnnouncement(whc botsfw.WebhookContext, receiptID string, creatorName string) (m botmsg.MessageFromBot, err error) {
	var inlineMessageID string
	switch input := whc.Input().(type) {
	case botinput.ChosenInlineResult:
		inlineMessageID = input.GetInlineMessageID()
	case telegram.WebhookCallbackQuery:
		inlineMessageID = input.GetInlineMessageID()
	default:
		return m, fmt.Errorf("showReceiptAnnouncement: Unsupported InputType=%T", input)
	}

	ctx := whc.Context()

	receipt, err := dal4debtus.Default.Receipt.GetReceiptByID(ctx, nil, receiptID)
	if err != nil {
		return m, err
	}
	if creatorName == "" {
		user, err := dal4userus.GetUserByID(ctx, nil, receipt.Data.CreatorUserID)
		if err != nil {
			return m, err
		}
		creatorName = user.Data.Names.GetFullName()
	}

	messageText := getInlineReceiptMessageText(whc, whc.GetBotCode(), whc.Locale().Code5, creatorName, receiptID)
	m, err = whc.NewEditMessage(messageText, botmsg.FormatHTML)
	m.EditMessageUID = telegram.NewInlineMessageUID(inlineMessageID)
	m.DisableWebPagePreview = true
	kbRows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(
				whc.Translate(trans.COMMAND_TEXT_VIEW_RECEIPT_DETAILS),
				fmt.Sprintf("%s?id=%s&locale=%s",
					viewReceiptInTelegramCommandCode, receiptID, whc.Locale().Code5,
				),
			),
		},
	}
	kbRows = append(kbRows, inlinekeyboards.GetChooseLangInlineKeyboard(
		fmt.Sprintf("%s?id=%s", changeReceiptLangCommandCode, receiptID)+"&locale=%v", // Intentionally &locale separate
		whc.Locale().Code5,
	)...)
	m.Keyboard = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: kbRows,
	}
	return
}

const viewReceiptInTelegramCommandCode = "tg_view_receipt"

func GetUrlForReceiptInTelegram(botCode string, receiptID string, localeCode5 string) string {
	return bothelper.StartTelegramBotUrl(botCode, "receipt", "id="+receiptID, "lang="+localeCode5)
}
