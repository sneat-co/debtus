package cmds4debtus

import (
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_settings"
)

var botParams = cmds4anybot.BotParams{
	AppURL: "https://debtus.app",
	StartInGroupAction: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		m.Text = "StartInGroupAction is not implemented yet"
		return
	},
	//GetGroupBillCardInlineKeyboard:   getGroupBillCardInlineKeyboard,
	//GetPrivateBillCardInlineKeyboard: getPrivateBillCardInlineKeyboard,
	//DelayUpdateBillCardOnUserJoin:    delayUpdateBillCardOnUserJoin,
	//OnAfterBillCurrencySelected:      getWhoPaidInlineKeyboard,
	//ShowGroupMembers:                 showGroupMembers,
	HelpCommandAction: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return dtb_general.HelpCommandAction(whc, true)
	},
	//InGroupWelcomeMessage: func(whc botsfw.WebhookContext, group models.Group) (m botmsg.MessageFromBot, err error) {
	//	m, err = shared_all.GroupSettingsAction(whc, group, false)
	//	if err != nil {
	//		return
	//	}
	//	if _, err = whc.Responder().SendMessage(whc.Context(), m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
	//		return
	//	}
	//
	//	return whc.NewEditMessage(whc.Translate(trans.MESSAGE_TEXT_HI)+
	//		"\n\n"+ whc.Translate(trans.SPLITUS_TEXT_HI_IN_GROUP)+
	//		"\n\n"+ whc.Translate(trans.SPLITUS_TEXT_ABOUT_ME_AND_CO),
	//		sneatbots.MessageFormatHTML)
	//},
	GetWelcomeMessageText: func(whc botsfw.WebhookContext) (text string, err error) {
		text = "Hi there"
		return
	},
	//
	//
	//
	StartInBotAction: dtb_settings.StartInBotAction,
	SetMainMenu: func(whc botsfw.WebhookContext, messageText string, showHint bool) (m botmsg.MessageFromBot, err error) {
		err = dtb_general.SetMainMenuKeyboard(whc, &m)
		return
	},
}
