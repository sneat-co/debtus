package cmds4splitusbot

import (
	"context"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

// fakeTgInput is a telegram input exposing a raw tgbotapi.Update.
type fakeTgInput struct {
	fakeInlineInput
	upd *tgbotapi.Update
}

func (f *fakeTgInput) TgUpdate() *tgbotapi.Update { return f.upd }

// fakeTextInput implements botinput.TextMessage.
type fakeTextInput struct {
	fakeInlineInput
	text string
}

func (f *fakeTextInput) Text() string             { return f.text }
func (f *fakeTextInput) IsEdited() bool           { return false }
func (f *fakeTextInput) InputType() botinput.Type { return botinput.TypeText }

// fakeChosenResult implements botinput.ChosenInlineResult.
type fakeChosenResult struct {
	fakeInlineInput
	resultID        string
	inlineMessageID string
}

func (f *fakeChosenResult) GetResultID() string        { return f.resultID }
func (f *fakeChosenResult) GetInlineMessageID() string { return f.inlineMessageID }
func (f *fakeChosenResult) InputType() botinput.Type   { return botinput.TypeChosenInlineResult }

// fakeAppUser implements botsfwmodels.AppUserData and the User interface
// asserted inside joinBillAction.
type fakeAppUser struct {
	fullName        string
	primaryCurrency string
	lastCurrencies  []string
}

func (f *fakeAppUser) BotsFwAdapter() botsfwmodels.AppUserAdapter {
	return (&dbo4userus.UserDbo{}).BotsFwAdapter()
}
func (f *fakeAppUser) FullName() string           { return f.fullName }
func (f *fakeAppUser) GetPrimaryCurrency() string { return f.primaryCurrency }
func (f *fakeAppUser) GetLastCurrencies() []string {
	return f.lastCurrencies
}

// adapterOnlyAppUser implements only botsfwmodels.AppUserData.
type adapterOnlyAppUser struct{}

func (adapterOnlyAppUser) BotsFwAdapter() botsfwmodels.AppUserAdapter {
	return (&dbo4userus.UserDbo{}).BotsFwAdapter()
}

func TestGetBillIDFromUrlInEditedMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newWhcWithInput := func(input botinput.InputMessage) *mock_botsfw.MockWebhookContext {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(input).AnyTimes()
		return whc
	}

	t.Run("not_telegram_input", func(t *testing.T) {
		whc := newWhcWithInput(&fakeInlineInput{})
		if got := getBillIDFromUrlInEditedMessage(whc); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_edited_message", func(t *testing.T) {
		whc := newWhcWithInput(&fakeTgInput{upd: &tgbotapi.Update{}})
		if got := getBillIDFromUrlInEditedMessage(whc); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_entities", func(t *testing.T) {
		whc := newWhcWithInput(&fakeTgInput{upd: &tgbotapi.Update{EditedMessage: &tgbotapi.Message{}}})
		if got := getBillIDFromUrlInEditedMessage(whc); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("with_bill_link", func(t *testing.T) {
		entities := []tgbotapi.MessageEntity{
			{Type: "bold"},
			{Type: "text_link", URL: "https://t.me/testbot?start=bill-12345"},
		}
		whc := newWhcWithInput(&fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &entities},
		}})
		if got := getBillIDFromUrlInEditedMessage(whc); got != "12345" {
			t.Errorf("expected 12345, got %q", got)
		}
	})

	t.Run("non_matching_link", func(t *testing.T) {
		entities := []tgbotapi.MessageEntity{
			{Type: "text_link", URL: "https://t.me/testbot?start=other"},
		}
		whc := newWhcWithInput(&fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &entities},
		}})
		if got := getBillIDFromUrlInEditedMessage(whc); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestEditedBillCardHookCommand(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "12345", nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	billEntities := []tgbotapi.MessageEntity{
		{Type: "text_link", URL: "https://t.me/testbot?start=bill-12345"},
	}
	billInput := func() *fakeTgInput {
		return &fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &billEntities},
		}}
	}

	t.Run("matcher_is_in_group_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if EditedBillCardHookCommand.Matcher(EditedBillCardHookCommand, whc) {
			t.Error("expected false")
		}
	})

	t.Run("matcher_in_group_with_bill", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil)
		whc.EXPECT().Input().Return(billInput()).AnyTimes()
		if !EditedBillCardHookCommand.Matcher(EditedBillCardHookCommand, whc) {
			t.Error("expected true")
		}
	})

	t.Run("action_no_bill_id", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeTgInput{upd: &tgbotapi.Update{}}).AnyTimes()
		if _, err := EditedBillCardHookCommand.Action(whc); err == nil {
			t.Fatal("expected error for empty billID")
		}
	})

	t.Run("action_no_group_id", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(billInput()).AnyTimes()
		// ChatData is nil (preset) -> GetUserGroupID returns empty -> warning branch
		if _, err := EditedBillCardHookCommand.Action(whc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestChosenInlineResultCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("not_a_bill_result", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeChosenResult{resultID: "other?x=1"}).AnyTimes()
		if _, err := chosenInlineResultCommand.Action(whc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCreateBillFromInlineChosenResult(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	_ = ctx
	_ = db

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("unexpected_result_id", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := createBillFromInlineChosenResult(whc, &fakeChosenResult{resultID: "nope"}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bad_query_params", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := createBillFromInlineChosenResult(whc, &fakeChosenResult{resultID: "bill?%zz"}); err == nil {
			t.Fatal("expected query parse error")
		}
	})

	t.Run("set_locale_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().SetLocale("ru-RU").Return(errTest)
		if _, err := createBillFromInlineChosenResult(whc, &fakeChosenResult{resultID: "bill?lng=ru-RU&amount=100usd"}); err == nil {
			t.Fatal("expected set locale error")
		}
	})

	t.Run("amount_parse_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		// 320 digits overflow float64 range -> strconv.ParseFloat error
		huge := strings.Repeat("9", 320)
		cr := &fakeChosenResult{resultID: "bill?amount=" + huge + "usd"}
		cr.query = huge + "usd dinner"
		if _, err := createBillFromInlineChosenResult(whc, cr); err == nil {
			t.Fatal("expected amount parse error")
		}
	})

	t.Run("create_bill_error_zero_amount", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		cr := &fakeChosenResult{resultID: "bill?amount=0usd"}
		if _, err := createBillFromInlineChosenResult(whc, cr); err == nil {
			t.Fatal("expected create bill error for zero amount")
		}
	})

	t.Run("success_send_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, errTest)
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		cr := &fakeChosenResult{resultID: "bill?amount=100usd", inlineMessageID: "im1"}
		cr.query = "100usd dinner"
		if _, err := createBillFromInlineChosenResult(whc, cr); err == nil {
			t.Fatal("expected send error")
		}
	})

	t.Run("success", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil)
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		cr := &fakeChosenResult{resultID: "bill?amount=100usd", inlineMessageID: "im1"}
		// no name in query -> trans.NO_NAME branch
		if _, err := createBillFromInlineChosenResult(whc, cr); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("via_command_action", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		cr := &fakeChosenResult{resultID: "bill?amount=200usd", inlineMessageID: "im2"}
		cr.query = "200usd lunch"
		whc.EXPECT().Input().Return(cr).AnyTimes()
		if _, err := chosenInlineResultCommand.Action(whc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestInlineQueryHandlerTelegramBranches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newTgInput := func(lang string) *fakeTgInput {
		return &fakeTgInput{
			fakeInlineInput: fakeInlineInput{id: "iq1", query: ""},
			upd: &tgbotapi.Update{
				InlineQuery: &tgbotapi.InlineQuery{From: &tgbotapi.User{LanguageCode: lang}},
			},
		}
	}

	t.Run("app_user_data_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := newTgInput("en")
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().AppUserData().Return(nil, errTest)
		if _, _, err := InlineQueryHandler(whc, input); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("preferred_locale", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := newTgInput("en")
		whc.EXPECT().Input().Return(input).AnyTimes()
		userDbo := &dbo4userus.UserDbo{}
		userDbo.PreferredLocale = "ru-RU"
		whc.EXPECT().AppUserData().Return(userDbo, nil)
		whc.EXPECT().SetLocale("ru-RU").Return(nil)
		handled, _, err := InlineQueryHandler(whc, input)
		if err != nil || !handled {
			t.Fatalf("expected handled empty query, got handled=%v err=%v", handled, err)
		}
	})

	t.Run("ru_language_set_locale_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := newTgInput("ru")
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().AppUserData().Return(&dbo4userus.UserDbo{}, nil)
		whc.EXPECT().SetLocale(i18n.LocaleCodeRuRU).Return(errTest)
		if _, _, err := InlineQueryHandler(whc, input); err == nil {
			t.Fatal("expected set-locale error")
		}
	})

	t.Run("ru_language_ok", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := newTgInput("ru")
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().AppUserData().Return(&dbo4userus.UserDbo{}, nil)
		whc.EXPECT().SetLocale(i18n.LocaleCodeRuRU).Return(nil)
		handled, _, err := InlineQueryHandler(whc, input)
		if err != nil || !handled {
			t.Fatalf("expected handled, got handled=%v err=%v", handled, err)
		}
	})
}

func TestJoinBillCommand(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// AddBillMember loads the splitus space of the bill, so bills that reach it
	// must reference a seeded space.
	splitusSpace := models4splitus.NewSplitusSpaceEntry("spaceJ")
	putRecord(t, ctx, db, splitusSpace.Record)
	withSpace := func(mutate func(*models4splitus.BillDbo)) func(*models4splitus.BillDbo) {
		return func(dbo *models4splitus.BillDbo) {
			dbo.SpaceID = "spaceJ"
			if mutate != nil {
				mutate(dbo)
			}
		}
	}

	appUser := &fakeAppUser{fullName: "John Doe", primaryCurrency: "USD"}

	newJoinWhc := func(input botinput.InputMessage, user botsfwmodels.AppUserData) *mock_botsfw.MockWebhookContext {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		if user != nil {
			whc.EXPECT().AppUserData().Return(user, nil).AnyTimes()
		}
		whc.EXPECT().SetContext(gomock.Any()).AnyTimes()
		return whc
	}

	t.Run("missing_bill_id", func(t *testing.T) {
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-"}, nil)
		if _, err := joinBillCommand.Action(whc); err == nil {
			t.Fatal("expected missing bill ID error")
		}
	})

	t.Run("bill_not_found", func(t *testing.T) {
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-missing"}, nil)
		if _, err := joinBillCommand.Action(whc); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("app_user_data_error", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj0", nil)
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj0"}, nil)
		whc.EXPECT().AppUserData().Return(nil, errTest).AnyTimes()
		if _, err := joinBillCommand.Action(whc); err == nil {
			t.Fatal("expected app user data error")
		}
	})

	t.Run("cast_failure", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj1", nil)
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj1"}, adapterOnlyAppUser{})
		if _, err := joinBillCommand.Action(whc); err == nil || !strings.Contains(err.Error(), "cast") {
			t.Fatalf("expected cast error, got: %v", err)
		}
	})

	t.Run("empty_user_name", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj2", nil)
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj2"}, &fakeAppUser{})
		if _, err := joinBillCommand.Action(whc); err == nil || !strings.Contains(err.Error(), "userName") {
			t.Fatalf("expected userName error, got: %v", err)
		}
	})

	// NOTE: BillCommon.AddOrGetMember checks `index != len(billMembers)-1`
	// against a nil named return value, so adding a NEW member to a bill always
	// panics. The statements of joinBillAction up to facade4splitus.AddBillMember
	// are still exercised; the remainder is unreachable (production bug).
	expectAddMemberPanic := func(t *testing.T, f func() error) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic from BillCommon.AddOrGetMember")
			}
		}()
		_ = f()
	}

	t.Run("join_with_currency", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj3", withSpace(nil))
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj3"}, appUser)
		expectAddMemberPanic(t, func() error {
			_, err := joinBillCommand.Action(whc)
			return err
		})
	})

	t.Run("currency_from_primary", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj4", withSpace(func(dbo *models4splitus.BillDbo) { dbo.Currency = "" }))
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj4"}, appUser)
		expectAddMemberPanic(t, func() error {
			_, err := joinBillCommand.Action(whc)
			return err
		})
	})

	t.Run("currency_from_last_currencies", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj5", withSpace(func(dbo *models4splitus.BillDbo) { dbo.Currency = "" }))
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj5"}, &fakeAppUser{fullName: "Jane", lastCurrencies: []string{"EUR"}})
		expectAddMemberPanic(t, func() error {
			_, err := joinBillCommand.Action(whc)
			return err
		})
	})

	t.Run("currency_guessed_from_locale", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj6", withSpace(func(dbo *models4splitus.BillDbo) { dbo.Currency = "" }))
		whc := newJoinWhc(&fakeTextInput{text: "/start join_bill-bj6"}, &fakeAppUser{fullName: "Jane"})
		expectAddMemberPanic(t, func() error {
			_, err := joinBillCommand.Action(whc)
			return err
		})
	})

	t.Run("callback_paid", func(t *testing.T) {
		putValidBill(t, ctx, db, "bj7", withSpace(nil))
		whc := newJoinWhc(&fakeTextInput{}, appUser)
		expectAddMemberPanic(t, func() error {
			_, err := joinBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?bill=bj7&i=paid"))
			return err
		})
	})

	t.Run("already_member_text_unchanged", func(t *testing.T) {
		member := billMember("m1", "John Doe")
		member.UserID = "u1"
		putValidBill(t, ctx, db, "bj8", withSpace(func(dbo *models4splitus.BillDbo) {
			dbo.Members = []*briefs4splitus.BillMemberBrief{member}
		}))
		input := &fakeTgTextInput{
			fakeTextInput: fakeTextInput{text: "/start join_bill-bj8"},
			upd: &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{Text: "some old text"},
			}},
		}
		whc := newJoinWhc(input, appUser)
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		if _, err := joinBillCommand.Action(whc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("already_member_send_error", func(t *testing.T) {
		member := billMember("m1", "John Doe")
		member.UserID = "u1"
		putValidBill(t, ctx, db, "bj9", withSpace(func(dbo *models4splitus.BillDbo) {
			dbo.Members = []*briefs4splitus.BillMemberBrief{member}
		}))
		input := &fakeTgTextInput{
			fakeTextInput: fakeTextInput{text: "/start join_bill-bj9"},
			upd: &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{Text: "differs from card"},
			}},
		}
		whc := newJoinWhc(input, appUser)
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, errTest).AnyTimes()
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		_, err := joinBillCommand.Action(whc)
		if err == nil {
			t.Fatal("expected send error")
		}
	})
}

// fakeTgTextInput is a text message that also exposes a telegram update.
type fakeTgTextInput struct {
	fakeTextInput
	upd *tgbotapi.Update
}

func (f *fakeTgTextInput) TgUpdate() *tgbotapi.Update { return f.upd }

// fakeCallbackInput implements botinput.InputMessage, botinput.CallbackQuery
// and telegram.WebhookCallbackQuery.
type fakeCallbackInput struct {
	inlineMessageID string
}

func (f *fakeCallbackInput) GetID() string                { return "cb1" }
func (f *fakeCallbackInput) GetInlineMessageID() string   { return f.inlineMessageID }
func (f *fakeCallbackInput) GetChatInstanceID() string    { return "" }
func (f *fakeCallbackInput) GetFrom() botinput.Sender     { return nil }
func (f *fakeCallbackInput) GetMessage() botinput.Message { return nil }
func (f *fakeCallbackInput) GetData() string              { return "" }
func (f *fakeCallbackInput) Chat() botinput.Chat          { return nil }
func (f *fakeCallbackInput) GetSender() botinput.User     { return nil }
func (f *fakeCallbackInput) GetRecipient() botinput.Recipient {
	return nil
}
func (f *fakeCallbackInput) GetTime() time.Time       { return time.Time{} }
func (f *fakeCallbackInput) InputType() botinput.Type { return botinput.TypeCallbackQuery }
func (f *fakeCallbackInput) MessageIntID() int        { return 0 }
func (f *fakeCallbackInput) MessageStringID() string  { return "" }
func (f *fakeCallbackInput) BotChatID() (string, error) {
	return "", nil
}
func (f *fakeCallbackInput) LogRequest() {}

func TestNewBillCommandCallback(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	_ = ctx
	_ = db

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userWithName := func() *dbo4userus.UserDbo {
		u := &dbo4userus.UserDbo{}
		u.Names = &person.NameFields{FirstName: "John", LastName: "Doe"}
		return u
	}

	t.Run("invalid_param_i", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := newBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?i=other")); err == nil {
			t.Fatal("expected param error")
		}
	})

	t.Run("invalid_amount", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := newBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?i=owe&v=abc")); err == nil {
			t.Fatal("expected amount parse error")
		}
	})

	t.Run("no_user_name", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeCallbackInput{inlineMessageID: "im1"}).AnyTimes()
		noNameUser := &dbo4userus.UserDbo{}
		noNameUser.Names = &person.NameFields{}
		whc.EXPECT().AppUserData().Return(noNameUser, nil).AnyTimes()
		if _, err := newBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?i=paid&v=10")); err == nil {
			t.Fatal("expected no-name error")
		}
	})

	t.Run("app_user_data_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeCallbackInput{inlineMessageID: "im1"}).AnyTimes()
		whc.EXPECT().AppUserData().Return(nil, errTest).AnyTimes()
		if _, err := newBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?i=paid&v=10")); err == nil {
			t.Fatal("expected app user data error")
		}
	})

	t.Run("set_bill_members_error", func(t *testing.T) {
		// The production code constructs the bill member with only Paid set
		// (no Name), so SetBillMembers always fails and the transaction branch
		// below it is unreachable.
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeCallbackInput{inlineMessageID: "im1"}).AnyTimes()
		whc.EXPECT().AppUserData().Return(userWithName(), nil).AnyTimes()
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		_, err := newBillCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?i=paid&v=10"))
		if err == nil || !strings.Contains(err.Error(), "no name") {
			t.Fatalf("expected SetBillMembers no-name error, got: %v", err)
		}
	})
}
