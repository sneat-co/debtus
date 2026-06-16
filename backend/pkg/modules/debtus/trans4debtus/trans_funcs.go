package trans4debtus

import (
	"fmt"
	"strings"

	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func AskToTranslate(t i18n.SingleLocaleTranslator) string {
	return strings.Replace(t.Translate(trans.MESSAGE_TEXT_ASK_TO_TRANSLATE),
		"<a>",
		`<a href="https://goo.gl/tZsqW1">`, // https://github.com/senat-co/debtstracker-translations
		1)
}

func YouCanHelp(t i18n.SingleLocaleTranslator, s, botCode string) string {
	s = t.Translate(s)
	s = strings.Replace(s, "<a storebot>", Ahref(StorebotUrl(botCode)), 1)
	s = strings.Replace(s, "<a share-vk>", Ahref(ShareToVkUrl()), 1)
	s = strings.Replace(s, "<a share-fb>", Ahref(ShareToFacebookUrl()), 1)
	s = strings.Replace(s, "<a share-twitter>", Ahref(ShareToTwitter()), 1)
	return s
}

func Ahref(url string) string {
	return fmt.Sprintf(`<a href="%v">`, url)
}

func StorebotUrl(botID string) string {
	return "https://t.me/storebot?start=" + botID
}

func ShareToFacebookUrl() string {
	return "https://goo.gl/WyrRLg" // "https://www.facebook.com/sharer/sharer.php?u=https%3A//debtus.app/"
}

func ShareToVkUrl() string {
	return "https://goo.gl/lcnPJ3" // "https://vk.com/share.php?url=https%3A//debtus.app/&title=Отличный%20Telegram%20бот%20для%20учёта%20долгов%20-%20https%3A//t.me/DebtusBot"
}

func ShareToTwitter() string {
	return "https://goo.gl/Xbv004" // "https://twitter.com/home?status=The%20%40DebtusApp%20is%20awesome.%20Check%20their%20%23Telegram%20bot%20https%3A//t.me/DebtusBot"
}
