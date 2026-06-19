package dtb_general

import (
	"fmt"
	"strings"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/utm4bots"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/sneat-translations/trans"
)

func AdSlot(whc botsfw.WebhookContext, place string) string {
	utmParams := utm4bots.FillUtmParams(whc, utm.Params{Campaign: place})
	link := fmt.Sprintf(`href="https://debtus.app/%v/ads#%v"`, whc.Locale().SiteCode(), utmParams)
	return strings.Replace(whc.Translate(trans.MESSAGE_TEXT_YOUR_AD_COULD_BE_HERE), "href", link, 1)
}
