package dtb_general

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/admin"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/trans4debtus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

const (
	FeedbackCommandCode = "feedback"
	//FEEDBACK_UNDECIDED  = "undecided"
)

var runReadwriteTransaction = facade.RunReadwriteTransaction

var saveFeedback = facade4debtus.SaveFeedback

var getFeedbackByID = func(ctx context.Context, tx dal.ReadSession, feedbackID string) (models4debtus.Feedback, error) {
	return dal4debtus.Default.Feedback.GetFeedbackByID(ctx, tx, feedbackID)
}

var sendFeedbackToAdmins = admin.SendFeedbackToAdmins

/*
var FeedbackCallbackCommand = botsfw.NewCallbackCommand(FeedbackCommandCode, func(whc botsfw.WebhookContext, callbackUrl *url.URL) (botmsg.MessageFromBot, error) {
	return feedbackCommand.Action(whc)
})

var feedbackCommand = botsfw.Command{
	Code:     FeedbackCommandCode,
	Commands: trans.Commands(trans.COMMAND_TEXT_FEEDBACK),
	Title:    trans.COMMAND_TEXT_HIGH_FIVE,
	Icon:     emoji.STAR_ICON,
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		feedbackCommand.Action(whc)
		chatEntity := whc.ChatData()
		switch chatEntity.GetAwaitingReplyTo() {
		//case "":
		//	return showFeedbackOptions(whc, chatEntity)
		case FeedbackCommandCode:
			mt := whc.Input().(botsfw.WebhookTextMessage).Text()
			words := strings.SplitN(mt, " ", 2)
			feedbackEntity := models.FeedbackData{
				UserID: whc.AppUserID(),
			}
			//mainMenuButton := []tgbotapi.InlineKeyboardButton{
			//	{
			//		Text: whc.CommandText(trans.COMMAND_TEXT_MAIN_MENU_TITLE, emoji.MAIN_MENU_ICON),
			//		CallbackData: mainMenuCommandCode,
			//	},
			//}

			switch words[0] {
			case emoji.EMO_SMILING_ICON:
				feedbackEntity.Rate = "Positive"
				thankYouText := strings.Replace(
					whc.Translate(trans.MESSAGE_TEXT_ON_FEEDBACK_POSITIVE),
					fmt.Sprintf("{{%v}}", trans.MESSAGE_TEXT_YOU_CAN_HELP_BY),
					YouCanHelp(whc, trans.MESSAGE_TEXT_YOU_CAN_HELP_BY, whc.GetBotCode()),
					1)
				if thankYouText, err = FeedbackLinks(whc, thankYouText); err != nil {
					return
				}
				m = whc.NewMessage(emoji.EMO_SMILING_RED_CHEEKS + " " + thankYouText + "\n" + AskToTranslate(whc))
			case emoji.EMO_NEUTRAL:
				feedbackEntity.Rate = "Neutral"
				var text, ideaUrl, bugUrl string
				if text, err = FeedbackLinks(whc, whc.Translate(trans.MESSAGE_TEXT_ON_FEEDBACK_NEUTRAL)); err != nil {
					return
				}
				if ideaUrl, err = getUserReportUrl(whc, "idea"); err != nil {
					return
				}
				if bugUrl, err = getUserReportUrl(whc, "bug"); err != nil {
					return
				}
				m = whc.NewMessage(emoji.EMO_CONFUSED + " " + text + "\n\n" + AskToTranslate(whc))
				m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
					[]tgbotapi.InlineKeyboardButton{btnSubmitIdea(whc, ideaUrl)},
					[]tgbotapi.InlineKeyboardButton{btnSubmitBug(whc, bugUrl)},
					//mainMenuButton,
				)
			case emoji.EMO_ANGRY_ICON:
				feedbackEntity.Rate = "Angry"
				var text, ideaUrl, bugUrl string
				if text, err = FeedbackLinks(whc, whc.Translate(trans.MESSAGE_TEXT_ON_FEEDBACK_NEGATIVE)); err != nil {
					return
				}
				if bugUrl, err = getUserReportUrl(whc, "bug"); err != nil {
					return
				}
				if ideaUrl, err = getUserReportUrl(whc, "idea"); err != nil {
					return
				}
				m = whc.NewMessage(emoji.EMO_EMBARRASSED + " " + text)
				m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
					[]tgbotapi.InlineKeyboardButton{btnSubmitBug(whc, bugUrl)},
					[]tgbotapi.InlineKeyboardButton{btnSubmitIdea(whc, ideaUrl)},
					//mainMenuButton,
				)
			case emoji.EMO_THINKING:
				feedbackEntity.Rate = FEEDBACK_UNDECIDED
			default:
				m = whc.NewMessage(whc.Translate(trans.MESSAGE_TEXT_PLEASE_CHOOSE_FROM_OPTIONS_PROVIDED))
				m.Keyboard = feedbackOptionsTelegramKeyboard(whc)
				return m, nil
			}
			m.DisableWebPagePreview = true

			ctx := whc.Context()
			whc.GetAppUser()
			if _, _, err = facade4debtus.SaveFeedback(c, &feedbackEntity); err != nil {
				return m, errors.Wrap(err, "Failed to save Feedback to DB")
			}
			if feedbackEntity.Rate == FEEDBACK_UNDECIDED {
				return MainMenuAction(whc, "", false)
			} else {
				//if _, err = whc.Responder().SendMessage(c, m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
				//	return m, err
				//}
				//m = whc.NewMessageByCode(trans.MESSAGE_TEXT_BACK_TO_MAIN_MENU)
				m.Keyboard = tgbotapi.NewReplyKeyboard(
					[]tgbotapi.KeyboardButton{{Text: whc.CommandText(trans.COMMAND_TEXT_MAIN_MENU_TITLE, emoji.MAIN_MENU_ICON)}},
				)
				return m, err
			}
		default:
			return showFeedbackOptions(whc, chatEntity)
		}
	},
	CallbackAction: func(whc botsfw.WebhookContext, _ *url.URL) (m botmsg.MessageFromBot, err error) {
		m, err = showFeedbackOptions(whc, whc.ChatData())
		if _, err = whc.Responder().SendMessage(whc.Context(), m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
			return m, err
		}
		return HelpCommandAction(whc, false)
	},
}
*/

func feedbackCommandAction(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	m = whc.NewMessageByCode(trans.MESSAGE_TEXT_DO_YOU_LIKE_OUR_BOT)
	m.Text = strings.Replace(m.Text, "{{bot}}", whc.GetBotCode(), 1)
	m.Keyboard = feedbackOptionsTelegramKeyboard(whc)
	return m, err
}

var feedbackCommand = botsfw.Command{
	Code:           FeedbackCommandCode,
	InputTypes:     []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Title:          trans.COMMAND_TEXT_FEEDBACK,
	Commands:       trans.Commands(trans.COMMAND_TEXT_FEEDBACK, FeedbackCommandCode, emoji.STAR_ICON),
	Icon:           emoji.STAR_ICON,
	Action:         feedbackCommandAction,
	CallbackAction: feedbackCommandCallbackAction,
}

func feedbackCommandCallbackAction(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	like := callbackUrl.Query().Get("like")
	if like == "" {
		m, err = feedbackCommandAction(whc)
		return
	}
	feedbackEntity := models4debtus.FeedbackData{
		UserStrID: whc.AppUserID(),
		CreatedOn: general4debtus.CreatedOn{
			CreatedOnPlatform: whc.BotPlatform().ID(),
			CreatedOnID:       whc.GetBotCode(),
		},
	}
	switch like {
	case "yes":
		feedbackEntity.Rate = "like"
	case "no":
		feedbackEntity.Rate = "dislike"
	default:
		err = fmt.Errorf("unexpected 'like' value: %v", like)
		return
	}
	var feedback models4debtus.Feedback
	if err = runReadwriteTransaction(whc.Context(), func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if feedback, _, err = saveFeedback(ctx, tx, "", &feedbackEntity); err != nil {
			return
		}
		return nil
	}, dal.TxWithCrossGroup()); err != nil {
		return
	}
	switch like {
	case "yes":
		m, err = askIfCanRateAtStoreBot(whc)
	case "no":
		m, err = askToWriteFeedback(whc, feedback.ID)
	}
	return
}

func feedbackOptionsTelegramKeyboard(whc botsfw.WebhookContext) *tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_YES_EXCLAMATION, emoji.GREEN_CHECKBOX), CallbackData: FeedbackCommandCode + "?like=yes"},
			{Text: whc.Translate(trans.COMMAND_TEXT_NOT_TOO_MUCH, emoji.CROSS_MARK), CallbackData: FeedbackCommandCode + "?like=no"},
		},
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_WRITE_FEEDBACK, emoji.MEMO_ICON), CallbackData: FeedbackTextCommandCode},
		},
	)
}

func askIfCanRateAtStoreBot(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	m, err = editTelegramMessageText(whc, "", whc.Translate(trans.MESSAGE_TEXT_CAN_YOU_RATE_AT_STOREBOT))
	m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_YES, emoji.GREEN_CHECKBOX), CallbackData: canYouRateCommandCode + "?will-rate=yes"},
			{Text: whc.Translate(trans.COMMAND_TEXT_NO, emoji.CROSS_MARK), CallbackData: canYouRateCommandCode + "?will-rate=no"},
		},
	)
	return
}

const canYouRateCommandCode = "can_you_rate"

var canYouRateCommand = botsfw.Command{
	Code: canYouRateCommandCode,
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		logus.Debugf(whc.Context(), "canYouRateCommand.CallbackAction): whc.ChatData().GetPreferredLanguage()=%v", whc.ChatData().GetPreferredLanguage())
		if callbackUrl == nil || callbackUrl.RawQuery == "" {
			m, err = askIfCanRateAtStoreBot(whc)
		} else {
			switch callbackUrl.Query().Get("will-rate") {
			case "yes":
				m, err = editTelegramMessageText(whc, "", strings.Replace(whc.Translate(trans.MESSAGE_TEXT_HOW_TO_RATE_AT_STOREBOT), "{{bot}}", whc.GetBotCode(), 1))
			case "no":
				thankYouText := strings.Replace(
					whc.Translate(trans.MESSAGE_TEXT_ON_REFUSED_TO_RATE),
					fmt.Sprintf("{{%v}}", trans.MESSAGE_TEXT_YOU_CAN_HELP_BY),
					trans4debtus.YouCanHelp(whc, trans.MESSAGE_TEXT_YOU_CAN_HELP_BY, whc.GetBotCode()),
					1)
				if thankYouText, err = FeedbackLinks(whc, thankYouText); err != nil {
					return
				}
				if m, err = editTelegramMessageText(whc, "/", thankYouText); err != nil {
					return
				}
				m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
					[]tgbotapi.InlineKeyboardButton{
						{Text: whc.Translate(trans.COMMAND_TEXT_WRITE_FEEDBACK, emoji.MEMO_ICON), CallbackData: FeedbackTextCommandCode},
					},
					[]tgbotapi.InlineKeyboardButton{
						{Text: emoji.MAIN_MENU_ICON + " " + whc.Translate(trans.COMMAND_TEXT_MAIN_MENU_TITLE), CallbackData: mainMenuCommandCode},
					},
				)
			default:
				m = whc.NewMessage(fmt.Sprintf("Unknown 'will-rate' value, expected yes/no, got: %v", callbackUrl.Query().Get("reply")))
				logus.Errorf(whc.Context(), m.Text)
			}
		}
		return
	},
}

func askToWriteFeedback(whc botsfw.WebhookContext, feedbackID string) (m botmsg.MessageFromBot, err error) {
	m = whc.NewMessageByCode(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER)
	//m, err = editTelegramMessageText(whc, FeedbackTextCommandCode, whc.Translate(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER))
	whc.ChatData().SetAwaitingReplyTo(FeedbackTextCommandCode)
	if feedbackID != "" {
		whc.ChatData().AddWizardParam("feedback", feedbackID)
	}
	m.Keyboard = tgbotapi.NewHideKeyboard(false)
	return
}

func editTelegramMessageText(whc botsfw.WebhookContext, awaitingReplyTo, text string) (m botmsg.MessageFromBot, err error) {
	var (
		tgChatID int64
		chatID   string
	)

	if chatID, err = whc.Input().BotChatID(); err != nil {
		return
	}

	if tgChatID, err = strconv.ParseInt(chatID, 10, 64); err != nil {
		return
	}
	// TODO: Does it changes locale from RU to EN?
	messageID := whc.Input().(telegram.WebhookCallbackQuery).GetMessage().MessageIntID()
	if m, err = whc.NewEditMessage(text, botmsg.FormatHTML); err != nil {
		return
	}
	m.EditMessageUID = telegram.NewChatMessageUID(tgChatID, int(messageID))
	if awaitingReplyTo != "" {
		if awaitingReplyTo == "/" {
			awaitingReplyTo = ""
		}
		whc.ChatData().SetAwaitingReplyTo(awaitingReplyTo)
	}
	return
}

const FeedbackTextCommandCode = "feedback_text"

var feedbackTextCommand = botsfw.Command{
	Code:       FeedbackTextCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		switch whc.Input().(type) {
		case botinput.TextMessage:
			mt := whc.Input().(botinput.TextMessage).Text()
			feedbackParam := whc.ChatData().GetWizardParam("feedback")

			var feedback models4debtus.Feedback
			ctx := whc.Context()
			if err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
				if feedbackParam == "" {
					feedback.FeedbackData = &models4debtus.FeedbackData{
						Rate:      "none",
						UserStrID: whc.AppUserID(),
						Text:      mt,
						CreatedOn: general4debtus.CreatedOn{
							CreatedOnPlatform: whc.BotPlatform().ID(),
							CreatedOnID:       whc.GetBotCode(),
						},
					}
				} else {
					feedback.ID = feedbackParam
					if feedback, err = getFeedbackByID(ctx, tx, feedbackParam); err != nil {
						return
					}
					feedback.Text = mt
				}
				if feedback, _, err = saveFeedback(ctx, tx, "", feedback.FeedbackData); err != nil {
					return
				}
				return nil
			}, dal.TxWithCrossGroup()); err != nil {
				return
			}
			m = whc.NewMessageByCode(trans.MESSAGE_TEXT_THANKS)
			m.Text += fmt.Sprintf(` Feedback #<a href="https://debtus.app/pwa/#/feedback/%s">%s</a>`, feedback.ID, feedback.ID)
			if err = SetMainMenuKeyboard(whc, &m); err != nil {
				return
			}
			if err2 := sendFeedbackToAdmins(ctx, "DebtusBotToken", feedback); err2 != nil {
				logus.Errorf(ctx, "failed to notify admins: %v", err)
			}
		default:
			m = whc.NewMessageByCode(trans.MESSAGE_TEXT_PLEASE_SEND_TEXT)
		}
		return
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return askToWriteFeedback(whc, "")
	},
}

func FeedbackLinks(t i18n.SingleLocaleTranslator, s string) (string, error) {
	ideaUrl, err := getUserReportUrl(t, "idea")
	if err != nil {
		return s, err
	}
	bugUrl, err := getUserReportUrl(t, "bug")
	if err != nil {
		return s, err
	}
	s = strings.Replace(s, "<a suggest-idea>", trans4debtus.Ahref(ideaUrl), 1)
	s = strings.Replace(s, "<a submit-bug>", trans4debtus.Ahref(bugUrl), 1)
	return s, nil
}
