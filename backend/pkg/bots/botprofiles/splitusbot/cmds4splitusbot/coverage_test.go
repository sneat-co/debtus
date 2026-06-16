package cmds4splitusbot

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse url %q: %v", s, err)
	}
	return u
}

// reentrantDB wraps a dalgo DB so nested RunReadwriteTransaction calls reuse
// the outer transaction instead of deadlocking on the memory adapter mutex.
// Tests are single-goroutine, so a plain depth counter is enough.
type reentrantDB struct {
	dal.DB
	depth int
	tx    dal.ReadwriteTransaction
}

func (db *reentrantDB) RunReadwriteTransaction(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
	if db.depth > 0 {
		return f(ctx, db.tx)
	}
	db.depth++
	defer func() { db.depth--; db.tx = nil }()
	return db.DB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		db.tx = tx
		return f(ctx, tx)
	}, opts...)
}

// withMemDB overrides facade.GetSneatDB with an in-memory dalgo DB and restores it after the test.
func withMemDB(t *testing.T) dal.DB {
	t.Helper()
	memDB := &reentrantDB{DB: dalgo2memory.NewDB()}
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return memDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = orig })
	return memDB
}

// putBill stores a bill so facade4splitus.GetBillByID can read it back.
func putBill(t *testing.T, ctx context.Context, db dal.DB, billID string, data *models4splitus.BillDbo) {
	t.Helper()
	key := dal.NewKeyWithID(models4splitus.BillKind, billID)
	rec := dal.NewRecordWithData(key, data)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, rec)
	}); err != nil {
		t.Fatalf("failed to put bill: %v", err)
	}
}

func TestGetBillMembersCallbackData(t *testing.T) {
	got := GetBillMembersCallbackData("bill123")
	if !strings.Contains(got, "bill123") {
		t.Errorf("expected bill ID in callback data, got %q", got)
	}
	if !strings.Contains(got, billMembersCommandCode) {
		t.Errorf("expected command code in callback data, got %q", got)
	}
}

func TestGetBill(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	t.Run("missing_bill_param", func(t *testing.T) {
		u := mustParseURL(t, "http://x/cb")
		_, err := getBill(ctx, nil, u)
		if err == nil {
			t.Fatal("expected error for missing bill param")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		u := mustParseURL(t, "http://x/cb?bill=nope")
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, err := getBill(ctx, tx, u)
			return err
		})
		if err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("success", func(t *testing.T) {
		putBill(t, ctx, db, "b1", &models4splitus.BillDbo{})
		u := mustParseURL(t, "http://x/cb?bill=b1")
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := getBill(ctx, tx, u)
			if err != nil {
				return err
			}
			if bill.ID != "b1" {
				t.Errorf("expected bill id b1, got %q", bill.ID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestBillCallbackAction(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putBill(t, ctx, db, "bc1", &models4splitus.BillDbo{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("not_in_group_success", func(t *testing.T) {
		whc := mock_botsfw.NewMockWebhookContext(ctrl)
		whc.EXPECT().Context().Return(ctx).AnyTimes()
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()

		action := billCallbackAction(func(whc botsfw.WebhookContext, tx dal.ReadwriteTransaction, cbURL *url.URL, bill models4splitus.BillEntry) (botmsg.MessageFromBot, error) {
			return botmsg.MessageFromBot{}, nil
		})
		_, err := action(whc, mustParseURL(t, "http://x/cb?bill=bc1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("getbill_error", func(t *testing.T) {
		whc := mock_botsfw.NewMockWebhookContext(ctrl)
		whc.EXPECT().Context().Return(ctx).AnyTimes()
		action := billCallbackAction(func(whc botsfw.WebhookContext, tx dal.ReadwriteTransaction, cbURL *url.URL, bill models4splitus.BillEntry) (botmsg.MessageFromBot, error) {
			return botmsg.MessageFromBot{}, nil
		})
		_, err := action(whc, mustParseURL(t, "http://x/cb"))
		if err == nil {
			t.Fatal("expected error for missing bill param")
		}
	})
}

func TestSettleGroupActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	cases := []struct {
		name string
		call func() error
	}{
		{"start", func() error {
			_, err := settleGroupStartAction(whc, []string{"space=space1"})
			return err
		}},
		{"ask_counterparty", func() error {
			_, err := settleGroupAskForCounterpartyAction(whc, dbo4spaceus.SpaceEntry{})
			return err
		}},
		{"counterparty_chosen", func() error {
			_, err := settleGroupCounterpartyChosenAction(whc, dbo4spaceus.SpaceEntry{}, "m1")
			return err
		}},
		{"counterparty_confirmed", func() error {
			_, err := settleGroupCounterpartyConfirmedAction(whc, dbo4spaceus.SpaceEntry{}, "m1", "USD")
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil || !strings.Contains(err.Error(), "not implemented") {
				t.Errorf("expected 'not implemented' error, got: %v", err)
			}
		})
	}
}

func TestInlineQueryNotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	_, err := inlineQueryJoinGroup(whc, "anything")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' error, got: %v", err)
	}
}

func TestInlineEmptyQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	iq := &fakeInlineInput{id: "iq1", query: ""}
	m, err := inlineEmptyQuery(whc, iq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.BotMessage == nil {
		t.Error("expected BotMessage to be set")
	}
}

func TestInlineQueryNewBill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeInput := &fakeInlineInput{id: "iq1", query: "100usd lunch"}
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeInput).AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string { return key }).AnyTimes()

	m, err := inlineQueryNewBill(whc, "100", "usd", "lunch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.BotMessage == nil {
		t.Error("expected BotMessage to be set")
	}
	if !strings.Contains(m.Text, "lunch") {
		t.Errorf("expected bill name in text, got %q", m.Text)
	}
}

func TestInlineQueryHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		name        string
		query       string
		wantHandled bool
	}{
		{"empty", "", true},
		{"join_group", joinSpaceCommandCode + "?id=g1", true},
		{"new_bill", "100usd lunch", true},
		{"no_match", "xyz", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeInput := &fakeInlineInput{id: "iq1", query: c.query}
			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().Context().Return(context.Background()).AnyTimes()
			whc.EXPECT().Input().Return(fakeInput).AnyTimes()
			whc.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
			whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string { return key }).AnyTimes()

			handled, _, _ := InlineQueryHandler(whc, fakeInput)
			if handled != c.wantHandled {
				t.Errorf("handled=%v, want %v", handled, c.wantHandled)
			}
		})
	}
}

func TestGetPublicBillCardInlineKeyboard(t *testing.T) {
	translator := newTestTranslator(context.Background())
	kb := getPublicBillCardInlineKeyboard(translator, "mybot", "bill1")
	if kb == nil || len(kb.InlineKeyboard) == 0 {
		t.Fatal("expected non-empty keyboard")
	}
}

func TestUpdateInlineBillCardMessage(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)
	bill := models4splitus.NewBillEntry("b1", nil)

	t.Run("empty_id", func(t *testing.T) {
		em := tgbotapi.NewEditMessageText(0, 0, "im1", "")
		noID := models4splitus.NewBillEntry("", nil)
		err := updateInlineBillCardMessage(ctx, translator, true, em, noID, "bot", "")
		if err == nil {
			t.Fatal("expected error for empty bill ID")
		}
	})

	t.Run("nil_data", func(t *testing.T) {
		em := tgbotapi.NewEditMessageText(0, 0, "im1", "")
		nilData := models4splitus.NewBillEntry("b1", nil)
		nilData.Data = nil
		err := updateInlineBillCardMessage(ctx, translator, true, em, nilData, "bot", "")
		if err == nil {
			t.Fatal("expected error for nil bill data")
		}
	})

	t.Run("group_chat", func(t *testing.T) {
		em := tgbotapi.NewEditMessageText(0, 0, "im1", "")
		err := updateInlineBillCardMessage(ctx, translator, true, em, bill, "bot", "footer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("private_chat", func(t *testing.T) {
		em := tgbotapi.NewEditMessageText(0, 0, "im1", "")
		err := updateInlineBillCardMessage(ctx, translator, false, em, bill, "bot", "footer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDelayUpdateBillCardOnUserJoin(t *testing.T) {
	ctx := context.Background()
	if err := delayUpdateBillCardOnUserJoin(ctx, "b1", "msg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelayedUpdateBillCards(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	t.Run("not_found", func(t *testing.T) {
		if err := delayedUpdateBillCards(ctx, "missing", "f"); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("success_with_chat_ids", func(t *testing.T) {
		putBill(t, ctx, db, "bd1", &models4splitus.BillDbo{TgChatMessageIDs: []string{"im1@bot@en-US"}})
		if err := delayedUpdateBillCards(ctx, "bd1", "f"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDelayedUpdateBillTgChartCard(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	t.Run("not_found", func(t *testing.T) {
		if err := delayedUpdateBillTgChartCard(ctx, "missing", "im1@bot@en-US", "f"); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("no_bot_settings", func(t *testing.T) {
		putBill(t, ctx, db, "bt1", &models4splitus.BillDbo{})
		// GetBotSettingsByCode for unknown bot returns error -> function returns nil
		err := delayedUpdateBillTgChartCard(ctx, "bt1", "im1@unknownbot@en-US", "f")
		if err != nil {
			t.Fatalf("expected nil (settings error handled), got: %v", err)
		}
	})
}

func TestRegisterCommand(t *testing.T) {
	cases := []struct {
		name         string
		isStandalone bool
	}{
		{"standalone", true},
		{"shared", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var count int
			RegisterCommand(func(commands ...botsfw.Command) { count += len(commands) }, c.isStandalone)
			if count == 0 {
				t.Error("expected commands to be registered")
			}
		})
	}
}

func TestGroupSettingsSetCurrencyCommand(t *testing.T) {
	cmd := groupSettingsSetCurrencyCommand()
	if cmd.Code != GroupSettingsSetCurrencyCommandCode {
		t.Errorf("unexpected command code: %q", cmd.Code)
	}
	if cmd.CallbackAction == nil {
		t.Error("expected CallbackAction to be set")
	}
}

// --- helpers / fakes ---

type fakeInlineInput struct {
	id    string
	query string
	chat  botinput.Chat
}

type fakeChat struct{}

func (fakeChat) GetID() string     { return "chat1" }
func (fakeChat) GetType() string   { return "private" }
func (fakeChat) IsGroupChat() bool { return false }

func (f *fakeInlineInput) GetID() any                       { return f.id }
func (f *fakeInlineInput) GetInlineQueryID() string         { return f.id }
func (f *fakeInlineInput) GetFrom() botinput.Sender         { return nil }
func (f *fakeInlineInput) GetQuery() string                 { return f.query }
func (f *fakeInlineInput) GetOffset() string                { return "" }
func (f *fakeInlineInput) LogRequest()                      {}
func (f *fakeInlineInput) InputType() botinput.Type         { return botinput.TypeInlineQuery }
func (f *fakeInlineInput) MessageIntID() int                { return 0 }
func (f *fakeInlineInput) MessageStringID() string          { return "" }
func (f *fakeInlineInput) BotChatID() (string, error)       { return "", nil }
func (f *fakeInlineInput) Chat() botinput.Chat              { return f.chat }
func (f *fakeInlineInput) GetSender() botinput.User         { return nil }
func (f *fakeInlineInput) GetRecipient() botinput.Recipient { return nil }
func (f *fakeInlineInput) GetTime() time.Time               { return time.Time{} }
