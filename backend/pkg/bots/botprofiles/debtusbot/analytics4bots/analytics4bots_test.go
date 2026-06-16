package analytics4bots

import (
	"context"
	"testing"

	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"go.uber.org/mock/gomock"
)

func TestReceiptSentFromBot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalytics := mock_botsfw.NewMockWebhookAnalytics(ctrl)
	mockAnalytics.EXPECT().Enqueue(gomock.Any()).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Analytics().Return(mockAnalytics).AnyTimes()

	ReceiptSentFromBot(whc, "Telegram")
}
