package dtb_inline

import (
	"fmt"
	"html"
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"

	//"fmt"
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-translations/trans"

	//"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/decimal"
	"github.com/strongo/logus"

	//"html"
	//"net/url"
	"regexp"
	"strings"
)

var ReInlineQueryAmount = regexp.MustCompile(`^\s*(\d+(?:\.\d*)?)\s*((?:\b|\B).+?)?\s*$`)

func InlineNewRecord(whc botsfw.WebhookContext, amountMatches []string) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "InlineNewRecord()")

	inlineQuery := whc.Input().(botinput.InlineQuery)
	var (
		amountValue    decimal.Decimal64p2
		amountCurrency money.CurrencyCode
	)
	if amountValue, err = decimal.ParseDecimal64p2(strings.TrimRight(amountMatches[1], ".")); err != nil {
		return
	}
	currencyCode := strings.TrimRight(amountMatches[2], ".,;()[]{} ")
	logus.Debugf(ctx, "currencyCode: %v", currencyCode)
	if currencyCode != "" {
		if len(currencyCode) > 20 {
			currencyCode = currencyCode[:20]
		}

		switch ccLow := strings.ToLower(currencyCode); ccLow {
		case money.CurrencySymbolRUR, "р", "руб", "рубля", "рублей", "rub", "rubles", "ruble", "rubley":
			amountCurrency = money.CurrencySymbolRUR
		case "eur", "euro", money.CurrencySymbolEUR:
			amountCurrency = money.CurrencyEUR
		case "гривна", "гривен", "г", money.CurrencySymbolUAH:
			amountCurrency = money.CurrencyUAH
		case "тенге", "теңге", "т", money.CurrencySymbolKZT:
			amountCurrency = money.CurrencyKZT
		default:
			amountCurrency = money.CurrencyCode(currencyCode)
		}
	} else {
		amountCurrency = money.CurrencyUSD
	}

	amountText := html.EscapeString(money.NewAmount(amountCurrency, amountValue).String())

	newBillCallbackData := fmt.Sprintf("new-bill?v=%v&ctx=%v", amountMatches[1], url.QueryEscape(string(amountCurrency)))
	m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.GetInlineQueryID(),
		Results: []tgbotapi.InlineQueryResult{
			tgbotapi.InlineQueryResultArticle{
				InlineQueryResultBase: tgbotapi.InlineQueryResultBase{
					ID:    "SplitBill_" + whc.Locale().Code5,
					Type:  "article",
					Title: "🛒 " + whc.Translate(trans.ARTICLE_TITLE_SPLIT_BILL),
					ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
							{
								{Text: whc.Translate(trans.COMMAND_TEXT_I_PAID), CallbackData: newBillCallbackData + "&i=paid"},
								{Text: whc.Translate(trans.COMMAND_TEXT_I_OWE), CallbackData: newBillCallbackData + "&i=owe"},
							},
						},
					},
				},
				Description: whc.Translate(trans.ARTICLE_SUBTITLE_SPLIT_BILL, amountText),
				InputMessageContent: tgbotapi.InputTextMessageContent{
					MessageText:           whc.Translate(trans.MESSAGE_TEXT_BILL_HEADER, amountText),
					ParseMode:             "HTML",
					DisableWebPagePreview: true,
				},
			},
			tgbotapi.InlineQueryResultArticle{
				InlineQueryResultBase: tgbotapi.InlineQueryResultBase{
					ID:    "NewDebt_" + whc.Locale().Code5,
					Type:  "article",
					Title: "💵 " + whc.Translate(trans.ARTICLE_NEW_DEBT_TITLE),
					ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
							{
								{Text: whc.Translate(trans.COMMAND_TEXT_I_OWE), CallbackData: "i-owed?debt=new"},
								{Text: whc.Translate(trans.COMMAND_TEXT_OWED_TO_ME), CallbackData: "owed2me?debt=new"},
							},
						},
					},
				},
				Description: whc.Translate(trans.ARTICLE_NEW_DEBT_SUBTITLE, amountText),
				InputMessageContent: tgbotapi.InputTextMessageContent{
					MessageText:           whc.Translate(trans.MESSAGE_TEXT_NEW_DEBT_HEADER, amountText),
					ParseMode:             "HTML",
					DisableWebPagePreview: true,
				},
			},
		},
	})
	return m, err
}
