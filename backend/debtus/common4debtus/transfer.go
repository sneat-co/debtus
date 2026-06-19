package common4debtus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/strongo/i18n"

	"html/template"
)

func GetBalanceUrlForUser(ctx context.Context, userID int64, locale i18n.Locale, createdOnPlatform, createdOnID string) string {
	return getUrlForUser(ctx, userID, locale, "debts", createdOnPlatform, createdOnID)
}

func GetHistoryUrlForUser(ctx context.Context, userID int64, locale i18n.Locale, createdOnPlatform, createdOnID string) string {
	return getUrlForUser(ctx, userID, locale, "history", createdOnPlatform, createdOnID)
}

func getUrlForUser(ctx context.Context, userID int64, locale i18n.Locale, page, createdOnPlatform, createdOnID string) string {
	token, _ := token4auth.IssueBotToken(ctx, strconv.FormatInt(userID, 10), createdOnPlatform, createdOnID)
	host := GetWebsiteHost(createdOnID)
	url := fmt.Sprintf("https://%v/app/#", host)
	switch page {
	case "history":
		url += "/history/"
	case "debts":
		url += "/debts/"
	default:
		url += "page=" + page
	}
	return url + fmt.Sprintf("&lang=%v&secret=%v", locale.SiteCode(), token)
}

func GetTransferUrlForUser(ctx context.Context, transferID string, userID string, locale i18n.Locale, utmParams utm.Params) string {
	var buffer bytes.Buffer
	WriteTransferUrlForUser(ctx, &buffer, transferID, userID, locale, utmParams)
	return buffer.String()
}

func WriteTransferUrlForUser(ctx context.Context, writer io.Writer, transferID string, userID string, locale i18n.Locale, utmParams utm.Params) {
	host := GetWebsiteHost(utmParams.Source)
	_, _ = fmt.Fprintf(writer,
		"https://%v/transfer?id=%v&lang=%v",
		host, transferID, locale.SiteCode(),
	)
	if !utmParams.IsEmpty() {
		_, _ = fmt.Fprintf(writer, "&%v", utmParams.ShortString())
	}
	if userID != "" {
		token, err := token4auth.IssueBotToken(ctx, userID, utmParams.Medium, utmParams.Source)
		if err != nil {
			_, _ = fmt.Fprintf(writer, "&secret=ERROR:%v", err.Error())
		}
		_, _ = fmt.Fprintf(writer, "&secret=%v", token)
	}
}

func GetChooseCurrencyUrlForUser(ctx context.Context, userID string, locale i18n.Locale, createdOnPlatform, createdOnID, contextData string) string {
	token, _ := token4auth.IssueBotToken(ctx, userID, createdOnPlatform, createdOnID)
	host := GetWebsiteHost(createdOnID)
	return fmt.Sprintf(
		"https://%v/app/#/choose-currency?lang=%v&%v&secret=%v",
		host, locale.SiteCode(), contextData, token,
	)
}

func GetWebsiteHost(createdOnID string) string {
	createdOnID = strings.ToLower(createdOnID)
	if strings.Contains(createdOnID, "dev") {
		return "dev.debtus.app" // TODO: confirm dev hosting exists; was debtusbot-dev1.appspot.com (GAE, dead)
	} else if strings.Contains(createdOnID, ".local") {
		return "local.debtus.app"
	} else {
		return "debtus.app"
	}
}

func GetPathAndQueryForInvite(inviteCode string, utmParams utm.Params) string {
	return fmt.Sprintf("ack?invite=%v#%v", template.URLQueryEscaper(inviteCode), utmParams)
}
