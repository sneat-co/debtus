package dtb_transfer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/analytics4bots"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"

	"github.com/bots-go-framework/bots-fw-telegram/telegram"
)

//func InlineAcceptTransfer(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
//	inlineQuery := whc.InputInlineQuery()
//	m.TelegramInlineCongig = &tgbotapi.InlineConfig{
//		InlineQueryID: inlineQuery.GetInlineQueryID(),
//		SwitchPMText: "Accept transfer",
//		SwitchPMParameter: "accept?transfer=ABC",
//	}
//	return m, err
//}

const createReceiptIfNoInlineNotificationCommandCode = "create_receipt"

var createReceiptIfNoInlineNotificationCommand = botsfw.Command{
	Code:       createReceiptIfNoInlineNotificationCommandCode,
	InputTypes: []botinput.Type{botinput.TypeCallbackQuery},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return OnInlineChosenCreateReceipt(whc, whc.Input().(telegram.WebhookCallbackQuery).GetInlineMessageID(), callbackUrl)
	},
}

func InlineSendReceipt(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "InlineSendReceipt()")
	inlineQuery := whc.Input().(botinput.InlineQuery)
	query := inlineQuery.GetQuery()
	values, err := url.ParseQuery(query[len("receipt?"):])
	if err != nil {
		return m, err
	}
	idParam := values.Get("id")
	if cleanID := strings.Trim(idParam, " \",.;!@#$%^&*(){}[]`~?/\\|"); cleanID != idParam {
		logus.Debugf(ctx, "Unclean receipt ContactID: %v, cleaned: %v", idParam, cleanID)
		idParam = cleanID
	}
	transferID := idParam
	if transferID == "" {
		return m, fmt.Errorf("missing transfer ContactID")
	}
	var transfer models4debtus.TransferEntry
	transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, nil, transferID)
	if err != nil {
		logus.Infof(ctx, "Faield to get transfer by ContactID: %v", transferID)
		return m, err
	}

	logus.Debugf(ctx, "Loaded transfer: %v", transfer)
	creator := whc.Input().GetSender()

	m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.GetInlineQueryID(),
		//SwitchPmText: "Accept invite",
		//SwitchPmParameter: "invite?code=ABC",
		Results: []tgbotapi.InlineQueryResult{
			tgbotapi.InlineQueryResultArticle{
				InlineQueryResultBase: tgbotapi.InlineQueryResultBase{
					ID:    query,
					Type:  tgbotapi.InlineQueryResultTypeArticle, // TODO: Move to constructor
					Title: fmt.Sprintf(whc.Translate(trans.INLINE_RECEIPT_TITLE), transfer.Data.GetAmount()),
					ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
							{
								{
									Text:         whc.Translate(trans.COMMAND_TEXT_WAIT_A_SECOND),
									CallbackData: fmt.Sprintf("%s?id=%s", createReceiptIfNoInlineNotificationCommandCode, transferID),
								},
							},
						},
					},
				},
				ThumbURL:    "https://debtus.app/img/debtus-512x512.png", //TODO: Replace with receipt image
				ThumbHeight: 512,
				ThumbWidth:  512,
				Description: whc.Translate(trans.INLINE_RECEIPT_DESCRIPTION),
				InputMessageContent: tgbotapi.InputTextMessageContent{
					MessageText: getInlineReceiptMessageText(whc, whc.GetBotCode(), whc.Locale().Code5, fmt.Sprintf("%v %v", creator.GetFirstName(), creator.GetLastName()), ""),
					ParseMode:   "HTML",
				},
			},
		},
	})
	logus.Debugf(ctx, "MessageFromBot: %v", m)

	//logus.Debugf(ctx, "Calling botApi.Send(inlineConfig=%v)", inlineConfig)
	//
	//botApi := &tgbotapi.BotAPI{
	//	Token:  whc.GetBotSettings().Token,
	//	Debug:  true,
	//	Client: whc.GetHTTPClient(),
	//}
	//mes, err := botApi.AnswerInlineQuery(inlineConfig)
	//if err != nil {
	//	logus.Errorf(ctx, "Failed to send inline results: %v", err)
	//}
	//s, err := json.Marshal(mes)
	//if err != nil {
	//	logus.Errorf(ctx, "Failed to marshal inline results response: %v, %v", err, mes)
	//}
	//logus.Infof(ctx, "botApi.Send(inlineConfig): %v", string(s))
	return m, err
}

func getInlineReceiptMessageText(t i18n.SingleLocaleTranslator, botCode, localeCode5, creator string, receiptID string) string {
	data := map[string]interface{}{
		"Creator":  creator,
		"SiteLink": template.HTML(`<a href="https://debtus.app/#utm_source=telegram&utm_medium=bot&utm_campaign=receipt-inline">Debtus</a>`),
	}
	if receiptID != "" {
		data["ReceiptUrl"] = GetUrlForReceiptInTelegram(botCode, receiptID, localeCode5)
	}
	var buf bytes.Buffer
	if receiptID == "" {
		buf.WriteString(t.Translate(trans.INLINE_RECEIPT_GENERATING_MESSAGE, data))
	} else {
		buf.WriteString(t.Translate(trans.INLINE_RECEIPT_MESSAGE, data))
	}

	//buf.WriteString("\n\n" + t.Translate(trans.INLINE_RECEIPT_FOOTER, data))

	if receiptID != "" {
		buf.WriteString("\n\n" + t.Translate(trans.INLINE_RECEIPT_CHOOSE_LANGUAGE, data))
	}

	return buf.String()
}

func OnInlineChosenCreateReceipt(whc botsfw.WebhookContext, inlineMessageID string, queryUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	_ = inlineMessageID

	logus.Debugf(ctx, "OnInlineChosenCreateReceipt(queryUrl: %v)", queryUrl)
	transferID := queryUrl.Query().Get("id")
	creator := whc.Input().GetSender()
	creatorName := fmt.Sprintf("%v %v", creator.GetFirstName(), creator.GetLastName())

	transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, nil, transferID)
	if err != nil {
		return m, err
	}
	receiptData := models4debtus.NewReceiptEntity(whc.AppUserID(), transferID, transfer.Data.Counterparty().UserID, whc.Locale().Code5, string(telegram.PlatformID), "", general4debtus.CreatedOn{
		CreatedOnID:       whc.GetBotCode(), // TODO: Replace with method call.
		CreatedOnPlatform: whc.BotPlatform().ID(),
	})
	receipt, err := dal4debtus.Default.Receipt.CreateReceipt(ctx, receiptData)
	if err != nil {
		return m, err
	}

	if err = dal4debtus.Default.Receipt.DelayedMarkReceiptAsSent(ctx, receipt.ID, transferID, time.Now()); err != nil {
		logus.Errorf(ctx, "Failed DelayedMarkReceiptAsSent: %v", err)
	}
	if m, err = showReceiptAnnouncement(whc, receipt.ID, creatorName); err != nil {
		return m, err
	}

	analytics4bots.ReceiptSentFromBot(whc, "telegram")

	//_, err = whc.Responder().SendMessage(ctx, m, botsfw.BotAPISendMessageOverHTTPS)
	//if err != nil {
	//	logus.Errorf(ctx, "Failed to send inline response: %v", err.Error())
	//}
	//m = whc.NewMessage("")
	return
}
