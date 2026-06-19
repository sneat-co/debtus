package cmds4splitusbot

import (
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-translations/trans"
)

func RegisterCommand(registerCommands func(commands ...botsfw.Command), isStandaloneBot bool) {
	if isStandaloneBot {
		registerCommands(menuCommand)
	}
	RegisterSharedCommands(registerCommands)
	cmds4anybot.RegisterCommonCommands(registerCommands, cmds4anybot.BotParams{
		StartInGroupAction: StartInGroupAction,
		StartInBotAction:   StartInBotAction,
		HelpCommandAction: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
			m.Text = "HelpCommandAction is not implemented yet"
			return
		},
		//GetGroupBillCardInlineKeyboard:   getGroupBillCardInlineKeyboard,
		//GetPrivateBillCardInlineKeyboard: getPrivateBillCardInlineKeyboard,
		//DelayUpdateBillCardOnUserJoin:    delayUpdateBillCardOnUserJoin,
		//OnAfterBillCurrencySelected:      getWhoPaidInlineKeyboard,
		//ShowGroupMembers:                 showGroupMembers,
		GetWelcomeMessageText: func(whc botsfw.WebhookContext) (text string, err error) {
			var user dbo4userus.UserEntry
			if user, err = cmds4anybot.GetUser(whc); err != nil {
				return
			}
			text = whc.Translate(
				trans.MESSAGE_TEXT_HI_USERNAME, user.Data.Names.FirstName) + " " + whc.Translate(trans.SPLITUS_TEXT_HI) +
				"\n\n" + whc.Translate(trans.SPLITUS_TEXT_ABOUT_ME_AND_CO) +
				"\n\n" + whc.Translate(trans.SPLITUS_TG_COMMANDS)

			return
		},
		SetMainMenu: func(whc botsfw.WebhookContext, messageText string, showHint bool) (m botmsg.MessageFromBot, err error) {
			SetMainMenu(whc, &m)
			return
		},
	})
}

func RegisterSharedCommands(registerCommands func(commands ...botsfw.Command)) {
	registerCommands(
		EditedBillCardHookCommand,
		billsCommand,
		settingsCommand,
		settleBillsCommand,
		outstandingBalanceCommand,
		joinBillCommand,
		closeBillCommand,
		editBillCommand,
		newBillCommand,
		groupBalanceCommand,
		billSharesCommand,
		billSplitModesListCommand,
		finalizeBillCommand,
		deleteBillCommand,
		restoreBillCommand,
		billChangeSplitModeCommand,
		changeBillPayerCommand,
		spaceSplitCommand,
		joinSpaceCommand,
		setBillCurrencyCommand,
		groupCommand,
		leaveGroupCommand,
		billCardCommand,
		billMembersCommand,
		inviteToBillCommand,
		setBillDueDateCommand,
		changeBillTotalCommand,
		addBillComment,
		groupMembersCommand,
		groupSettingsSetCurrencyCommand(),
		groupsCommand,
		groupSettingsChooseCurrencyCommand,
		settleGroupAskForCounterpartyCommand,
		settleGroupCounterpartyChosenCommand,
		settleGroupCounterpartyConfirmedCommand,
		chosenInlineResultCommand,
		newChatMembersCommand,
		//updateBillTgChatCardCommand,
	)
}
