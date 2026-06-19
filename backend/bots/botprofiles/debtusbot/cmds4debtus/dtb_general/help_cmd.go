package dtb_general

import (
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func HelpCommandAction(whc botsfw.WebhookContext, showFeedbackButton bool) (m botmsg.MessageFromBot, err error) {
	var openReportUrl, bugUrl, ideaUrl string
	if openReportUrl, err = getUserReportUrl(whc, ""); err != nil {
		return
	}
	if bugUrl, err = getUserReportUrl(whc, "bug"); err != nil {
		return
	}
	if ideaUrl, err = getUserReportUrl(whc, "idea"); err != nil {
		return
	}
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{
				Text: emoji.PUBLIC_LOUDSPEAKER + " " + whc.Translate(trans.COMMAND_TEXT_OPEN_USER_REPORT),
				URL:  openReportUrl,
			},
		},
		[]tgbotapi.InlineKeyboardButton{btnSubmitBug(whc, bugUrl)},
		[]tgbotapi.InlineKeyboardButton{btnSubmitIdea(whc, ideaUrl)},
	)
	if showFeedbackButton {
		keyboardMarkup.InlineKeyboard = append(
			keyboardMarkup.InlineKeyboard,
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         emoji.STAR_ICON + " " + whc.Translate(trans.COMMAND_TEXT_ASK_FOR_FEEDBACK),
					CallbackData: FeedbackCommandCode,
				},
			})
	}
	if showFeedbackButton {
		m = whc.NewMessageByCode(trans.MESSAGE_TEXT_HELP)
		m.Keyboard = keyboardMarkup
	} else {
		if m, err = whc.NewEditMessage("", botmsg.FormatText); err != nil {
			return
		}
		m.Keyboard = keyboardMarkup
	}

	return m, err
}

var getUserReportUrl = getUserReportUrlImpl

func getUserReportUrlImpl(t i18n.SingleLocaleTranslator, submit string) (string, error) {
	switch t.Locale().Code5 {
	case i18n.LocaleCodeRuRU:
		switch submit {
		case "idea":
			return "https://goo.gl/dAKHFC", nil
		case "bug":
			return "https://goo.gl/jQM2K5", nil
		case "":
			return "https://goo.gl/Vge31X", nil
		}
	default:
		switch submit {
		case "idea":
			return "https://goo.gl/sl09Wr", nil
		case "bug":
			return "https://goo.gl/x5H6Fn", nil
		case "":
			return "https://goo.gl/3tB0FG", nil
		}
	}
	return "", fmt.Errorf("parameter 'submit' should be either 'idea', 'bug' or empty string, got: %q", submit)
}

func btnSubmitIdea(whc botsfw.WebhookContext, url string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.InlineKeyboardButton{
		Text: emoji.BULB_ICON + " " + whc.Translate(trans.COMMAND_TEXT_SUBMIT_AN_IDEA),
		URL:  url,
	}
}

func btnSubmitBug(whc botsfw.WebhookContext, url string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.InlineKeyboardButton{
		Text: emoji.ERROR_ICON + " " + whc.Translate(trans.COMMAND_TEXT_REPORT_A_BUG),
		URL:  url,
	}
}

const adsCommandCode = "ads"

var adsCommand = botsfw.Command{
	Code:       adsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Icon:       emoji.NEWSPAPER_ICON,
	Commands:   []string{emoji.NEWSPAPER_ICON, "/ads", "/реклама"},
	Title:      trans.COMMAND_TEXT_HELP,
	Titles:     map[string]string{botsfw.ShortTitle: ""},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		chatData := whc.ChatData()

		yesOption := emoji.PHONE_ICON + " " + whc.Translate(trans.COMMAND_TEXT_SUBSCRIBE_TO_APP)
		noOption := whc.Translate(trans.COMMAND_TEXT_I_AM_FINE_WITH_BOT)
		if chatData.GetAwaitingReplyTo() == "" {
			chatData.SetAwaitingReplyTo(adsCommandCode)
			m = whc.NewMessage(emoji.NEWSPAPER_ICON + " " + whc.Translate(trans.MESSAGE_TEXT_YOUR_ABOUT_ADS))
			m.DisableWebPagePreview = true
			m.Keyboard = tgbotapi.NewReplyKeyboard(
				[]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(yesOption)},
				[]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(noOption)},
				[]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(MainMenuCommand.DefaultTitle(whc))},
			)
		} else {
			switch whc.Input().(botinput.TextMessage).Text() {
			case yesOption:
				m = whc.NewMessageByCode(trans.MESSAGE_TEXT_SUBSCRIBED_TO_APP)
				if err = SetMainMenuKeyboard(whc, &m); err != nil {
					return
				}
				chatData.SetAwaitingReplyTo("")
			case noOption:
				m = whc.NewMessageByCode(trans.MESSAGE_TEXT_NOT_INTERESTED_IN_APP)
				if err = SetMainMenuKeyboard(whc, &m); err != nil {
					return
				}
				chatData.SetAwaitingReplyTo("")
			default:
				m = whc.NewMessageByCode(trans.MESSAGE_TEXT_PLEASE_CHOOSE_FROM_OPTIONS_PROVIDED)
			}
		}
		return m, err
	},
}
