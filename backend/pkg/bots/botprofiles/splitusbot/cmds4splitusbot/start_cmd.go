package cmds4splitusbot

import (
	"strings"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

func StartInGroupAction(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	// TODO: The old implementation (commented below) added the sender to the
	// group and asked for the group's primary currency. It relied on the
	// obsolete facade4debtus.Group facade and needs to be redesigned on top of
	// current space models. For now we just greet the group.
	logus.Debugf(whc.Context(), "splitus.StartInGroupAction()")
	m.Text = "\n\n" + whc.Translate(trans.SPLITUS_TEXT_HI_IN_GROUP)
	m.Format = botmsg.FormatHTML
	return
	//ctx := whc.Context()
	//logus.Debugf(ctx, "splitus.StartInGroupAction()")
	//var group models.Group
	//if group, err = shared_space.GetGroup(whc, nil); err != nil {
	//	return
	//}
	////var appUserData botsfwmodels.AppUserData
	////if appUserData, err = whc.AppUserData(); err != nil {
	////	return
	////}
	//
	////appUser := appUserData.(*models.DebutsAppUserDataOBSOLETE)
	//
	//var botUser record.DataWithID[string, botsfwmodels.BotUserData]
	//
	//if botUser, err = whc.BotUser(); err != nil && !dal.IsNotFound(err) {
	//	return
	//}
	//
	//if group, _, err = facade4debtus.Group.AddUsersToTheGroupAndOutstandingBills(c, group.ContactID, []facade4debtus.NewUser{
	//	{
	//		//UserTitle:        appUserData.FullName(),
	//		BotUserData: botUser.Data,
	//		ChatMember:  whc.Input().GetSender(),
	//	},
	//}); err != nil {
	//	err = fmt.Errorf("%w: failed to add user to the group", err)
	//	return
	//}
	//m.Text = 	//	"\n\n" + whc.Translate(trans.SPLITUS_TEXT_HI_IN_GROUP) +
	//	"\n\n<b>" + whc.Translate(trans.MESSAGE_TEXT_ASK_PRIMARY_CURRENCY_FOR_GROUP) + "</b>"
	//
	//m.Format = botmsg.FormatHTML
	//m.Keyboard = currenciesInlineKeyboard(
	//	GroupSettingsSetCurrencyCommandCode+"?start=y&group="+group.ContactID,
	//	[]tgbotapi.InlineKeyboardButton{
	//		{
	//			Text: whc.Translate(trans.BT_OTHER_CURRENCY),
	//			URL:  bothelper.StartTelegramBotUrl(whc.GetBotCode(), GroupSettingsChooseCurrencyCommandCode, "group"+group.ContactID),
	//		},
	//	},
	//)
	//return
}

func StartInBotAction(whc botsfw.WebhookContext, startParams []string) (m botmsg.MessageFromBot, err error) {
	logus.Debugf(whc.Context(), "splitus.StartInBotAction() => startParams: %v", startParams)
	if len(startParams) > 0 {
		switch {
		case strings.HasPrefix(startParams[0], "bill-"):
			return startBillAction(whc, startParams[0])
		case startParams[0] == SettleGroupAskForCounterpartyCommandCode:
			return settleGroupStartAction(whc, startParams[1:])
		}
	}
	err = cmds4anybot.ErrUnknownStartParam
	return
}
