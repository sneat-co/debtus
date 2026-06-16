package splitusbot

import (
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/sneat-co/sneat-bots/pkg/bots/botinitparams"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/splitusbot/cmds4splitusbot"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/const4splitus"
)

var botProfile botsfw.BotProfile

func GetProfile(p botinitparams.BotProfileParams) botsfw.BotProfile {
	if botProfile == nil {
		router := botswebhook.NewWebhookRouter(p.ErrFooterText)
		cmds4splitusbot.RegisterCommand(router.RegisterCommands, true)
		botProfile = anybot.NewProfile(const4splitus.BotProfileID, router,
			botsfw.BotTranslations{
				Description:      "",
				ShortDescription: "",
			},
		)
	}
	return botProfile
}
