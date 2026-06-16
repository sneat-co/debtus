package dtb_transfer

import (
	"context"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"go.uber.org/mock/gomock"
)

func TestGetTransferSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("mybot").AnyTimes()
	whc.EXPECT().MustBotChatID().Return("12345").AnyTimes()

	src := GetTransferSource(whc)
	if src == nil {
		t.Fatal("expected non-nil TransferSource")
	}
}

func TestCallbackTransferHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}
	})

	m, err := callbackTransferHistory(whc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "TODO") {
		t.Errorf("expected message to contain TODO, got %q", m.Text)
	}
}

func TestTransferWizardCompletedCommand_UnknownCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return("some-path").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := TransferWizardCompletedCommand("invalid-code")
	_, err := cmd.Action(whc)
	if err == nil {
		t.Fatal("expected an error for unknown code, got nil")
	}
	if !strings.Contains(err.Error(), "unknown code") {
		t.Errorf("unexpected error: %v", err)
	}
}
