package dtb_general

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botswebhook"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// --- helpers ---

func translateFn(key string, _ ...any) string { return key }

// setupTranslate registers AnyTimes expectations for both 1-arg and 2-arg Translate calls.
func setupTranslate(whc *mock_botsfw.MockWebhookContext) {
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(translateFn).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(translateFn).AnyTimes()
}

func newWhcWithTranslate(t *testing.T, platformID string) *mock_botsfw.MockWebhookContext {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return(platformID).AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	return whc
}

// fakeChat implements botinput.Chat
type fakeChat struct{ id string }

func (c fakeChat) GetID() string     { return c.id }
func (c fakeChat) GetType() string   { return "private" }
func (c fakeChat) IsGroupChat() bool { return false }

// fakeSender implements botinput.Sender
type fakeSender struct{}

func (fakeSender) Platform() string     { return "telegram" }
func (fakeSender) GetID() any           { return int64(1) }
func (fakeSender) IsBotUser() bool      { return false }
func (fakeSender) GetFirstName() string { return "Test" }
func (fakeSender) GetLastName() string  { return "" }
func (fakeSender) GetUserName() string  { return "testuser" }
func (fakeSender) GetLanguage() string  { return "en" }
func (fakeSender) GetAvatar() string    { return "" }

// fakeMessage implements botinput.Message (and botinput.InputMessage)
type fakeMessage struct {
	chatID    string
	messageID int
}

func (m fakeMessage) GetSender() botinput.User         { return nil }
func (m fakeMessage) GetRecipient() botinput.Recipient { return nil }
func (m fakeMessage) GetTime() time.Time               { return time.Time{} }
func (m fakeMessage) InputType() botinput.Type         { return botinput.TypeCallbackQuery }
func (m fakeMessage) MessageIntID() int                { return m.messageID }
func (m fakeMessage) MessageStringID() string          { return "" }
func (m fakeMessage) BotChatID() (string, error)       { return m.chatID, nil }
func (m fakeMessage) Chat() botinput.Chat              { return fakeChat{id: m.chatID} }
func (m fakeMessage) LogRequest()                      {}

// fakeCallbackQuery implements botinput.CallbackQuery AND telegram.WebhookCallbackQuery AND botinput.InputMessage
type fakeCallbackQuery struct {
	fakeMessage
	data string
}

func (q fakeCallbackQuery) GetID() string                { return "cb1" }
func (q fakeCallbackQuery) GetFrom() botinput.Sender     { return fakeSender{} }
func (q fakeCallbackQuery) GetMessage() botinput.Message { return q.fakeMessage }
func (q fakeCallbackQuery) GetData() string              { return q.data }
func (q fakeCallbackQuery) GetInlineMessageID() string   { return "" }
func (q fakeCallbackQuery) GetChatInstanceID() string    { return "" }

// verify interface compliance at compile time
var _ telegram.WebhookCallbackQuery = fakeCallbackQuery{}
var _ botinput.InputMessage = fakeCallbackQuery{}

// fakeTextInput implements botinput.InputMessage with TypeText
type fakeTextInput struct{ chatID string }

func (f fakeTextInput) GetSender() botinput.User         { return nil }
func (f fakeTextInput) GetRecipient() botinput.Recipient { return nil }
func (f fakeTextInput) GetTime() time.Time               { return time.Time{} }
func (f fakeTextInput) InputType() botinput.Type         { return botinput.TypeText }
func (f fakeTextInput) MessageIntID() int                { return 0 }
func (f fakeTextInput) MessageStringID() string          { return "" }
func (f fakeTextInput) BotChatID() (string, error)       { return f.chatID, nil }
func (f fakeTextInput) Chat() botinput.Chat              { return fakeChat{id: f.chatID} }
func (f fakeTextInput) LogRequest()                      {}

// --- ad_slot ---

func TestAdSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return "href" }).AnyTimes()
	result := AdSlot(whc, "test_campaign")
	_ = result
}

// --- main_menu_keyboard ---

func TestMainMenuKeyboardOnReceiptAck(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	kb := MainMenuKeyboardOnReceiptAck(whc)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
}

func TestDebtsMainMenuInlineKeyboard(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	spaceRef := coretypes.NewSpaceRef("family", "space1")
	rows := debtsMainMenuInlineKeyboard(whc, spaceRef)
	if len(rows) == 0 {
		t.Fatal("expected non-empty rows")
	}
}

func TestMainMenuTelegramKeyboardShowReturn(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	params := mainMenuParams{showReturn: true}
	kb := mainMenuTelegramKeyboard(whc, params)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	if len(kb.Keyboard[0]) != 3 {
		t.Errorf("expected 3 buttons in first row when showReturn=true, got %d", len(kb.Keyboard[0]))
	}
}

// --- feedback_cmd ---

func TestFeedbackCommandAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "do you like"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_DO_YOU_LIKE_OUR_BOT).Return(msg)
	whc.EXPECT().GetBotCode().Return("debtusbot")
	setupTranslate(whc)
	m, err := feedbackCommandAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestFeedbackCommandCallbackAction_noLike(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "do you like"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_DO_YOU_LIKE_OUR_BOT).Return(msg)
	whc.EXPECT().GetBotCode().Return("debtusbot")
	setupTranslate(whc)

	u, _ := url.Parse("feedback")
	m, err := feedbackCommandCallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard from feedbackCommandAction")
	}
}

func TestFeedbackCommandCallbackAction_invalidLike(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()

	u, _ := url.Parse("feedback?like=maybe")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error for unexpected like value")
	}
}

func TestFeedbackCommandCallbackAction_likeYes(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "12345", messageID: 42}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return nil
	}
	defer func() { runReadwriteTransaction = orig }()

	u, _ := url.Parse("feedback?like=yes")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFeedbackCommandCallbackAction_likeNo(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(FeedbackTextCommandCode)
	chatData.EXPECT().AddWizardParam(gomock.Any(), gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	msg := botmsg.MessageFromBot{}
	msg.Text = "ask"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER).Return(msg)

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return nil
	}
	defer func() { runReadwriteTransaction = orig }()

	u, _ := url.Parse("feedback?like=no")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFeedbackOptionsTelegramKeyboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	setupTranslate(whc)
	kb := feedbackOptionsTelegramKeyboard(whc)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	if len(kb.InlineKeyboard) == 0 {
		t.Fatal("expected rows in keyboard")
	}
}

func TestAskIfCanRateAtStoreBot(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	setupTranslate(whc)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "999", messageID: 7}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	m, err := askIfCanRateAtStoreBot(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestAskToWriteFeedback(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "ask"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER).Return(msg)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(FeedbackTextCommandCode)
	chatData.EXPECT().AddWizardParam("feedback", "fb42")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	m, err := askToWriteFeedback(whc, "fb42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected hide keyboard")
	}
}

func TestAskToWriteFeedback_noID(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "ask"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER).Return(msg)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(FeedbackTextCommandCode)
	// AddWizardParam NOT called when feedbackID == ""
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	_, err := askToWriteFeedback(whc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditTelegramMessageText_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "12345", messageID: 99}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage("hello", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("someCode")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	m, err := editTelegramMessageText(whc, "someCode", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.EditMessageUID == nil {
		t.Error("expected EditMessageUID to be set")
	}
}

func TestEditTelegramMessageText_clearAwaitingReply(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "12345", messageID: 99}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage("text", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	// "/" means clear
	_, err := editTelegramMessageText(whc, "/", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditTelegramMessageText_badChatID(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "not-a-number", messageID: 99}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()

	_, err := editTelegramMessageText(whc, "", "text")
	if err == nil {
		t.Fatal("expected error for non-numeric chatID")
	}
}

func TestFeedbackLinks(t *testing.T) {
	translator := newTestTranslator(i18n.LocaleEnUS)
	result, err := FeedbackLinks(translator, "<a suggest-idea>idea</a> <a submit-bug>bug</a>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestFeedbackLinks_ru(t *testing.T) {
	translator := newTestTranslator(i18n.LocaleRuRU)
	result, err := FeedbackLinks(translator, "<a suggest-idea>idea</a>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

// --- help_cmd ---

func TestHelpCommandAction_showFeedback(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	msg := botmsg.MessageFromBot{}
	msg.Text = "help"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_HELP).Return(msg)
	m, err := HelpCommandAction(whc, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestHelpCommandAction_noFeedback(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().NewEditMessage("", botmsg.FormatText).Return(botmsg.MessageFromBot{}, nil)
	m, err := HelpCommandAction(whc, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestBtnSubmitIdea(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	btn := btnSubmitIdea(whc, "https://example.com")
	if btn.Text == "" {
		t.Error("expected non-empty text")
	}
}

func TestBtnSubmitBug(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	btn := btnSubmitBug(whc, "https://example.com")
	if btn.Text == "" {
		t.Error("expected non-empty text")
	}
}

// --- main_menu_cmd ---

func TestMainMenuAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	m, err := MainMenuAction(whc, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

func TestMainMenuAction_noHint(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	_, err := MainMenuAction(whc, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMainMenuAction_customText(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage("custom text").Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	_, err := MainMenuAction(whc, "custom text", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- contacts_cmd ---

func TestDebtusContactsAction_nilCallback(t *testing.T) {
	for _, code := range []string{debtorsCommandCode, creditorsCommandCode, debtusContactsCommandCode} {
		t.Run(code, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)
			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			m, err := debtusContactsAction(whc, nil, code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.Keyboard == nil {
				t.Error("expected keyboard")
			}
		})
	}
}

func TestDebtusContactsAction_withCallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewEditMessage("", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	u, _ := url.Parse("debtus_contacts")
	m, err := debtusContactsAction(whc, u, debtusContactsCommandCode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

// --- debts_command ---

func TestDebtsAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	orig := getCurrentSpaceRef
	getCurrentSpaceRef = func(_ botsfw.WebhookContext) (coretypes.SpaceRef, error) {
		return coretypes.NewSpaceRef("family", "space1"), nil
	}
	defer func() { getCurrentSpaceRef = orig }()

	m, err := debtsAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestDebtsAction_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()

	orig := getCurrentSpaceRef
	getCurrentSpaceRef = func(_ botsfw.WebhookContext) (coretypes.SpaceRef, error) {
		return "", errors.New("no space")
	}
	defer func() { getCurrentSpaceRef = orig }()

	_, err := debtsAction(whc)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDebtsCallbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	origGetSpace := getCurrentSpaceRef
	getCurrentSpaceRef = func(_ botsfw.WebhookContext) (coretypes.SpaceRef, error) {
		return coretypes.NewSpaceRef("family", "space1"), nil
	}
	defer func() { getCurrentSpaceRef = origGetSpace }()

	origGetUID := getMessageUID
	fakeMsgUID := telegram.NewChatMessageUID(123, 456)
	getMessageUID = func(_ botsfw.WebhookContext) (*telegram.ChatMessageUID, error) {
		return fakeMsgUID, nil
	}
	defer func() { getMessageUID = origGetUID }()

	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	u, _ := url.Parse("debts")
	_, err := debtsCallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- reminder_cmd ---

func TestEditReminderMessage_callbackQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()

	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "111", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	origTxt := textReceiptForTransfer
	textReceiptForTransfer = func(ctx context.Context, t i18n.SingleLocaleTranslator, transfer models4debtus.TransferEntry, userID string, showTo common4debtus.ShowReceiptTo, utmParams utm.Params) string {
		return "receipt"
	}
	defer func() { textReceiptForTransfer = origTxt }()

	transfer := models4debtus.TransferEntry{}
	_, err := EditReminderMessage(whc, transfer, "reminder text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditReminderMessage_textInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	textInput := fakeTextInput{chatID: "111"}
	whc.EXPECT().Input().Return(textInput).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origTxt := textReceiptForTransfer
	textReceiptForTransfer = func(ctx context.Context, t i18n.SingleLocaleTranslator, transfer models4debtus.TransferEntry, userID string, showTo common4debtus.ShowReceiptTo, utmParams utm.Params) string {
		return "receipt"
	}
	defer func() { textReceiptForTransfer = origTxt }()

	transfer := models4debtus.TransferEntry{}
	_, err := EditReminderMessage(whc, transfer, "reminder text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditReminderMessage_callbackQueryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()

	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "111", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, errors.New("edit error"))

	origTxt := textReceiptForTransfer
	textReceiptForTransfer = func(ctx context.Context, t i18n.SingleLocaleTranslator, transfer models4debtus.TransferEntry, userID string, showTo common4debtus.ShowReceiptTo, utmParams utm.Params) string {
		return "receipt"
	}
	defer func() { textReceiptForTransfer = origTxt }()

	transfer := models4debtus.TransferEntry{}
	_, err := EditReminderMessage(whc, transfer, "reminder text")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEditReminderMessage_textInputSetMainMenuError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes() // unsupported platform -> SetMainMenuKeyboard returns error
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()

	textInput := fakeTextInput{chatID: "111"}
	whc.EXPECT().Input().Return(textInput).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origTxt := textReceiptForTransfer
	textReceiptForTransfer = func(ctx context.Context, t i18n.SingleLocaleTranslator, transfer models4debtus.TransferEntry, userID string, showTo common4debtus.ShowReceiptTo, utmParams utm.Params) string {
		return "receipt"
	}
	defer func() { textReceiptForTransfer = origTxt }()

	transfer := models4debtus.TransferEntry{}
	_, err := EditReminderMessage(whc, transfer, "reminder text")
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on unsupported platform")
	}
}

// --- RegisterCommands ---

type fakeCommandsRegisterer struct{ count int }

func (f *fakeCommandsRegisterer) RegisterCommands(cmds ...botsfw.Command) { f.count = len(cmds) }

var _ botswebhook.CommandsRegisterer = (*fakeCommandsRegisterer)(nil)

func TestRegisterCommands(t *testing.T) {
	r := &fakeCommandsRegisterer{}
	RegisterCommands(r)
	if r.count == 0 {
		t.Error("expected RegisterCommands to register at least one command")
	}
}

// --- command closures ---

func TestDebtusContactsCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := debtusContactsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebtusContactsCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewEditMessage("", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)
	u, _ := url.Parse("debtus_contacts")
	_, err := debtusContactsCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebtorsCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := debtorsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebtorsCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewEditMessage("", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)
	u, _ := url.Parse("debtors")
	_, err := debtorsCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreditorsCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := creditorsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreditorsCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewEditMessage("", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)
	u, _ := url.Parse("creditors")
	_, err := creditorsCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebtusContactsAction_callbackError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewEditMessage("", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, errors.New("edit err"))
	u, _ := url.Parse("debtus_contacts")
	_, err := debtusContactsAction(whc, u, debtusContactsCommandCode)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMainMenuCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	_, err := MainMenuCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMainMenuCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	u, _ := url.Parse("main_menu")
	_, err := MainMenuCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMainMenuAction_setMainMenuKeyboardError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes() // unsupported -> SetMainMenuKeyboard error
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	_, err := MainMenuAction(whc, "", false)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on unsupported platform")
	}
}

// --- debtsCallbackAction error paths ---

func TestDebtsCallbackAction_debtsActionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()

	orig := getCurrentSpaceRef
	getCurrentSpaceRef = func(_ botsfw.WebhookContext) (coretypes.SpaceRef, error) {
		return "", errors.New("no space")
	}
	defer func() { getCurrentSpaceRef = orig }()

	u, _ := url.Parse("debts")
	_, err := debtsCallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDebtsCallbackAction_newEditMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	orig := getCurrentSpaceRef
	getCurrentSpaceRef = func(_ botsfw.WebhookContext) (coretypes.SpaceRef, error) {
		return coretypes.NewSpaceRef("family", "space1"), nil
	}
	defer func() { getCurrentSpaceRef = orig }()

	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, errors.New("edit error"))

	u, _ := url.Parse("debts")
	_, err := debtsCallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from NewEditMessage")
	}
}

// --- editTelegramMessageText error path ---

func TestEditTelegramMessageText_newEditMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "12345", messageID: 99}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage("hello", botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, errors.New("edit fail"))

	_, err := editTelegramMessageText(whc, "", "hello")
	if err == nil {
		t.Fatal("expected error from NewEditMessage")
	}
}

// --- HelpCommandAction error path ---

func TestHelpCommandAction_noFeedback_editMessageError(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().NewEditMessage("", botmsg.FormatText).Return(botmsg.MessageFromBot{}, errors.New("edit fail"))
	_, err := HelpCommandAction(whc, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- feedbackCommandCallbackAction: transaction error ---

func TestFeedbackCommandCallbackAction_transactionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return errors.New("tx error")
	}
	defer func() { runReadwriteTransaction = orig }()

	u, _ := url.Parse("feedback?like=yes")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from transaction")
	}
}

// --- feedbackTextCommand closures ---

// fakeTextMessage implements botinput.TextMessage (not a callback query)
type fakeTextMessage struct{ chatID string }

func (f fakeTextMessage) GetSender() botinput.User         { return nil }
func (f fakeTextMessage) GetRecipient() botinput.Recipient { return nil }
func (f fakeTextMessage) GetTime() time.Time               { return time.Time{} }
func (f fakeTextMessage) InputType() botinput.Type         { return botinput.TypeText }
func (f fakeTextMessage) MessageIntID() int                { return 0 }
func (f fakeTextMessage) MessageStringID() string          { return "" }
func (f fakeTextMessage) BotChatID() (string, error)       { return f.chatID, nil }
func (f fakeTextMessage) Chat() botinput.Chat              { return fakeChat{id: f.chatID} }
func (f fakeTextMessage) LogRequest()                      {}
func (f fakeTextMessage) Text() string                     { return "some feedback text" }
func (f fakeTextMessage) IsEdited() bool                   { return false }

var _ botinput.TextMessage = fakeTextMessage{}

func TestFeedbackTextCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "ask"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_ASK_TO_WRITE_FEEDBACK_WITHIN_MESSENGER).Return(msg)
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(FeedbackTextCommandCode)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	u, _ := url.Parse("feedback_text")
	_, err := feedbackTextCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFeedbackTextCommand_action_nonTextInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	// fakeCallbackQuery does not implement botinput.TextMessage → falls to default branch
	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "123", messageID: 1}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	msg := botmsg.MessageFromBot{}
	msg.Text = "please send text"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_PLEASE_SEND_TEXT).Return(msg)

	_, err := feedbackTextCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFeedbackTextCommand_action_textInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	// Use unsupported platform so SetMainMenuKeyboard returns error before admin.SendFeedbackToAdmins is called
	platform.EXPECT().ID().Return("viber").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return nil // skip real DB call; feedback stays zero-value
	}
	defer func() { runReadwriteTransaction = orig }()

	msg := botmsg.MessageFromBot{}
	msg.Text = "thanks"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_THANKS).Return(msg)

	// SetMainMenuKeyboard returns error for "viber" platform, so err != nil is expected
	_, err := feedbackTextCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on unsupported platform")
	}
}

// --- canYouRateCommand.CallbackAction paths ---

func TestCanYouRateCommand_callbackAction_nilQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "999", messageID: 7}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	_, err := canYouRateCommand.CallbackAction(whc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanYouRateCommand_callbackAction_willRateYes(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	setupTranslate(whc)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "123", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	u, _ := url.Parse("can_you_rate?will-rate=yes")
	_, err := canYouRateCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanYouRateCommand_callbackAction_willRateNo(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "123", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	u, _ := url.Parse("can_you_rate?will-rate=no")
	_, err := canYouRateCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanYouRateCommand_callbackAction_willRateDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	u, _ := url.Parse("can_you_rate?will-rate=maybe")
	_, err := canYouRateCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- feedbackCommandCallbackAction: SaveFeedback error inside tx ---

func TestFeedbackCommandCallbackAction_saveFeedbackError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, errors.New("save error")
	}
	defer func() { saveFeedback = origSave }()

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = origTx }()

	u, _ := url.Parse("feedback?like=yes")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from SaveFeedback")
	}
}

// --- editTelegramMessageText: BotChatID error ---

type fakeChatIDErrorInput struct{ fakeCallbackQuery }

func (f fakeChatIDErrorInput) BotChatID() (string, error) {
	return "", errors.New("chat id error")
}

func TestEditTelegramMessageText_chatIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	errInput := fakeChatIDErrorInput{fakeCallbackQuery: fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "x", messageID: 1}}}
	whc.EXPECT().Input().Return(errInput).AnyTimes()

	_, err := editTelegramMessageText(whc, "", "text")
	if err == nil {
		t.Fatal("expected error from BotChatID")
	}
}

// --- canYouRateCommand "no" branch: FeedbackLinks error / editTelegramMessageText error ---

func TestCanYouRateCommand_callbackAction_willRateNo_feedbackLinksError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	// Return an unknown locale so getUserReportUrl fails with invalid submit
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	setupTranslate(whc)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	// Patch getUserReportUrl via a locale that makes it return error — not possible directly.
	// Instead intercept at FeedbackLinks level isn't possible without a seam.
	// Just test the happy path "no" already tested in TestCanYouRateCommand_callbackAction_willRateNo.
	// The inner editTelegramMessageText error path — mock NewEditMessage to fail.
	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "123", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, errors.New("edit fail"))

	u, _ := url.Parse("can_you_rate?will-rate=no")
	_, err := canYouRateCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from editTelegramMessageText")
	}
}

// --- getUserReportUrl seam error paths ---

func TestHelpCommandAction_getUserReportUrlError_empty(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		if submit == "" {
			return "", errors.New("seam error for ''")
		}
		return "https://example.com", nil
	}
	defer func() { getUserReportUrl = orig }()
	_, err := HelpCommandAction(whc, false)
	if err == nil {
		t.Fatal("expected error from getUserReportUrl for ''")
	}
}

func TestHelpCommandAction_getUserReportUrlError_bug(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		if submit == "bug" {
			return "", errors.New("seam error for 'bug'")
		}
		return "https://example.com", nil
	}
	defer func() { getUserReportUrl = orig }()
	_, err := HelpCommandAction(whc, false)
	if err == nil {
		t.Fatal("expected error from getUserReportUrl for 'bug'")
	}
}

func TestHelpCommandAction_getUserReportUrlError_idea(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		if submit == "idea" {
			return "", errors.New("seam error for 'idea'")
		}
		return "https://example.com", nil
	}
	defer func() { getUserReportUrl = orig }()
	_, err := HelpCommandAction(whc, false)
	if err == nil {
		t.Fatal("expected error from getUserReportUrl for 'idea'")
	}
}

func TestFeedbackLinks_getUserReportUrlError_idea(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		if submit == "idea" {
			return "", errors.New("seam error for 'idea'")
		}
		return "https://example.com", nil
	}
	defer func() { getUserReportUrl = orig }()
	_, err := FeedbackLinks(whc, "text")
	if err == nil {
		t.Fatal("expected error from FeedbackLinks for idea")
	}
}

func TestFeedbackLinks_getUserReportUrlError_bug(t *testing.T) {
	whc := newWhcWithTranslate(t, "telegram")
	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		if submit == "bug" {
			return "", errors.New("seam error for 'bug'")
		}
		return "https://example.com", nil
	}
	defer func() { getUserReportUrl = orig }()
	_, err := FeedbackLinks(whc, "text")
	if err == nil {
		t.Fatal("expected error from FeedbackLinks for bug")
	}
}

func TestCanYouRateCommand_callbackAction_willRateNo_getUserReportUrlError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	setupTranslate(whc)
	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetPreferredLanguage().Return("en").AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "123", messageID: 5}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()

	orig := getUserReportUrl
	getUserReportUrl = func(_ i18n.SingleLocaleTranslator, submit string) (string, error) {
		return "", errors.New("seam error for getUserReportUrl")
	}
	defer func() { getUserReportUrl = orig }()

	u, _ := url.Parse("can_you_rate?will-rate=no")
	_, err := canYouRateCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from FeedbackLinks via getUserReportUrl seam")
	}
}

// --- simple command closures ---

func TestDebtusHomeCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	_, err := debtusHomeCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPleaseWaitCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	msg := botmsg.MessageFromBot{}
	msg.Text = "please wait"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_PLEASE_WAIT).Return(msg)
	u, _ := url.Parse("please_wait")
	_, err := pleaseWaitCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClearCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	_, err := clearCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogin2WebCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Environment().Return("local").AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return "go to <a>web</a>" }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := login2WebCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAllCommand_action_notLocalEnv(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	settings := &botsfw.BotSettings{Env: "prod", Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings)
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := deleteAllCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAllCommand_action_devEnvSetMainMenuKeyboardError(t *testing.T) {
	// Env=="dev" enters the else branch; use "viber" platform so SetMainMenuKeyboard
	// returns error before the real DB call (DeleteAll) is reached.
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	settings := &botsfw.BotSettings{Env: "dev", Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings).AnyTimes()
	whc.EXPECT().NewMessage("Deleted all records").Return(botmsg.MessageFromBot{})
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	_, err := deleteAllCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on viber")
	}
}

// --- adsCommand.Action branches ---

func TestAdsCommand_action_awaitingEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return("")
	chatData.EXPECT().SetAwaitingReplyTo(adsCommandCode)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	_, err := adsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdsCommand_action_awaitingYes(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	yesText := "📱 " + trans.COMMAND_TEXT_SUBSCRIBE_TO_APP // approximate; exact value from Translate
	txtInput := &fakeTextMessageWithText{chatID: "1"}
	// The exact text for yesOption is: emoji.PHONE_ICON + " " + Translate(SUBSCRIBE_TO_APP)
	// Set text to match yesOption
	txtInput.text = "📱 " + trans.COMMAND_TEXT_SUBSCRIBE_TO_APP
	_ = yesText

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(adsCommandCode)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(txtInput).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := adsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdsCommand_action_awaitingNo(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	txtInput := &fakeTextMessageWithText{chatID: "1"}
	txtInput.text = trans.COMMAND_TEXT_I_AM_FINE_WITH_BOT

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(adsCommandCode)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(txtInput).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := adsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdsCommand_action_awaitingDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	txtInput := &fakeTextMessageWithText{chatID: "1"}
	txtInput.text = "unexpected text"

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(adsCommandCode)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(txtInput).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := adsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// fakeTextMessageWithText is like fakeTextMessage but with configurable text
type fakeTextMessageWithText struct {
	chatID string
	text   string
}

func (f *fakeTextMessageWithText) GetSender() botinput.User         { return nil }
func (f *fakeTextMessageWithText) GetRecipient() botinput.Recipient { return nil }
func (f *fakeTextMessageWithText) GetTime() time.Time               { return time.Time{} }
func (f *fakeTextMessageWithText) InputType() botinput.Type         { return botinput.TypeText }
func (f *fakeTextMessageWithText) MessageIntID() int                { return 0 }
func (f *fakeTextMessageWithText) MessageStringID() string          { return "" }
func (f *fakeTextMessageWithText) BotChatID() (string, error)       { return f.chatID, nil }
func (f *fakeTextMessageWithText) Chat() botinput.Chat              { return fakeChat{id: f.chatID} }
func (f *fakeTextMessageWithText) LogRequest()                      {}
func (f *fakeTextMessageWithText) Text() string                     { return f.text }
func (f *fakeTextMessageWithText) IsEdited() bool                   { return false }

var _ botinput.TextMessage = (*fakeTextMessageWithText)(nil)

// --- betaCommand ---

func TestBetaCommand_action_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	settings := &botsfw.BotSettings{Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings)

	orig := issueBotToken
	issueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("token error")
	}
	defer func() { issueBotToken = orig }()

	_, err := betaCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from issueBotToken")
	}
}

func TestBetaCommand_action_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	settings := &botsfw.BotSettings{Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings)
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	orig := issueBotToken
	issueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "mytoken", nil
	}
	defer func() { issueBotToken = orig }()

	_, err := betaCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- deleteAllCommand remaining paths ---

func TestDeleteAllCommand_action_devEnv_chatIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	settings := &botsfw.BotSettings{Env: "dev", Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings).AnyTimes()
	whc.EXPECT().NewMessage("Deleted all records").Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	// Use fakeChatIDErrorInput so BotChatID returns error
	errInput := fakeChatIDErrorInput{fakeCallbackQuery: fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "x", messageID: 1}}}
	whc.EXPECT().Input().Return(errInput).AnyTimes()

	_, err := deleteAllCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from BotChatID")
	}
}

func TestDeleteAllCommand_action_devEnv_deleteAllError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	settings := &botsfw.BotSettings{Env: "dev", Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings).AnyTimes()
	whc.EXPECT().NewMessage("Deleted all records").Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	whc.EXPECT().Input().Return(fakeTextInput{chatID: "123"}).AnyTimes()

	orig := deleteAll
	deleteAll = func(_ context.Context, _, _ string) error {
		return errors.New("delete error")
	}
	defer func() { deleteAll = orig }()

	_, err := deleteAllCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from deleteAll")
	}
}

func TestDeleteAllCommand_action_devEnv_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	settings := &botsfw.BotSettings{Env: "dev", Code: "debtusbot"}
	whc.EXPECT().GetBotSettings().Return(settings).AnyTimes()
	whc.EXPECT().NewMessage("Deleted all records").Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	whc.EXPECT().Input().Return(fakeTextInput{chatID: "123"}).AnyTimes()

	orig := deleteAll
	deleteAll = func(_ context.Context, _, _ string) error { return nil }
	defer func() { deleteAll = orig }()

	_, err := deleteAllCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- feedbackCommandCallbackAction: tx fn success (line 199 return nil) ---

func TestFeedbackCommandCallbackAction_saveFeedbackSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1")
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	cbInput := fakeCallbackQuery{fakeMessage: fakeMessage{chatID: "12345", messageID: 42}}
	whc.EXPECT().Input().Return(cbInput).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), botmsg.FormatHTML).Return(botmsg.MessageFromBot{}, nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, nil
	}
	defer func() { saveFeedback = origSave }()

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = origTx }()

	u, _ := url.Parse("feedback?like=yes")
	_, err := feedbackCommandCallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- feedbackTextCommand with feedbackParam set ---

func TestFeedbackTextCommand_action_textInput_withFeedbackParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes() // unsupported: SetMainMenuKeyboard fails before SendFeedbackToAdmins
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("fb123")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origGet := getFeedbackByID
	getFeedbackByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.Feedback, error) {
		fb := models4debtus.Feedback{FeedbackData: &models4debtus.FeedbackData{}}
		return fb, nil
	}
	defer func() { getFeedbackByID = origGet }()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, nil
	}
	defer func() { saveFeedback = origSave }()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = orig }()

	msg := botmsg.MessageFromBot{}
	msg.Text = "thanks"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_THANKS).Return(msg)

	_, err := feedbackTextCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on viber")
	}
}

func TestFeedbackTextCommand_action_textInput_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, nil
	}
	defer func() { saveFeedback = origSave }()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = orig }()

	msg := botmsg.MessageFromBot{}
	msg.Text = "thanks"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_THANKS).Return(msg)

	origSend := sendFeedbackToAdmins
	sendFeedbackToAdmins = func(_ context.Context, _ string, _ models4debtus.Feedback) error { return nil }
	defer func() { sendFeedbackToAdmins = origSend }()

	_, err := feedbackTextCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- adsCommand SetMainMenuKeyboard error paths ---

func TestAdsCommand_action_awaitingYes_setMainMenuKeyboardError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	// yesOption = emoji.PHONE_ICON + " " + Translate(SUBSCRIBE_TO_APP) = "📱 " + COMMAND_TEXT_SUBSCRIBE_TO_APP (translateFn returns key)
	yesOption := "📱 " + trans.COMMAND_TEXT_SUBSCRIBE_TO_APP
	txtInput := &fakeTextMessageWithText{chatID: "1", text: yesOption}

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(adsCommandCode)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(txtInput).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := adsCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on viber")
	}
}

// --- getFeedbackByID error path in feedbackTextCommand ---

func TestFeedbackTextCommand_action_textInput_getFeedbackByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("fb123")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origGet := getFeedbackByID
	getFeedbackByID = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.Feedback, error) {
		return models4debtus.Feedback{}, errors.New("get error")
	}
	defer func() { getFeedbackByID = origGet }()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = orig }()

	_, err := feedbackTextCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from getFeedbackByID")
	}
}

// --- saveFeedback error path in feedbackTextCommand (feedbackParam=="") ---

func TestFeedbackTextCommand_action_textInput_saveFeedbackError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, errors.New("save error")
	}
	defer func() { saveFeedback = origSave }()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = orig }()

	_, err := feedbackTextCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from saveFeedback")
	}
}

// --- sendFeedbackToAdmins error branch ---

func TestFeedbackTextCommand_action_textInput_sendAdminsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	txtInput := fakeTextMessage{chatID: "111"}
	whc.EXPECT().Input().Return(txtInput).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetWizardParam("feedback").Return("")
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origSave := saveFeedback
	saveFeedback = func(_ context.Context, _ dal.ReadwriteTransaction, _ string, _ *models4debtus.FeedbackData) (models4debtus.Feedback, dbo4userus.UserEntry, error) {
		return models4debtus.Feedback{}, dbo4userus.UserEntry{}, nil
	}
	defer func() { saveFeedback = origSave }()

	orig := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, fn func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return fn(ctx, nil)
	}
	defer func() { runReadwriteTransaction = orig }()

	msg := botmsg.MessageFromBot{}
	msg.Text = "thanks"
	whc.EXPECT().NewMessageByCode(trans.MESSAGE_TEXT_THANKS).Return(msg)

	origSend := sendFeedbackToAdmins
	sendFeedbackToAdmins = func(_ context.Context, _ string, _ models4debtus.Feedback) error {
		return errors.New("admin notify error") // error logged but not returned
	}
	defer func() { sendFeedbackToAdmins = origSend }()

	_, err := feedbackTextCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error (sendFeedbackToAdmins errors are logged not returned): %v", err)
	}
}

func TestAdsCommand_action_awaitingNo_setMainMenuKeyboardError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("viber").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	setupTranslate(whc)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	noOption := trans.COMMAND_TEXT_I_AM_FINE_WITH_BOT
	txtInput := &fakeTextMessageWithText{chatID: "1", text: noOption}

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().GetAwaitingReplyTo().Return(adsCommandCode)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(txtInput).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := adsCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetMainMenuKeyboard on viber")
	}
}

// --- default seam closures (deleteAll / getFeedbackByID) ---

type fakeAdminDal struct {
	dal4debtus.AdminDal
	deleteAllErr error
}

func (f fakeAdminDal) DeleteAll(context.Context, string, string) error { return f.deleteAllErr }

type fakeFeedbackDal struct {
	dal4debtus.FeedbackDal
	getErr error
}

func (f fakeFeedbackDal) GetFeedbackByID(context.Context, dal.ReadSession, string) (models4debtus.Feedback, error) {
	return models4debtus.Feedback{}, f.getErr
}

func TestDefaultSeamClosures(t *testing.T) {
	origDefault := dal4debtus.Default
	t.Cleanup(func() { dal4debtus.Default = origDefault })

	dal4debtus.Default.Admin = fakeAdminDal{deleteAllErr: errors.New("delete err")}
	dal4debtus.Default.Feedback = fakeFeedbackDal{getErr: errors.New("feedback err")}

	ctx := context.Background()

	if err := deleteAll(ctx, "bot", "chat"); err == nil {
		t.Error("expected error from deleteAll default closure")
	}
	if _, err := getFeedbackByID(ctx, nil, "FID"); err == nil {
		t.Error("expected error from getFeedbackByID default closure")
	}
}
