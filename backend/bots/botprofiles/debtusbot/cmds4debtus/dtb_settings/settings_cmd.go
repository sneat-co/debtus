package dtb_settings

import (
	"net/url"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
)

var settingsCommand = cmds4anybot.SettingsCommandTemplate

const debtusSettingsID = "debtus"

var settingsMainAction = cmds4anybot.SettingsMainAction

func init() {
	settingsCommand.Code = "debtus_settings"
	settingsCommand.Commands = []string{"/debtus_settings"}
	settingsCommand.Action = func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return settingsMainAction(whc, debtusSettingsID)
	}
	settingsCommand.CallbackAction = func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return settingsMainAction(whc, debtusSettingsID)
	}
}
