package cmds4splitusbot

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/debtussender"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const groupsCommandCode = "groups"

var groupsCommand = botsfw.Command{
	Code:       groupsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   trans.Commands(trans.COMMAND_TEXT_GROUPS, emoji.MAN_AND_WOMAN, "/"+groupsCommandCode),
	Icon:       emoji.MAN_AND_WOMAN,
	Title:      trans.COMMAND_TEXT_GROUPS,
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return groupsAction(whc, false, 0)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		query := callbackUrl.Query()
		isRefresh := query.Get("do") == "refresh"
		if m, err = groupsAction(whc, isRefresh || query.Get("edit") == "1", 0); err != nil {
			return
		}
		if isRefresh {
			ctx := whc.Context()
			logus.Debugf(ctx, "do == 'refresh'")
			if m, err = debtussender.SendRefreshOrNothingChanged(whc, m); err != nil {
				return
			}
		}
		return
	},
}

func groupsAction(whc botsfw.WebhookContext, isEdit bool, groupsMessageID int) (m botmsg.MessageFromBot, err error) {
	// TODO: The old implementation (commented below) relied on the obsolete
	// DebutsAppUserDataOBSOLETE.ActiveGroups() user model and needs to be
	// redesigned on top of current space/contactus models.
	_, _ = isEdit, groupsMessageID
	var isInGroup bool
	if isInGroup, err = whc.IsInGroup(); err != nil {
		return
	} else if isInGroup {
		m.Text = "This command supported just in private chat with @" + whc.GetBotCode()
		return
	}
	m.Text = "Sorry, the groups list is not available yet."
	return
	//if whc.IsInGroup() {
	//	m.Text = "This command supported just in private chat with @" + whc.GetBotCode()
	//	return
	//}
	//ctx := whc.Context()
	//logus.Debugf(ctx, "groupsAction(isEdit=%v, groupsMessageID=%d)", isEdit, groupsMessageID)
	//buf := new(bytes.Buffer)
	//
	//fmt.Fprintf(buf, "<b>%v</b>\n\n", whc.Translate(trans.MESSAGE_TEXT_YOUR_BILL_SPLITTING_GROUPS))
	//
	//var appUserData botsfwmodels.AppUserData
	//if appUserData, err = whc.AppUserData(); err != nil {
	//	return
	//}
	//appUserEntity := appUserData.(*models.DebutsAppUserDataOBSOLETE)
	//
	//groups := appUserEntity.ActiveGroups()
	//
	//{ // Filter groups known to bot or not linked to bot
	//	botCode := whc.GetBotCode()
	//	var j = 0
	//	for _, g := range groups {
	//		knownGroup := false
	//		if len(g.TgBots) == 0 {
	//			knownGroup = true
	//		} else {
	//			for _, tgBot := range g.TgBots {
	//				if tgBot == botCode {
	//					knownGroup = true
	//					break
	//				}
	//			}
	//		}
	//		if knownGroup {
	//			groups[j] = g
	//			j += 1
	//		}
	//	}
	//	groups = groups[:j]
	//}
	//
	//if len(groups) == 0 {
	//	buf.WriteString(whc.Translate(trans.MESSAGE_TEXT_NO_GROUPS))
	//} else {
	//	for i, group := range groups {
	//		fmt.Fprintf(buf, "  %d. %v\n", i+1, group.UserTitle)
	//	}
	//
	//	fmt.Fprint(buf, "\n", whc.Translate(trans.MESSAGE_TEXT_USE_ARROWS_TO_SELECT_GROUP))
	//}
	//
	//m.Text = buf.CallbackData()
	//
	//tgKeyboard := tgbotapi.NewInlineKeyboardMarkup(
	//	[]tgbotapi.InlineKeyboardButton{},
	//)
	//if len(groups) > 0 {
	//	tgKeyboard.InlineKeyboard = append(tgKeyboard.InlineKeyboard, groupsNavButtons(whc, groups, ""))
	//}
	//
	//if groupsMessageID == 0 {
	//	if isEdit {
	//		groupsMessageID = whc.Input().(telegram.WebhookCallbackQuery).TgUpdate().CallbackQuery.Message.MessageID
	//	}
	//} else {
	//	m.EditMessageUID = telegram.ChatMessageUID{MessageID: groupsMessageID}
	//}
	//
	//tgKeyboard.InlineKeyboard = append(tgKeyboard.InlineKeyboard,
	//	[]tgbotapi.InlineKeyboardButton{
	//		shared_space.NewGroupTelegramInlineButton(whc, groupsMessageID),
	//	},
	//	[]tgbotapi.InlineKeyboardButton{
	//		{
	//			Text:         whc.Translate(trans.COMMAND_TEXT_REFRESH),
	//			CallbackData: groupsCommandCode + "?do=refresh",
	//		},
	//	},
	//)
	//
	//m.Keyboard = tgKeyboard
	//m.IsEdit = isEdit
	//m.Format = botmsg.FormatHTML
	//if !isEdit {
	//	var msg botsfw.OnMessageSentResponse
	//	if msg, err = whc.Responder().SendMessage(c, m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
	//		return
	//	}
	//	return groupsAction(whc, true, msg.TelegramMessage.(tgbotapi.Message).MessageID)
	//}
	//return
}
