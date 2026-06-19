package cmds4splitusbot

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
)

const LeaveGroupCommandCode = "leave_group"

var leaveGroupCommand = bothelper.NewSpaceCallbackCommand(LeaveGroupCommandCode,
	func(whc botsfw.WebhookContext, _ *url.URL, space dbo4spaceus.SpaceEntry) (m botmsg.MessageFromBot, err error) {
		return
	},
)
