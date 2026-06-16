package cmds4splitusbot

import (
	"testing"

	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func TestRegisterSharedCommands(t *testing.T) {
	router := botswebhook.NewWebhookRouter(nil)
	RegisterSharedCommands(router.RegisterCommands)
}
