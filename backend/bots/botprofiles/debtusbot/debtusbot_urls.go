package debtusbot

import (
	"fmt"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
)

func GetNewDebtPageUrl(whc botsfw.WebhookContext, direction models4debtus.TransferDirection, utmCampaign string) string {
	botID := whc.GetBotCode()
	botPlatform := whc.BotPlatform().ID()
	ctx := whc.Context()
	appUserID := whc.AppUserID()
	botIssuer := token4auth.GetBotIssuer(botPlatform, botID)
	token, _ := token4auth.IssueAuthToken(ctx, appUserID, botIssuer)
	host := common4debtus.GetWebsiteHost(botID)
	// utmParams := NewUtmParams(whc, utmCampaign)
	return fmt.Sprintf(
		"https://%v/open/new-debt?d=%v&lang=%v&secret=%v",
		host, direction, whc.Locale().SiteCode(), token, // utmParams, - commented out as: Viber response.Status=3: keyboard is not valid. is too long (length: 274, maximum allowed: 250)]
	)
}
