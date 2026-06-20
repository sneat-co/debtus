package dtb_transfer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// fakeTextMsg is a minimal botinput.TextMessage used to drive command actions.
type fakeTextMsg struct{ text string }

func (f fakeTextMsg) GetSender() botinput.User         { return nil }
func (f fakeTextMsg) GetRecipient() botinput.Recipient { return nil }
func (f fakeTextMsg) GetTime() time.Time               { return time.Time{} }
func (f fakeTextMsg) InputType() botinput.Type         { return botinput.TypeText }
func (f fakeTextMsg) MessageIntID() int                { return 0 }
func (f fakeTextMsg) MessageStringID() string          { return "" }
func (f fakeTextMsg) BotChatID() (string, error)       { return "chat1", nil }
func (f fakeTextMsg) Chat() botinput.Chat              { return fakeChatID{} }
func (f fakeTextMsg) LogRequest()                      {}
func (f fakeTextMsg) Text() string                     { return f.text }
func (f fakeTextMsg) IsEdited() bool                   { return false }

var _ botinput.TextMessage = fakeTextMsg{}

// fakeContactMsg implements both botinput.InputMessage and botinput.ContactMessage.
type fakeContactMsg struct{}

func (f fakeContactMsg) GetSender() botinput.User         { return nil }
func (f fakeContactMsg) GetRecipient() botinput.Recipient { return nil }
func (f fakeContactMsg) GetTime() time.Time               { return time.Time{} }
func (f fakeContactMsg) InputType() botinput.Type         { return botinput.TypeContact }
func (f fakeContactMsg) MessageIntID() int                { return 0 }
func (f fakeContactMsg) MessageStringID() string          { return "" }
func (f fakeContactMsg) BotChatID() (string, error)       { return "chat1", nil }
func (f fakeContactMsg) Chat() botinput.Chat              { return fakeChatID{} }
func (f fakeContactMsg) LogRequest()                      {}
func (f fakeContactMsg) GetPhoneNumber() string           { return "+123" }
func (f fakeContactMsg) GetFirstName() string             { return "First" }
func (f fakeContactMsg) GetLastName() string              { return "Last" }
func (f fakeContactMsg) GetBotUserID() string             { return "" }
func (f fakeContactMsg) GetVCard() string                 { return "" }

var (
	_ botinput.InputMessage   = fakeContactMsg{}
	_ botinput.ContactMessage = fakeContactMsg{}
)

type fakeChatID struct{}

func (fakeChatID) GetID() string     { return "chat1" }
func (fakeChatID) GetType() string   { return "private" }
func (fakeChatID) IsGroupChat() bool { return false }

// setupTranslateWhc wires Translate / NewMessage / Locale on a fresh mocked whc.
func setupTranslateWhc(whc *mock_botsfw.MockWebhookContext) {
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}
	}).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).DoAndReturn(func(code string, _ ...any) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: code}}
	}).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).DoAndReturn(func(code string, _ ...any) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: code}}
	}).AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
}

// --- interestAction (transfer_set_interest_cmd.go) ---

func TestInterestAction_NoMatchReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "this is not interest! @#"}).AnyTimes()

	called := false
	next := func(botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		called = true
		return botmsg.MessageFromBot{}, nil
	}

	m, err := interestAction(whc, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text != "" {
		t.Errorf("expected empty message text for non-matching input, got %q", m.Text)
	}
	if called {
		t.Error("next action must not be called when input does not match interest pattern")
	}
}

func TestInterestAction_PeriodMissingAsksForPeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "10"}).AnyTimes()
	setupTranslateWhc(whc)
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	next := func(botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		t.Fatal("next must not be called when period is missing")
		return botmsg.MessageFromBot{}, nil
	}

	m, err := interestAction(whc, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected a please-specify-period message")
	}
}

func TestInterestAction_ValidWithPeriodLetterCallsNext(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"weekly", "5%/w"},
		{"monthly", "5%/m"},
		{"yearly", "5%/y"},
		{"numeric-days", "5%/30"},
		{"with-minimum-grace", "5%/30/10/3"},
		{"with-comment", "5%/30:my comment"},
		{"comma-percent", "5,5%/30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().Input().Return(fakeTextMsg{text: tc.in}).AnyTimes()
			chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
			chatData.EXPECT().AddWizardParam(gomock.Any(), gomock.Any()).AnyTimes()
			whc.EXPECT().ChatData().Return(chatData).AnyTimes()

			called := false
			next := func(botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
				called = true
				return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "next"}}, nil
			}

			m, err := interestAction(whc, next)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !called {
				t.Fatal("expected next action to be called for valid interest input")
			}
			if m.Text != "next" {
				t.Errorf("expected message from next action, got %q", m.Text)
			}
		})
	}
}

// --- TryToProcessHowMuchHasBeenReturned (transfer_return.go) ---

func TestTryToProcessHowMuchHasBeenReturned_NotANumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "not-a-number"}).AnyTimes()
	setupTranslateWhc(whc)

	m, err := TryToProcessHowMuchHasBeenReturned(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "NOT_A_NUMBER") {
		t.Errorf("expected not-a-number message, got %q", m.Text)
	}
}

func TestTryToProcessHowMuchHasBeenReturned_NegativeValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "-5"}).AnyTimes()
	setupTranslateWhc(whc)

	m, err := TryToProcessHowMuchHasBeenReturned(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "IS_NEGATIVE") {
		t.Errorf("expected is-negative message, got %q", m.Text)
	}
}

// --- AskTransferAmountCommand (transfer_common.go) ---

func newAwaitingChatData(ctrl *gomock.Controller, code string) *mock_botsfwmodels.MockBotChatData {
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(code).AnyTimes()
	chatData.EXPECT().IsAwaitingReplyTo(code).Return(true).AnyTimes()
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(false).AnyTimes()
	chatData.EXPECT().GetWizardParam(gomock.Any()).Return("USD").AnyTimes()
	chatData.EXPECT().AddWizardParam(gomock.Any(), gomock.Any()).AnyTimes()
	return chatData
}

func TestAskTransferAmountCommand_InvalidFloat(t *testing.T) {
	const code = "ask_amount"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeTextMsg{text: "abc"}).AnyTimes()
	setupTranslateWhc(whc)
	whc.EXPECT().ChatData().Return(newAwaitingChatData(ctrl, code)).AnyTimes()

	cmd := AskTransferAmountCommand(botsfw.CommandCode(code), "msg-fmt", botsfw.Command{})
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "INVALID_FLOAT") {
		t.Errorf("expected invalid-float message, got %q", m.Text)
	}
	if m.Format != botmsg.FormatHTML {
		t.Error("expected HTML format")
	}
}

func TestAskTransferAmountCommand_ValidAmountCallsNext(t *testing.T) {
	const code = "ask_amount"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeTextMsg{text: "12,34"}).AnyTimes()
	setupTranslateWhc(whc)
	whc.EXPECT().ChatData().Return(newAwaitingChatData(ctrl, code)).AnyTimes()

	called := false
	next := botsfw.Command{
		Code: "next",
		Action: func(botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
			called = true
			return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "next"}}, nil
		},
	}
	cmd := AskTransferAmountCommand(botsfw.CommandCode(code), "msg-fmt", next)
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected next command to be invoked for a valid amount")
	}
	if m.Text != "next" {
		t.Errorf("expected next command output, got %q", m.Text)
	}
}

func TestAskTransferAmountCommand_ContactMessageAsksForAmount(t *testing.T) {
	const code = "ask_amount"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeContactMsg{}).AnyTimes()
	setupTranslateWhc(whc)
	whc.EXPECT().ChatData().Return(newAwaitingChatData(ctrl, code)).AnyTimes()

	cmd := AskTransferAmountCommand(botsfw.CommandCode(code), "msg-fmt", botsfw.Command{})
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected a prompt asking to enter amount")
	}
}

func TestAskTransferAmountCommand_IncorrectlyMatchedReturnsError(t *testing.T) {
	const code = "ask_amount"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	// awaitingReplyTo path contains the code but is not exactly awaiting it.
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return("parent>" + code).AnyTimes()
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(false).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := AskTransferAmountCommand(botsfw.CommandCode(code), "msg-fmt", botsfw.Command{})
	_, err := cmd.Action(whc)
	if err == nil {
		t.Fatal("expected an error when command is incorrectly matched")
	}
	if !strings.Contains(err.Error(), "incorrectly matched") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- transferHistoryRows (transfer_history.go) ---

func TestTransferHistoryRows_RendersRows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("u1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().MustBotChatID().Return("chat1").AnyTimes()
	whc.EXPECT().Environment().Return("local").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()

	td := &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      "USD",
		AmountInCents: 10000,
		FromJson:      `{"userID":"u1","contactID":"c1","contactName":"Me"}`,
		ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
	}
	td.DtCreated = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	transfer := models4debtus.NewTransfer("t1", td)

	// A second transfer where u1 is the counterparty (creator is u2) → the
	// "to user" row template is selected, covering the else branch.
	td2 := &models4debtus.TransferData{
		CreatorUserID: "u2",
		Currency:      "USD",
		AmountInCents: 5000,
		FromJson:      `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
		ToJson:        `{"userID":"u1","contactID":"c1","contactName":"Me"}`,
	}
	td2.DtCreated = time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)
	transfer2 := models4debtus.NewTransfer("t2", td2)

	got := transferHistoryRows(whc, []models4debtus.TransferEntry{transfer, transfer2})
	if !strings.Contains(got, "HISTORY_ROW_FROM_USER_WITH_NAME") {
		t.Errorf("expected the from-user history row, got %q", got)
	}
	if !strings.Contains(got, "HISTORY_ROW_TO_USER_WITH_NAME") {
		t.Errorf("expected the to-user history row, got %q", got)
	}
}

// --- showHistoryCard (transfer_history.go) — empty AppUserID skips DAL ---

func TestShowHistoryCard_NoRecordsForEmptyUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	setupTranslateWhc(whc)

	m, err := showHistoryCard(whc, "", HistoryTopLimit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "NO_RECORDS") {
		t.Errorf("expected no-records message, got %q", m.Text)
	}
	if m.Keyboard == nil {
		t.Error("expected a back keyboard to be set")
	}
	if m.Format != botmsg.FormatHTML {
		t.Error("expected HTML format")
	}
}
