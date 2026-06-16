package dtb_admin

import (
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"go.uber.org/mock/gomock"
)

type fakeCommandsRegisterer struct{ count int }

func (f *fakeCommandsRegisterer) RegisterCommands(cmds ...botsfw.Command) { f.count += len(cmds) }

var _ botswebhook.CommandsRegisterer = (*fakeCommandsRegisterer)(nil)

func TestRegisterCommands(t *testing.T) {
	r := &fakeCommandsRegisterer{}
	RegisterCommands(r)
	if r.count == 0 {
		t.Error("expected RegisterCommands to register at least one command")
	}
}

func TestAdminCommandAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewMessage("Admin menu").Return(botmsg.MessageFromBot{})
	_, err := adminCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
