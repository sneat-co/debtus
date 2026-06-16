package cmds4debtus

import (
	"context"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/sneat-co/sneat-bots/pkg/bots/botinitparams"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/debtusbotconst"
)

var botProfile botsfw.BotProfile

func GetProfile(p botinitparams.BotProfileParams) botsfw.BotProfile {
	if botProfile == nil {
		botProfile = CreateDebtusBotProfile(p.ErrFooterText)
	}
	return botProfile
}

func CreateDebtusBotProfile(errFooterText func(ctx context.Context, botContext botswebhook.ErrorFooterArgs) string) botsfw.BotProfile {
	router := botswebhook.NewWebhookRouter(errFooterText)

	cmds4anybot.RegisterCommonCommands(router.RegisterCommands, botParams)
	RegisterCommands(router)
	return anybot.NewProfile(debtusbotconst.ProfileID, router,
		botsfw.BotTranslations{
			Description:      "",
			ShortDescription: "",
		},
	)
}
