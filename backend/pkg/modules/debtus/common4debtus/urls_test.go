package common4debtus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/strongo/i18n"
)

func stubIssueBotToken(t *testing.T, token string, err error) {
	t.Helper()
	backup := token4auth.IssueBotToken
	t.Cleanup(func() { token4auth.IssueBotToken = backup })
	token4auth.IssueBotToken = func(ctx context.Context, userID, createdOnPlatform, createdOnID string) (string, error) {
		return token, err
	}
}

func TestGetReceiptUrl(t *testing.T) {
	if got := GetReceiptUrl("r1", "debtus.app"); got != "https://debtus.app/receipt?id=r1" {
		t.Errorf("GetReceiptUrl() = %v", got)
	}
	for name, f := range map[string]func(){
		"empty_receipt_id": func() { GetReceiptUrl("", "debtus.app") },
		"empty_host":       func() { GetReceiptUrl("r1", "") },
		"host_not_domain":  func() { GetReceiptUrl("r1", "localhost") },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic")
				}
			}()
			f()
		})
	}
}

func TestGetWebsiteHost(t *testing.T) {
	for input, want := range map[string]string{
		"DevBot":     "dev.debtus.app",
		"some.local": "local.debtus.app",
		"DebtusBot":  "debtus.app",
	} {
		if got := GetWebsiteHost(input); got != want {
			t.Errorf("GetWebsiteHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGetPathAndQueryForInvite(t *testing.T) {
	got := GetPathAndQueryForInvite("code 1", utm.Params{Source: "src"})
	if !strings.HasPrefix(got, "ack?invite=code+1#") {
		t.Errorf("GetPathAndQueryForInvite() = %v", got)
	}
}

func TestGetBalanceAndHistoryUrlsForUser(t *testing.T) {
	stubIssueBotToken(t, "SECRET", nil)
	ctx := context.Background()

	balanceUrl := GetBalanceUrlForUser(ctx, 123, i18n.LocaleEnUS, "telegram", "DebtusBot")
	if !strings.Contains(balanceUrl, "/debts/") || !strings.Contains(balanceUrl, "secret=SECRET") {
		t.Errorf("GetBalanceUrlForUser() = %v", balanceUrl)
	}

	historyUrl := GetHistoryUrlForUser(ctx, 123, i18n.LocaleEnUS, "telegram", "DebtusBot")
	if !strings.Contains(historyUrl, "/history/") {
		t.Errorf("GetHistoryUrlForUser() = %v", historyUrl)
	}
}

func TestGetChooseCurrencyUrlForUser(t *testing.T) {
	stubIssueBotToken(t, "SECRET", nil)
	got := GetChooseCurrencyUrlForUser(context.Background(), "u1", i18n.LocaleEnUS, "telegram", "DebtusBot", "k=v")
	if !strings.Contains(got, "/choose-currency?") || !strings.Contains(got, "secret=SECRET") {
		t.Errorf("GetChooseCurrencyUrlForUser() = %v", got)
	}
}

func TestGetCounterpartyUrl_TokenError(t *testing.T) {
	stubIssueBotToken(t, "", errors.New("token error"))
	_, err := GetCounterpartyUrl(context.Background(), "c1", "u1", i18n.LocaleEnUS, utm.Params{Source: "DebtusBot", Medium: "bot", Campaign: "unit-test"})
	if err == nil {
		t.Error("expected error from token issuer")
	}
}

func TestGetTransferUrlForUser_TokenError(t *testing.T) {
	stubIssueBotToken(t, "", errors.New("token error"))
	got := GetTransferUrlForUser(context.Background(), "t1", "u1", i18n.LocaleEnUS, utm.Params{Source: "DebtusBot", Medium: "bot", Campaign: "unit-test"})
	if !strings.Contains(got, "secret=ERROR:token error") {
		t.Errorf("GetTransferUrlForUser() = %v", got)
	}
}
