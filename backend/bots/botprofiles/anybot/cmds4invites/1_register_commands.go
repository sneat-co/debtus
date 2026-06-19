package cmds4invites

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
)

func RegisterCommands(botProfileID string, router botswebhook.CommandsRegisterer) {
	cmds4anybot.RegisterStartCommandHandlers(botProfileID, cmds4anybot.StartCommandHandler{
		Prefix:      startInviteCommandPrefix,
		StartAction: startInviteCommandAction,
	})
	router.RegisterCommands(

		// Callback & text commands
		inviteCommand,
		createMassInviteCommand,
		askInviteAddressCallbackCommand,
		chosenInlineResultCommand,

		// Inline query commands
		inviteContactToJoinInlineQueryCommand,
	)
}
