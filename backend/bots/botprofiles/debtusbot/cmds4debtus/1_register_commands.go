package cmds4debtus

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_admin"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_retention"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_settings"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_transfer"
)

func RegisterCommands(router botswebhook.CommandsRegisterer) {
	cmds4anybot.RegisterInlineQueryHandlers("debtus_bot", inlineQueryHandler, chosenResultQueryHandler)
	dtb_admin.RegisterCommands(router)
	dtb_general.RegisterCommands(router)
	dtb_settings.RegisterCommands(router)
	dtb_transfer.RegisterCommands(router)
	router.RegisterCommands(
		newChatMembersCommand,
		dtb_retention.DeleteUserCommand,
		cmds4anybot.AddReferrerCommand,
		//OnboardingAskInviteChannelCommand, // We need it as otherwise do not handle replies.
		//SetPreferredLanguageCommand,
		//OnboardingAskInviteCodeCommand,
		//OnboardingCheckInviteCommand,
		//cmds4invites.inviteCommand,
		//dtb_fbm.FbmGetStartedCommand, // TODO: Move command to other package?
		//dtb_fbm.FbmMainMenuCommand,
		//dtb_fbm.FbmDebtsCommand,
		//dtb_fbm.FbmBillsCommand,
		//dtb_fbm.FbmSettingsCommand,
	)
}
