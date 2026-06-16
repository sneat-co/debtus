package debtusbot

import (
	"context"
	"testing"

	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

func TestNewDebtusTelegramChatRecord(t *testing.T) {
	r := NewDebtusTelegramChatRecord()
	if r == nil {
		t.Fatal("expected non-nil record")
	}
}

func TestGetNewDebtPageUrl(t *testing.T) {
	origIssue := token4auth.IssueAuthToken
	defer func() { token4auth.IssueAuthToken = origIssue }()
	token4auth.IssueAuthToken = func(_ context.Context, _, _ string) (string, error) {
		return "test-token", nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPlatform := mock_botsfw.NewMockBotPlatform(ctrl)
	mockPlatform.EXPECT().ID().Return("telegram").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("DebtusBot").AnyTimes()
	whc.EXPECT().BotPlatform().Return(mockPlatform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()

	url := GetNewDebtPageUrl(whc, models4debtus.TransferDirectionUser2Counterparty, "test-campaign")
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}
