package dtb_transfer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/contactusmodels/briefs4contactus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp/person"
)

func TestSendReceiptCallbackData(t *testing.T) {
	got := SendReceiptCallbackData("t123", "user")
	want := "send_receipt?by=user&transfer=t123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSendReceiptUrl(t *testing.T) {
	got := SendReceiptUrl("t123", "user")
	want := "https://debtus.app/pwa/send-receipt?by=user&transfer=t123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsCurrencyIcon(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want bool
	}{
		{"cd-icon", emoji.CD_ICON, true},
		{"beer-icon", emoji.BEER_ICON, true},
		{"smoking-icon", emoji.SMOKING_ICON, true},
		{"not-an-icon", "USD", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCurrencyIcon(tc.in); got != tc.want {
				t.Errorf("IsCurrencyIcon(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestCleanPhoneNumber(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"spaces", "+1 234 567", "+1234567"},
		{"parens", "(123)456", "123456"},
		{"mixed", " (123) 456 789 ", "+123456789"[1:]},
		{"plain", "1234567890", "1234567890"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanPhoneNumber(tc.in); got != tc.want {
				t.Errorf("cleanPhoneNumber(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGetUrlForReceiptInTelegram(t *testing.T) {
	got := GetUrlForReceiptInTelegram("mybot", "r1", "en-US")
	if !strings.Contains(got, "receipt") {
		t.Errorf("expected url to contain 'receipt', got %q", got)
	}
	if !strings.Contains(got, "id=r1") {
		t.Errorf("expected url to contain 'id=r1', got %q", got)
	}
	if !strings.Contains(got, "lang=en-US") {
		t.Errorf("expected url to contain 'lang=en-US', got %q", got)
	}
}

func TestGetReturnDirectionFromDebtValue(t *testing.T) {
	for _, tc := range []struct {
		name    string
		amount  money.Amount
		want    models4debtus.TransferDirection
		wantErr bool
	}{
		{"negative", money.NewAmount("USD", -100), models4debtus.TransferDirectionUser2Counterparty, false},
		{"positive", money.NewAmount("USD", 100), models4debtus.TransferDirectionCounterparty2User, false},
		{"zero", money.NewAmount("USD", 0), models4debtus.TransferDirection(""), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getReturnDirectionFromDebtValue(tc.amount)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsTransferNotificationsBlockedForChannel(t *testing.T) {
	for _, tc := range []struct {
		name    string
		blocked []string
		channel string
		want    bool
	}{
		{"blocked", []string{"telegram", "email"}, "email", true},
		{"not-blocked", []string{"telegram"}, "email", false},
		{"empty", nil, "email", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cp := &models4debtus.DebtusSpaceContactDbo{NoTransferUpdatesBy: tc.blocked}
			if got := IsTransferNotificationsBlockedForChannel(cp, tc.channel); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShortDate(t *testing.T) {
	tm := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	enT := newTestSingleLocaleTranslator(i18n.LocaleEnUS)
	if got := shortDate(tm, enT); got != "15 Jan 2024" {
		t.Errorf("en shortDate = %q, want %q", got, "15 Jan 2024")
	}
	ruT := newTestSingleLocaleTranslator(i18n.LocaleRuRU)
	got := shortDate(tm, ruT)
	if !strings.Contains(got, "2024") || !strings.HasPrefix(got, "15") {
		t.Errorf("ru shortDate = %q, expected to start with 15 and contain 2024", got)
	}
}

func TestBalanceForCounterpartyWithHeader(t *testing.T) {
	translator := newTestSingleLocaleTranslator(i18n.LocaleEnUS)
	b := money.Balance{money.CurrencyUSD: 1000}
	got := BalanceForCounterpartyWithHeader("LINK", b, translator)
	if !strings.Contains(got, "LINK") {
		t.Errorf("expected output to contain counterparty link, got %q", got)
	}
	if !strings.Contains(got, translator.Translate(trans.MESSAGE_TEXT_BALANCE_HEADER)) {
		t.Errorf("expected output to contain header, got %q", got)
	}
}

func TestByContact_ZeroBalanceSkipped(t *testing.T) {
	const id = "c1"
	contacts := map[string]*briefs4contactus.ContactBrief{
		id: {Names: &person.NameFields{FirstName: "Zero"}},
	}
	debtusContacts := map[string]*models4debtus.DebtusContactBrief{
		id: {Status: "active", Balance: money.Balance{}},
	}
	got := enMock(t).ByContact(context.TODO(), enLinker, contacts, debtusContacts)
	if got != "" {
		t.Errorf("expected empty output for zero-balance contact, got %q", got)
	}
}

func TestByContact_BalanceWithInterestError(t *testing.T) {
	const id = "c1"
	contacts := map[string]*briefs4contactus.ContactBrief{
		id: {Names: &person.NameFields{FirstName: "Erroring"}},
	}
	debtusContacts := map[string]*models4debtus.DebtusContactBrief{
		id: {Status: "active", Balance: money.Balance{"USD": 1000}},
	}

	orig := balanceWithInterestFn
	balanceWithInterestFn = func(_ *models4debtus.DebtusContactBrief, _ context.Context, _ time.Time) (money.Balance, error) {
		return nil, errTestInterest
	}
	t.Cleanup(func() { balanceWithInterestFn = orig })

	got := enMock(t).ByContact(context.TODO(), enLinker, contacts, debtusContacts)
	if !strings.Contains(got, emoji.ERROR_ICON) {
		t.Errorf("expected error icon in output, got %q", got)
	}
	if !strings.Contains(got, "boom") {
		t.Errorf("expected error message in output, got %q", got)
	}
}

var errTestInterest = errTest("boom")

type errTest string

func (e errTest) Error() string { return string(e) }

func newTestSingleLocaleTranslator(locale i18n.Locale) i18n.SingleLocaleTranslator {
	translator := i18n.NewMapTranslator(context.TODO(), i18n.LocaleCodeEnUK, trans.TRANS)
	return i18n.NewSingleMapTranslator(locale, translator)
}
