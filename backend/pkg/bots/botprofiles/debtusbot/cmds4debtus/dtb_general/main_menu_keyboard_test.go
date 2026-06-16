package dtb_general

import (
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"go.uber.org/mock/gomock"
)

func TestSetMainMenuKeyboard(t *testing.T) {
	newWhc := func(t *testing.T, platformID string) *mock_botsfw.MockWebhookContext {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		platform := mock_botsfw.NewMockBotPlatform(ctrl)
		platform.EXPECT().ID().Return(platformID).AnyTimes()
		whc := mock_botsfw.NewMockWebhookContext(ctrl)
		whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
		whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
		whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
		whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
		return whc
	}

	t.Run("telegram_sets_keyboard", func(t *testing.T) {
		whc := newWhc(t, "telegram")
		var m botmsg.MessageFromBot
		if err := SetMainMenuKeyboard(whc, &m); err != nil {
			t.Fatalf("SetMainMenuKeyboard() returned error: %v", err)
		}
		if m.Keyboard == nil {
			t.Error("expected keyboard to be set for telegram")
		}
	})

	t.Run("unsupported_platform_returns_error", func(t *testing.T) {
		whc := newWhc(t, "viber")
		var m botmsg.MessageFromBot
		if err := SetMainMenuKeyboard(whc, &m); err == nil {
			t.Fatal("expected error for unsupported platform, got nil")
		}
		if m.Keyboard != nil {
			t.Error("expected no keyboard for unsupported platform")
		}
	})
}
