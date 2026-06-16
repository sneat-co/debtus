package dtb_general

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func RegisterCommands(router botswebhook.CommandsRegisterer) {
	router.RegisterCommands(
		debtsCommand,
		MainMenuCommand,
		pleaseWaitCommand,
		feedbackCommand,
		feedbackTextCommand,
		canYouRateCommand,
		debtusHomeCommand,
		deleteAllCommand,
		betaCommand,
		login2WebCommand,
		clearCommand,
		adsCommand,
		debtusContactsCommand,
		debtorsCommand,
		creditorsCommand,
	)
}
