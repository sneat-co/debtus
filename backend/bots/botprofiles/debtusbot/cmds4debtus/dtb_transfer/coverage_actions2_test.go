package dtb_transfer

import (
	"context"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/sneat-co/sneat-translations/emoji"
	"go.uber.org/mock/gomock"
)

// --- cancelTransferWizardCommandAction (transfer_cancel_cmd.go) ---

func TestCancelTransferWizardCommandAction_UnsupportedPlatformReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("").Times(1)

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	setupTranslateWhc(whc)

	_, err := cancelTransferWizardCommandAction(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on unsupported platform")
	}
	if !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCancelTransferWizardCommandAction_TelegramSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("").Times(1)

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	setupTranslateWhc(whc)

	m, err := cancelTransferWizardCommandAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "CANCELED") {
		t.Errorf("expected cancellation message, got %q", m.Text)
	}
	if m.Keyboard == nil {
		t.Error("expected the main-menu keyboard to be set")
	}
}

// --- processSetDate (transfer_create.go) ---

func TestProcessSetDate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		in       string
		wantZero bool
		wantMsg  string
	}{
		{"valid-4digit-year", "15.01.2024", false, ""},
		{"valid-2digit-year", "15/01/24", false, ""},
		{"mismatched-separators", "15.01/2024", true, "INVALID_DATE"},
		{"invalid-year-length", "15.01.202", true, "INVALID_YEAR"},
		{"not-a-date", "hello there", true, "INVALID_DATE"},
		{"day-too-big", "55.01.2024", true, "WRONG_DATE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().Context().Return(context.Background()).AnyTimes()
			whc.EXPECT().Input().Return(fakeTextMsg{text: tc.in}).AnyTimes()
			setupTranslateWhc(whc)

			m, date, err := processSetDate(whc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantZero && !date.IsZero() {
				t.Errorf("expected zero date, got %v", date)
			}
			if !tc.wantZero && date.IsZero() {
				t.Errorf("expected a parsed date, got zero")
			}
			if tc.wantMsg != "" && !strings.Contains(m.Text, tc.wantMsg) {
				t.Errorf("expected message %q, got %q", tc.wantMsg, m.Text)
			}
		})
	}
}

// --- transferAskDueDateCommand (transfer_create.go) ---

func TestTransferAskDueDateCommand_NotAwaitingShowsKeyboard(t *testing.T) {
	const code = "ask_due"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslateWhc(whc)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(code).Return(false).AnyTimes()
	chatData.EXPECT().PushStepToAwaitingReplyTo(code).Times(1)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := transferAskDueDateCommand(botsfw.CommandCode(code), botsfw.Command{})
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected a reply keyboard offering due-date presets")
	}
}

func TestTransferAskDueDateCommand_PresetCallsNext(t *testing.T) {
	const code = "ask_due"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	// The translate stub returns the key, so "in_few_minutes" preset text equals its key.
	setupTranslateWhc(whc)
	// Input text matches the translated "tomorrow" preset (which is the key itself).
	whc.EXPECT().Input().Return(fakeTextMsg{text: "COMMAND_TEXT_TOMORROW"}).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(code).Return(true).AnyTimes()
	chatData.EXPECT().AddWizardParam("due", gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	called := false
	next := botsfw.Command{Action: func(botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		called = true
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "next"}}, nil
	}}

	m, err := cmd2Action(transferAskDueDateCommand(botsfw.CommandCode(code), next), whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected next command to be called for a recognized preset")
	}
	if m.Text != "next" {
		t.Errorf("expected next command output, got %q", m.Text)
	}
}

func TestTransferAskDueDateCommand_CalendarIconAsksForDate(t *testing.T) {
	const code = "ask_due"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslateWhc(whc)
	whc.EXPECT().Input().Return(fakeTextMsg{text: emoji.CALENDAR_ICON + " set date"}).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(code).Return(true).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := transferAskDueDateCommand(botsfw.CommandCode(code), botsfw.Command{})
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "ASK_DUE_DATE") {
		t.Errorf("expected ask-due-date message, got %q", m.Text)
	}
}

// cmd2Action is a tiny helper to invoke a command's Action.
func cmd2Action(cmd botsfw.Command, whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
	return cmd.Action(whc)
}
