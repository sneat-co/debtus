package delayed4debtus

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"go.uber.org/mock/gomock"
)

// setBotSettingsProvider registers a provider returning a single bot keyed by code.
func setBotSettingsProvider(t *testing.T, code, token string) {
	t.Helper()
	settings := &botsfw.BotSettings{Code: code, Token: token}
	provider := func(_ context.Context) botsfw.BotSettingsBy {
		return botsfw.BotSettingsBy{
			ByCode: map[string]*botsfw.BotSettings{code: settings},
		}
	}
	botsettings.SetBotSettingsProvider(provider)
	t.Cleanup(func() { botsettings.SetBotSettingsProvider(nil) })
}

// stubHttpClient overrides dal4debtus.Default.HttpClient to use the given round tripper.
func stubHttpClient(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	orig := dal4debtus.Default.HttpClient
	dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
		return &http.Client{Transport: rt}
	}
	t.Cleanup(func() { dal4debtus.Default.HttpClient = orig })
}

func TestEditTgMessageText_success(t *testing.T) {
	ctx := context.Background()
	const code = "DebtusBot"
	setBotSettingsProvider(t, code, "fake:TOKEN")

	// Stub the seam sendToTelegramFn so no real I/O happens.
	orig := sendToTelegramFn
	var called bool
	sendToTelegramFn = func(_ context.Context, _ tgbotapi.Sendable, _ botsfw.BotSettings) error {
		called = true
		return nil
	}
	t.Cleanup(func() { sendToTelegramFn = orig })

	if err := editTgMessageText(ctx, code, 123, 456, "hello"); err != nil {
		t.Fatalf("editTgMessageText() error = %v", err)
	}
	if !called {
		t.Error("sendToTelegramFn was not called")
	}
}

func TestEditTgMessageText_sendError(t *testing.T) {
	ctx := context.Background()
	const code = "DebtusBot"
	setBotSettingsProvider(t, code, "fake:TOKEN")

	wantErr := errors.New("send failed")
	orig := sendToTelegramFn
	sendToTelegramFn = func(_ context.Context, _ tgbotapi.Sendable, _ botsfw.BotSettings) error {
		return wantErr
	}
	t.Cleanup(func() { sendToTelegramFn = orig })

	if err := editTgMessageText(ctx, code, 123, 456, "hello"); !errors.Is(err, wantErr) {
		t.Fatalf("editTgMessageText() error = %v, want %v", err, wantErr)
	}
}

func TestGetTelegramBotApiByBotCode_success(t *testing.T) {
	ctx := context.Background()
	const code = "DebtusBot"
	setBotSettingsProvider(t, code, "fake:TOKEN")
	stubHttpClient(t, fakeRoundTripper{})

	api, err := getTelegramBotApiByBotCode(ctx, code)
	if err != nil {
		t.Fatalf("getTelegramBotApiByBotCode() error = %v", err)
	}
	if api == nil {
		t.Fatal("getTelegramBotApiByBotCode() returned nil api")
	}
}

func TestSendToTelegram(t *testing.T) {
	ctx := context.Background()
	settings := botsfw.BotSettings{Token: "fake:TOKEN"}
	msg := tgbotapi.NewEditMessageText(123, 1, "", "text")

	t.Run("success", func(t *testing.T) {
		stubHttpClient(t, fakeRoundTripper{})
		if err := sendToTelegram(ctx, msg, settings); err != nil {
			t.Fatalf("sendToTelegram() error = %v", err)
		}
	})

	t.Run("send_error", func(t *testing.T) {
		// Telegram API responds with ok:false → tgApi.Send returns an error.
		stubHttpClient(t, fakeRoundTripper{
			responseBody: `{"ok":false,"error_code":400,"description":"Bad Request"}`,
			statusCode:   http.StatusBadRequest,
		})
		if err := sendToTelegram(ctx, msg, settings); err == nil {
			t.Fatal("sendToTelegram() expected error, got nil")
		}
	})
}

// ---- DiscardReminder / DelayedDiscardReminderForTransfer wrappers -----------

// mockDBRunTx returns a MockDB whose RunReadwriteTransaction either returns
// dbErr directly (when invokeCallback is false) or invokes the callback with txFn.
func mockDBRunTx(ctrl *gomock.Controller, invokeCallback bool, dbErr error, tx dal.ReadwriteTransaction) *mock_dal.MockDB {
	db := mock_dal.NewMockDB(ctrl)
	if invokeCallback {
		db.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
				return f(ctx, tx)
			},
		)
	} else {
		db.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return(dbErr)
	}
	return db
}

func TestDiscardReminder_wrapper(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wantErr := errors.New("getmulti failed")
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(wantErr)

	origGetDB := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return mockDBRunTx(ctrl, true, nil, mockTx), nil
	}
	t.Cleanup(func() { facade.GetSneatDB = origGetDB })

	if err := DiscardReminder(ctx, "rem1", "t1", ""); !errors.Is(err, wantErr) {
		t.Fatalf("DiscardReminder() error = %v, want %v", err, wantErr)
	}
}

func TestDelayedDiscardReminderForTransfer_wrapper(t *testing.T) {
	ctx := context.Background()

	t.Run("propagates_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		wantErr := errors.New("getmulti failed")
		mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
		mockTx.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(wantErr)

		origGetDB := facade.GetSneatDB
		facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
			return mockDBRunTx(ctrl, true, nil, mockTx), nil
		}
		t.Cleanup(func() { facade.GetSneatDB = origGetDB })

		if err := DelayedDiscardReminderForTransfer(ctx, "rem1", "t1", ""); !errors.Is(err, wantErr) {
			t.Fatalf("DelayedDiscardReminderForTransfer() error = %v, want %v", err, wantErr)
		}
	})
}
