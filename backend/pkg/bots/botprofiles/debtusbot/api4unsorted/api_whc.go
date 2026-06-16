package api4unsorted

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/strongo/logus"
)

// errNotSupportedByAPIWebhookContext is returned by ApiWebhookContext methods
// that have no meaningful implementation outside of a real bot webhook request.
var errNotSupportedByAPIWebhookContext = errors.New("not supported by ApiWebhookContext")

type ApiWebhookContext struct {
	appUser    *dbo4userus.UserDbo
	appUserID  string
	botChatID  int64
	chatEntity botsfwmodels.BotChatData
	*botsfw.WebhookContextBase
}

var _ botsfw.WebhookContext = (*ApiWebhookContext)(nil)

func (ApiWebhookContext) IsInGroup() (bool, error) {
	return false, fmt.Errorf("%w: IsInGroup()", errNotSupportedByAPIWebhookContext)
}

var BotHost botsfw.BotHost

func NewApiWebhookContext(r *http.Request, appUser *dbo4userus.UserDbo, userID string, botChatID int64, chatData botsfwmodels.BotChatData) ApiWebhookContext {
	var botSettings botsfw.BotSettings
	botContext := botsfw.NewBotContext(BotHost, &botSettings)
	args := botsfw.NewCreateWebhookContextArgs(
		r,
		nil, /*anybot.TheAppContext*/
		*botContext,
		nil,
		nil,
	)
	whcb, err := botsfw.NewWebhookContextBase(
		args,
		telegram.Platform, // webhookInput
		nil,               // records fields setter
		func() (bool, error) { return false, nil },
		nil, // GaMeasurement
	)
	if err != nil {
		logus.Errorf(r.Context(), "failed to create WebhookContextBase: %v", err)
	}
	whc := ApiWebhookContext{
		appUser:            appUser,
		appUserID:          userID,
		botChatID:          botChatID,
		chatEntity:         chatData,
		WebhookContextBase: whcb,
	}
	if err := whc.SetLocale(chatData.GetPreferredLanguage()); err != nil {
		logus.Errorf(r.Context(), "failed to set locale: %v", err)
	}
	return whc
}

func (whc ApiWebhookContext) AppUserData() (botsfwmodels.AppUserData, error) {
	return nil, fmt.Errorf("%w: AppUserData()", errNotSupportedByAPIWebhookContext)
}

func (whc ApiWebhookContext) BotChatIntID() int64 {
	return whc.botChatID
}

func (whc ApiWebhookContext) ChatEntity() botsfwmodels.BotChatData {
	return whc.chatEntity
}

func (whc ApiWebhookContext) GetAppUser() (botsfwmodels.AppUserData, error) {
	return nil, fmt.Errorf("%w: GetAppUser()", errNotSupportedByAPIWebhookContext)
}

func (whc ApiWebhookContext) Init(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (whc ApiWebhookContext) IsNewerThen(chatEntity botsfwmodels.BotChatData) bool {
	_ = chatEntity
	return true
}

func (whc ApiWebhookContext) MessageText() string {
	return ""
}

func (whc ApiWebhookContext) NewEditMessage(text string, format botmsg.Format) (m botmsg.MessageFromBot, err error) {
	return m, fmt.Errorf("%w: NewEditMessage(text=%s, format=%v)", errNotSupportedByAPIWebhookContext, text, format)
}

func (whc ApiWebhookContext) Responder() botsfw.WebhookResponder {
	return apiWebhookResponder{}
}

// apiWebhookResponder is a stub WebhookResponder that returns errors instead of
// sending messages: ApiWebhookContext serves HTTP API requests, not bot webhooks.
type apiWebhookResponder struct{}

var _ botsfw.WebhookResponder = apiWebhookResponder{}

func (apiWebhookResponder) SendMessage(_ context.Context, _ botmsg.MessageFromBot, _ botmsg.BotAPISendMessageChannel) (response botsfw.OnMessageSentResponse, err error) {
	return response, fmt.Errorf("%w: Responder().SendMessage()", errNotSupportedByAPIWebhookContext)
}

func (apiWebhookResponder) DeleteMessage(_ context.Context, messageID string) error {
	return fmt.Errorf("%w: Responder().DeleteMessage(messageID=%s)", errNotSupportedByAPIWebhookContext, messageID)
}

func (whc ApiWebhookContext) UpdateLastProcessed(chatEntity botsfwmodels.BotChatData) error {
	_ = chatEntity
	return fmt.Errorf("%w: UpdateLastProcessed()", errNotSupportedByAPIWebhookContext)
}
