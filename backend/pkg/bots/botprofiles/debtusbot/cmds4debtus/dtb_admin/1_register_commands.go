package dtb_admin

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func RegisterCommands(router botswebhook.CommandsRegisterer) {
	router.RegisterCommands(
		adminCommand,
	)
}
