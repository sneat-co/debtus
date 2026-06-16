package debtussender

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsdal"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/strongo/i18n"
)

// fakeResponder is a minimal WebhookResponder that returns a preset error.
type fakeResponder struct {
	err error
}

func (f fakeResponder) SendMessage(_ context.Context, _ botmsg.MessageFromBot, _ botmsg.BotAPISendMessageChannel) (botsfw.OnMessageSentResponse, error) {
	return botsfw.OnMessageSentResponse{}, f.err
}

func (f fakeResponder) DeleteMessage(_ context.Context, _ string) error {
	return nil
}

var _ botsfw.WebhookResponder = fakeResponder{}

// fakeWhc is a minimal WebhookContext for testing SendRefreshOrNothingChanged.
type fakeWhc struct {
	responder botsfw.WebhookResponder
}

func (f fakeWhc) Context() context.Context                  { return context.Background() }
func (f fakeWhc) SetContext(_ context.Context)              {}
func (f fakeWhc) Request() *http.Request                    { return nil }
func (f fakeWhc) Environment() string                       { return "" }
func (f fakeWhc) BotPlatform() botsfw.BotPlatform           { return nil }
func (f fakeWhc) BotContext() botsfw.BotContext             { return botsfw.BotContext{} }
func (f fakeWhc) GetBotCode() string                        { return "" }
func (f fakeWhc) GetBotSettings() *botsfw.BotSettings       { return nil }
func (f fakeWhc) DB() dal.DB                                { return nil }
func (f fakeWhc) AppContext() botsfw.AppContext             { return nil }
func (f fakeWhc) ExecutionContext() botsfw.ExecutionContext { return nil }
func (f fakeWhc) Input() botinput.InputMessage              { return nil }
func (f fakeWhc) GetBotUserID() string                      { return "" }
func (f fakeWhc) MustBotChatID() string                     { return "" }
func (f fakeWhc) IsInGroup() (bool, error)                  { return false, nil }
func (f fakeWhc) ChatData() botsfwmodels.BotChatData        { return nil }
func (f fakeWhc) SaveBotChat() error                        { return nil }
func (f fakeWhc) GetBotUser() (botsdal.BotUser, error)      { return botsdal.BotUser{}, nil }
func (f fakeWhc) GetBotUserForUpdate(_ context.Context, _ dal.ReadwriteTransaction) (botsdal.BotUser, error) {
	return botsdal.BotUser{}, nil
}
func (f fakeWhc) AppUserID() string                                       { return "" }
func (f fakeWhc) SetUser(_ string, _ botsfwmodels.AppUserData)            {}
func (f fakeWhc) AppUserData() (botsfwmodels.AppUserData, error)          { return nil, nil }
func (f fakeWhc) RecordsFieldsSetter() botsfw.BotRecordsFieldsSetter      { return nil }
func (f fakeWhc) UpdateLastProcessed(_ botsfwmodels.BotChatData) error    { return nil }
func (f fakeWhc) IsNewerThen(_ botsfwmodels.BotChatData) bool             { return false }
func (f fakeWhc) Locale() i18n.Locale                                     { return i18n.Locale{} }
func (f fakeWhc) Translate(key string, _ ...any) string                   { return key }
func (f fakeWhc) TranslateWithMap(key string, _ map[string]string) string { return key }
func (f fakeWhc) TranslateNoWarning(key string, _ ...any) string          { return key }
func (f fakeWhc) SetLocale(_ string) error                                { return nil }
func (f fakeWhc) GetTranslator(_ string) i18n.SingleLocaleTranslator      { return f }
func (f fakeWhc) CommandText(_, _ string) string                          { return "" }
func (f fakeWhc) NewMessage(_ string) botmsg.MessageFromBot               { return botmsg.MessageFromBot{} }
func (f fakeWhc) NewMessageByCode(_ string, _ ...interface{}) botmsg.MessageFromBot {
	return botmsg.MessageFromBot{}
}
func (f fakeWhc) NewEditMessage(_ string, _ botmsg.Format) (botmsg.MessageFromBot, error) {
	return botmsg.MessageFromBot{}, nil
}
func (f fakeWhc) Responder() botsfw.WebhookResponder { return f.responder }
func (f fakeWhc) Analytics() botsfw.WebhookAnalytics { return nil }

var _ botsfw.WebhookContext = fakeWhc{}

func TestSendRefreshOrNothingChanged(t *testing.T) {
	t.Run("success_no_error", func(t *testing.T) {
		whc := fakeWhc{responder: fakeResponder{err: nil}}
		m := botmsg.MessageFromBot{}
		result, err := SendRefreshOrNothingChanged(whc, m)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result.BotMessage != nil {
			t.Fatal("expected BotMessage to be nil on success path")
		}
	})

	t.Run("error_400_api_response", func(t *testing.T) {
		apiErr := tgbotapi.APIResponse{Ok: false, ErrorCode: 400, Description: "Bad Request"}
		whc := fakeWhc{responder: fakeResponder{err: apiErr}}
		m := botmsg.MessageFromBot{}
		result, err := SendRefreshOrNothingChanged(whc, m)
		if err != nil {
			t.Fatalf("expected nil error for 400 APIResponse, got: %v", err)
		}
		if result.BotMessage == nil {
			t.Fatal("expected BotMessage to be set to CallbackAnswer for 400 error")
		}
	})

	t.Run("error_non_400", func(t *testing.T) {
		nonAPIErr := errors.New("some other error")
		whc := fakeWhc{responder: fakeResponder{err: nonAPIErr}}
		m := botmsg.MessageFromBot{}
		_, err := SendRefreshOrNothingChanged(whc, m)
		if err == nil {
			t.Fatal("expected error to be propagated for non-400 errors")
		}
	})
}
