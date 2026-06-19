package dtb_general

import (
	"fmt"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
)

const betaCommandCode = "beta"

var issueBotToken = token4auth.IssueBotToken

var betaCommand = botsfw.Command{
	Code:       betaCommandCode,
	Commands:   []string{"/beta"},
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		ctx := whc.Context()
		bot := whc.GetBotSettings()
		userID := whc.AppUserID()
		botPlatformID := whc.BotPlatform().ID()
		token, err := issueBotToken(ctx, userID, botPlatformID, bot.Code)
		if err != nil {
			return botmsg.MessageFromBot{}, err
		}
		host := common4debtus.GetWebsiteHost(bot.Code)
		betaUrl := fmt.Sprintf(
			"https://%s/app/#lang=%s&secret=%s",
			host, whc.Locale().SiteCode(), token,
		)
		return whc.NewMessage(betaUrl), nil
	},
}
