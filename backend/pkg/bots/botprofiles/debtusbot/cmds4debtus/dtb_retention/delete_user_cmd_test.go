package dtb_retention

import (
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"go.uber.org/mock/gomock"
)

func setupWhc(t *testing.T, platformID string) *mock_botsfw.MockWebhookContext {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return(platformID).AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	return whc
}

func TestDeleteUserCommand_success(t *testing.T) {
	whc := setupWhc(t, "telegram")
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	orig := setAccessGranted
	setAccessGranted = func(_ botsfw.WebhookContext, _ bool) error { return nil }
	defer func() { setAccessGranted = orig }()

	_, err := DeleteUserCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUserCommand_setAccessGrantedError_menuOK(t *testing.T) {
	whc := setupWhc(t, "telegram")
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{})
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	orig := setAccessGranted
	setAccessGranted = func(_ botsfw.WebhookContext, _ bool) error {
		return errors.New("access denied")
	}
	defer func() { setAccessGranted = orig }()

	_, err := DeleteUserCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUserCommand_setAccessGrantedError_menuFails(t *testing.T) {
	// Use unsupported platform so SetMainMenuKeyboard returns an error.
	whc := setupWhc(t, "viber")
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{})

	orig := setAccessGranted
	setAccessGranted = func(_ botsfw.WebhookContext, _ bool) error {
		return errors.New("access denied")
	}
	defer func() { setAccessGranted = orig }()

	_, err := DeleteUserCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on unsupported platform")
	}
}
