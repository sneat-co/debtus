package dtb_inline

import (
	"context"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// fakeInlineQuery implements botinput.InlineQuery AND botinput.InputMessage
// so whc.Input().(botinput.InlineQuery) type assertion succeeds.
type fakeInlineQuery struct {
	id string
}

func (f fakeInlineQuery) GetID() any               { return f.id }
func (f fakeInlineQuery) GetInlineQueryID() string { return f.id }
func (f fakeInlineQuery) GetFrom() botinput.Sender { return nil }
func (f fakeInlineQuery) GetQuery() string         { return "" }
func (f fakeInlineQuery) GetOffset() string        { return "" }

// botinput.InputMessage methods
func (f fakeInlineQuery) GetSender() botinput.User         { return nil }
func (f fakeInlineQuery) GetRecipient() botinput.Recipient { return nil }
func (f fakeInlineQuery) GetTime() time.Time               { return time.Time{} }
func (f fakeInlineQuery) InputType() botinput.Type         { return botinput.TypeInlineQuery }
func (f fakeInlineQuery) MessageIntID() int                { return 0 }
func (f fakeInlineQuery) MessageStringID() string          { return "" }
func (f fakeInlineQuery) BotChatID() (string, error)       { return "", nil }
func (f fakeInlineQuery) Chat() botinput.Chat              { return nil }
func (f fakeInlineQuery) LogRequest()                      {}

var _ botinput.InlineQuery = fakeInlineQuery{}

func setupWhcForInline(t *testing.T) *mock_botsfw.MockWebhookContext {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeInlineQuery{id: "qid1"}).AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	return whc
}

func TestInlineNewRecord(t *testing.T) {
	tests := []struct {
		name    string
		matches []string // [full, amount, currency]
		wantErr bool
	}{
		{
			name:    "USD default (empty currency)",
			matches: []string{"100", "100", ""},
		},
		{
			name:    "RUB by symbol",
			matches: []string{"50 ₽", "50", "₽"},
		},
		{
			name:    "RUB by word rub",
			matches: []string{"50 rub", "50", "rub"},
		},
		{
			name:    "RUB by word ruble",
			matches: []string{"50 ruble", "50", "ruble"},
		},
		{
			name:    "RUB by word rubles",
			matches: []string{"50 rubles", "50", "rubles"},
		},
		{
			name:    "RUB by word rubley",
			matches: []string{"50 rubley", "50", "rubley"},
		},
		{
			name:    "RUB by cyrillic р",
			matches: []string{"50 р", "50", "р"},
		},
		{
			name:    "RUB by cyrillic руб",
			matches: []string{"50 руб", "50", "руб"},
		},
		{
			name:    "RUB by cyrillic рубля",
			matches: []string{"50 рубля", "50", "рубля"},
		},
		{
			name:    "RUB by cyrillic рублей",
			matches: []string{"50 рублей", "50", "рублей"},
		},
		{
			name:    "EUR by word eur",
			matches: []string{"30 eur", "30", "eur"},
		},
		{
			name:    "EUR by word euro",
			matches: []string{"30 euro", "30", "euro"},
		},
		{
			name:    "EUR by symbol €",
			matches: []string{"30 €", "30", "€"},
		},
		{
			name:    "UAH by cyrillic гривна",
			matches: []string{"100 гривна", "100", "гривна"},
		},
		{
			name:    "UAH by cyrillic гривен",
			matches: []string{"100 гривен", "100", "гривен"},
		},
		{
			name:    "UAH by cyrillic г",
			matches: []string{"100 г", "100", "г"},
		},
		{
			name:    "UAH by symbol ₴",
			matches: []string{"100 ₴", "100", "₴"},
		},
		{
			name:    "KZT by cyrillic тенге",
			matches: []string{"100 тенге", "100", "тенге"},
		},
		{
			name:    "KZT by cyrillic теңге",
			matches: []string{"100 теңге", "100", "теңге"},
		},
		{
			name:    "KZT by cyrillic т",
			matches: []string{"100 т", "100", "т"},
		},
		{
			name:    "KZT by symbol ₸",
			matches: []string{"100 ₸", "100", "₸"},
		},
		{
			name:    "custom currency code",
			matches: []string{"200 GBP", "200", "GBP"},
		},
		{
			name:    "long currency code truncated",
			matches: []string{"200 ABCDEFGHIJKLMNOPQRSTU", "200", "ABCDEFGHIJKLMNOPQRSTU"},
		},
		{
			name:    "invalid amount",
			matches: []string{"abc xyz", "abc", "xyz"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			whc := setupWhcForInline(t)
			_, err := InlineNewRecord(whc, tc.matches)
			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
