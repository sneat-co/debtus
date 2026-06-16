package dtb_settings

import (
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"go.uber.org/mock/gomock"
)

func TestRegisterCommands(t *testing.T) {
	router := botswebhook.NewWebhookRouter(nil)
	RegisterCommands(router)

	registered := make(map[botsfw.CommandCode]bool)
	for _, byCode := range router.RegisteredCommands() {
		for code := range byCode {
			registered[code] = true
		}
	}
	if !registered[AskCurrencySettingCommandCode] {
		t.Errorf("command %q is not registered", AskCurrencySettingCommandCode)
	}
}

func TestAskCurrencySettingsCommand_Action(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(AskCurrencySettingCommandCode)

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewMessageByCode(gomock.Any()).DoAndReturn(func(code string, _ ...any) (m botmsg.MessageFromBot) {
		m.Text = code
		return m
	})
	whc.EXPECT().ChatData().Return(chatData)

	m, err := AskCurrencySettingsCommand.Action(whc)
	if err != nil {
		t.Fatalf("Action() returned error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected currency keyboard to be set")
	}
}
