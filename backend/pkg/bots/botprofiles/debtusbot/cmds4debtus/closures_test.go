package cmds4debtus

import (
	"context"
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// fakeBotPlatform satisfies botsfw.BotPlatform with a configurable ID.
type fakeBotPlatform struct{ id string }

func (f *fakeBotPlatform) ID() string      { return f.id }
func (f *fakeBotPlatform) Version() string { return "1.0" }

// --- botParams closures ---

func TestBotParams_StartInGroupAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	// no expectations needed — closure body does not call whc
	m, err := botParams.StartInGroupAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected non-empty Text from StartInGroupAction")
	}
}

func TestBotParams_GetWelcomeMessageText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	// no expectations needed — closure body does not call whc
	text, err := botParams.GetWelcomeMessageText(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Error("expected non-empty text from GetWelcomeMessageText")
	}
}

func TestBotParams_HelpCommandAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	// HelpCommandAction calls getUserReportUrl(whc, ...) which calls whc.Locale()
	// then whc.Translate(...) multiple times, and whc.NewMessageByCode(...)
	locale := i18n.Locale{Code5: "en-US"}
	whc.EXPECT().Locale().Return(locale).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()
	_, err := botParams.HelpCommandAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBotParams_SetMainMenu_telegram(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	// SetMainMenuKeyboard calls whc.BotPlatform().ID(), then telegram keyboard helpers
	// which call whc.Translate/CommandText/BotPlatform/etc.
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	m, err := botParams.SetMainMenu(whc, "hello", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

func TestBotParams_SetMainMenu_unsupportedPlatform(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "unknown"}).AnyTimes()
	_, err := botParams.SetMainMenu(whc, "hello", false)
	if err == nil {
		t.Fatal("expected error for unsupported platform")
	}
}

// --- newChatMembersCommand.Action ---

func TestNewChatMembersCommand_notInGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	m, err := newChatMembersCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

func TestNewChatMembersCommand_isInGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().IsInGroup().Return(true, nil).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	m, err := newChatMembersCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.BotMessage == nil {
		t.Error("expected BotMessage (LeaveChat) for group context")
	}
}

func TestNewChatMembersCommand_isInGroupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().IsInGroup().Return(false, errors.New("is-in-group error")).AnyTimes()
	_, err := newChatMembersCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from IsInGroup")
	}
}

// ensure imports are used
var _ botsfw.WebhookContext
