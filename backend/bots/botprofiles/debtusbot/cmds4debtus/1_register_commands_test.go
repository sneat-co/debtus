package cmds4debtus

import (
	"testing"

	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func TestRegisterCommands(t *testing.T) {
	router := botswebhook.NewWebhookRouter(nil)
	RegisterCommands(router)
}
