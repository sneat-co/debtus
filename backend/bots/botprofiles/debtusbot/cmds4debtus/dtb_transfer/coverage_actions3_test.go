package dtb_transfer

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"go.uber.org/mock/gomock"
)

// --- ProcessReturnAnswer (callback_reminder_return.go) ---

func TestProcessReturnAnswer_MissingReminderReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	u, _ := url.Parse("debt-returned?how-much=fully")
	_, err := ProcessReturnAnswer(whc, u)
	if err == nil {
		t.Fatal("expected an error when reminder id is missing")
	}
	if !strings.Contains(err.Error(), "missing reminder") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- SetDueDateCommand (transfer_create.go) ---

func TestSetDueDateCommand_ClearsAwaitingReply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("").Times(1)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	setupTranslateWhc(whc)

	m, err := SetDueDateCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected a confirmation message")
	}
}

// --- createStartTransferWizardCommand (transfer_create.go) ---

func newStartWizardCmd(code string) botsfw.Command {
	return createStartTransferWizardCommand(botsfw.CommandCode(code), "msg-text", []string{"gave"}, botsfw.Command{})
}

func TestStartTransferWizard_Matcher(t *testing.T) {
	const code = "start-wizard"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(code).AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "💿"}).AnyTimes() // CD_ICON is a currency icon
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := newStartWizardCmd(code)
	if !cmd.Matcher(cmd, whc) {
		t.Error("expected matcher to match a currency icon while awaiting the wizard code")
	}
}

func TestStartTransferWizard_MatcherNoMatch(t *testing.T) {
	const code = "start-wizard"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextMsg{text: "not-an-icon"}).AnyTimes()
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(code).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := newStartWizardCmd(code)
	if cmd.Matcher(cmd, whc) {
		t.Error("expected matcher not to match non-currency text")
	}
}

func TestStartTransferWizard_NotImplementedEllipsis(t *testing.T) {
	const code = "start-wizard"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeTextMsg{text: "..."}).AnyTimes()
	setupTranslateWhc(whc)
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := newStartWizardCmd(code)
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "NOT_IMPLEMENTED") {
		t.Errorf("expected not-implemented message, got %q", m.Text)
	}
}

func TestStartTransferWizard_NumericCurrencyName(t *testing.T) {
	const code = "start-wizard"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeTextMsg{text: "123"}).AnyTimes()
	setupTranslateWhc(whc)
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	cmd := newStartWizardCmd(code)
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "CURRENCY_NAME_IS_NUMBER") {
		t.Errorf("expected currency-name-is-number message, got %q", m.Text)
	}
}
