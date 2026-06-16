package dtb_settings

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func RegisterCommands(router botswebhook.CommandsRegisterer) {
	router.RegisterCommands(
		settingsCommand, // duplicate
		loginPinCommand,
		fixBalanceCommand,
		//OnboardingTellAboutInviteCodeCommand, // We need it as otherwise do not handle replies. Consider incorporate to StartCommand?
		AskCurrencySettingsCommand,
	)
}
