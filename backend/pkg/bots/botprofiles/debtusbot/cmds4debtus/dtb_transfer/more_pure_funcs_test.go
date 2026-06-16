package dtb_transfer

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/decimal"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

func TestGetInterestData(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"valid", "simple/10/30/0/0", false},
		{"unknown-formula", "bogus/10/30/0/0", true},
		{"bad-period", "simple/10/notanint/0/0", true},
		{"bad-percent", "simple/notnum/30/0/0", true},
		{"bad-minimum", "simple/10/30/notnum/0", true},
		{"bad-grace", "simple/10/30/0/notnum", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getInterestData(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTransferWizard_CounterpartyID(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		query string
		want  string
	}{
		{"counterparty-key", WizardParamCounterparty + "=cp1", "cp1"},
		{"contact-key-fallback", WizardParamContact + "=ct2", "ct2"},
		{"empty", "other=x", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatalf("parse query: %v", err)
			}
			w := TransferWizard{params: params}
			if got := w.CounterpartyID(ctx); got != tc.want {
				t.Errorf("CounterpartyID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSendReceiptByTelegramButton(t *testing.T) {
	tr := newTestSingleLocaleTranslator(i18n.LocaleEnUS)
	btn := sendReceiptByTelegramButton("t42", tr)
	if btn.SwitchInlineQuery == nil || !strings.Contains(*btn.SwitchInlineQuery, "receipt?id=t42") {
		t.Errorf("unexpected switch inline query: %+v", btn.SwitchInlineQuery)
	}
}

func TestDebtAmountButtonText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...interface{}) string {
		return "%v %v"
	}).AnyTimes()

	cp := models4debtus.NewDebtusSpaceContactEntry("space1", "c1", &models4debtus.DebtusSpaceContactDbo{})

	for _, tc := range []struct {
		name  string
		value decimal.Decimal64p2
	}{
		{"positive", decimal.NewDecimal64p2(100, 0)},
		{"negative", decimal.NewDecimal64p2(-100, 0)},
		{"zero", decimal.NewDecimal64p2(0, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := _debtAmountButtonText(whc, money.CurrencyUSD, tc.value, cp)
			if got == "" {
				t.Errorf("expected non-empty text")
			}
		})
	}
}

func TestGetReturnWizardParams(t *testing.T) {
	for _, tc := range []struct {
		name           string
		awaiting       string
		wantSpaceID    string
		wantCpID       string
		wantTransferID string
		wantErr        bool
	}{
		{
			name:           "well-formed",
			awaiting:       "cmd?" + WizardParamSpace + "=sp1&" + WizardParamCounterparty + "=cp1&" + WizardParamTransfer + "=tr1",
			wantSpaceID:    "sp1",
			wantCpID:       "cp1",
			wantTransferID: "tr1",
		},
		{
			name:     "malformed-query",
			awaiting: "cmd?%zz",
			wantErr:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
			chatData.EXPECT().GetAwaitingReplyTo().Return(tc.awaiting).AnyTimes()

			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().ChatData().Return(chatData).AnyTimes()

			spaceID, cpID, transferID, err := getReturnWizardParams(whc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(spaceID) != tc.wantSpaceID {
				t.Errorf("spaceID = %q, want %q", spaceID, tc.wantSpaceID)
			}
			if cpID != tc.wantCpID {
				t.Errorf("counterpartyID = %q, want %q", cpID, tc.wantCpID)
			}
			if transferID != tc.wantTransferID {
				t.Errorf("transferID = %q, want %q", transferID, tc.wantTransferID)
			}
		})
	}
}

type fakeAppUserWithCurrencies struct {
	currencies []money.CurrencyCode
}

func (f fakeAppUserWithCurrencies) GetLastCurrencies() []money.CurrencyCode { return f.currencies }

func (f fakeAppUserWithCurrencies) BotsFwAdapter() botsfwmodels.AppUserAdapter { return nil }

func TestAskTransferCurrencyButtons(t *testing.T) {
	for _, tc := range []struct {
		name       string
		currencies []money.CurrencyCode
	}{
		{"no-last-currencies", nil},
		{"with-last-currencies", []money.CurrencyCode{money.CurrencyUSD, money.CurrencyEUR}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().AppUserData().Return(fakeAppUserWithCurrencies{currencies: tc.currencies}, nil).AnyTimes()
			whc.EXPECT().Translate(trans.COMMAND_TEXT_CANCEL).Return("Cancel").AnyTimes()

			rows := AskTransferCurrencyButtons(whc)
			if len(rows) == 0 {
				t.Fatalf("expected non-empty rows")
			}
			last := rows[len(rows)-1]
			if len(last) != 1 || last[0] != "Cancel" {
				t.Errorf("expected last row to be the cancel button, got %v", last)
			}
		})
	}
}
