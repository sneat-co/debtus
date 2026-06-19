package splitusbot

import (
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/splitusbot/cmds4splitusbot"
	"github.com/sneat-co/debtus/backend/splitus/const4splitus"
	"github.com/sneat-co/sneat-bots/pkg/bots/botinitparams"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
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
