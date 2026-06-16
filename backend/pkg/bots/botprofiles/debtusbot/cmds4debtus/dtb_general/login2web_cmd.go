package dtb_general

import (
	"fmt"
	"strings"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	bots "github.com/sneat-co/debtus/backend/pkg/bots/botscompat"
	"github.com/sneat-co/sneat-translations/trans"
)

var login2WebCommand = botsfw.Command{
	Code:       "login2web",
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{"/login"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		mt := whc.Translate(trans.MESSAGE_TEXT_LOGIN_TO_WEB_APP)
		linker := bots.NewLinkerFromWhc(whc)
		mt = strings.Replace(mt, "<a>", fmt.Sprintf(`<a href="%v">`, linker.ToMainScreen()), 1)
		m = whc.NewMessage(mt)
		m.Format = botmsg.FormatHTML
		m.DisableWebPagePreview = true
		return
	},
}
