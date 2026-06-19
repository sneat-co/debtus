package cmds4splitusbot

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"

	"errors"

	"github.com/bots-go-framework/bots-fw-telegram/telegram"
)

var reInlineQueryNewBill = regexp.MustCompile(`^\s*(\d+(?:\.\d*)?)([^\s]*)\s+(.+?)\s*$`)

func InlineQueryHandler(whc botsfw.WebhookContext, inlineQuery botinput.InlineQuery) (handled bool, m botmsg.MessageFromBot, err error) {
	whc.Input().LogRequest()
	ctx := whc.Context()
	if tgInput, ok := whc.Input().(telegram.TgWebhookInput); ok {
		update := tgInput.TgUpdate()

		var appUserData botsfwmodels.AppUserData
		if appUserData, err = whc.AppUserData(); err != nil {
			return
		} else if preferredLocale := appUserData.BotsFwAdapter().GetPreferredLocale(); preferredLocale != "" {
			logus.Debugf(ctx, "User has preferring locale")
			_ = whc.SetLocale(preferredLocale)
		} else if tgLang := update.InlineQuery.From.LanguageCode; len(tgLang) >= 2 {
			switch strings.ToLower(tgLang[:2]) {
			case "ru":
				logus.Debugf(ctx, "Telegram client has known language code")
				if err = whc.SetLocale(i18n.LocaleCodeRuRU); err != nil {
					return
				}
			}
		}
	}
	query := strings.TrimSpace(inlineQuery.GetQuery())
	logus.Debugf(ctx, "inlineQueryCommand.Action(query=%v)", query)
	switch {
	case query == "":
		m, err = inlineEmptyQuery(whc, inlineQuery)
		handled = true
		return
	case strings.HasPrefix(query, joinSpaceCommandCode+"?id="):
		m, err = inlineQueryJoinGroup(whc, query)
		handled = true
		return
	default:
		if reMatches := reInlineQueryNewBill.FindStringSubmatch(query); reMatches != nil {
			m, err = inlineQueryNewBill(whc, reMatches[1], reMatches[2], reMatches[3])
			handled = true
			return
		}
		logus.Debugf(ctx, "Inline query not matched to any action: [%v]", query)
	}
	return
}

func inlineEmptyQuery(whc botsfw.WebhookContext, inlineQuery botinput.InlineQuery) (m botmsg.MessageFromBot, err error) {
	logus.Debugf(whc.Context(), "InlineEmptyQuery()")
	m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.GetInlineQueryID(),
		CacheTime:     60,
		//SwitchPMText:      "Help: How to use this bot?",
		//SwitchPMParameter: "help_inline",
	})
	return
}

func inlineQueryJoinGroup(whc botsfw.WebhookContext, query string) (m botmsg.MessageFromBot, err error) {
	err = errors.New("inlineQueryJoinGroup is not implemented")
	_, _ = whc, query
	//ctx := whc.Context()
	//
	//inlineQuery := whc.Input().(botsfw.WebhookInlineQuery)
	//
	//var group models4splitus.GroupEntry
	//if group.ContactID = query[len(joinSpaceCommandCode+"?id="):]; group.ContactID == "" {
	//	err = errors.New("Missing group ContactID")
	//	return
	//}
	//if group, err = dal4debtus.Group.GetGroupByID(ctx, nil, group.ContactID); err != nil {
	//	return
	//}
	//
	//joinBillInlineResult := tgbotapi.InlineQueryResultArticle{
	//	ContactID:          query,
	//	Type:        "article",
	//	Title:       "Send invite for joining",
	//	Description: "group: " + group.Data.UserTitle,
	//	InputMessageContent: tgbotapi.InputTextMessageContent{
	//		Text:      fmt.Sprintf("I'm inviting you to join <b>bills sharing</b> group @%v.", whc.GetBotCode()),
	//		ParseMode: "HTML",
	//	},
	//	ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
	//		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
	//			{
	//				{
	//					Text:         "Join",
	//					CallbackData: query,
	//				},
	//			},
	//		},
	//	},
	//}
	//
	//m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
	//	InlineQueryID: inlineQuery.GetInlineQueryID(),
	//	CacheTime:     60,
	//	Results: []interface{}{
	//		joinBillInlineResult,
	//	},
	//})
	return
}

func inlineQueryNewBill(whc botsfw.WebhookContext, amountNum, amountCurr, billName string) (m botmsg.MessageFromBot, err error) {
	if len(amountCurr) == 3 {
		amountCurr = strings.ToUpper(amountCurr)
	}

	m.Text = fmt.Sprintf("Amount: %v %v, BillEntry name: %v", amountNum, amountCurr, billName)

	inlineQuery := whc.Input().(botinput.InlineQuery)

	params := fmt.Sprintf("amount=%v&lang=%v", url.QueryEscape(amountNum+amountCurr), whc.Locale().Code5)

	resultID := "bill?" + params

	newBillInlineResult := tgbotapi.InlineQueryResultArticle{
		InlineQueryResultBase: tgbotapi.InlineQueryResultBase{
			ID:    resultID,
			Type:  "article",
			Title: fmt.Sprintf("%v: %v", whc.Translate(trans.COMMAND_TEXT_NEW_BILL), billName),
			ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						{
							Text:         whc.Translate(trans.MESSAGE_TEXT_PLEASE_WAIT),
							CallbackData: "creating-bill?" + params,
						},
					},
				},
			},
		},
		Description: fmt.Sprintf("%v: %v %v", whc.Translate(trans.HTML_AMOUNT), amountNum, amountCurr),
		InputMessageContent: tgbotapi.InputTextMessageContent{
			MessageText: fmt.Sprintf("<b>%v</b>: %v %v - %v",
				whc.Translate(trans.MESAGE_TEXT_CREATING_BILL),
				html.EscapeString(amountNum),
				html.EscapeString(amountCurr),
				html.EscapeString(billName),
			),
			ParseMode: "HTML",
		},
	}

	m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.GetInlineQueryID(),
		CacheTime:     60,
		Results: []tgbotapi.InlineQueryResult{
			newBillInlineResult,
		},
	})

	return
}
