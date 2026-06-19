package dtb_common

import (
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
)

func GetUser(whc botsfw.WebhookContext) (user dbo4userus.UserEntry, err error) {
	return cmds4anybot.GetUser(whc)
}
