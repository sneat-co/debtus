package dtb_settings

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"go.uber.org/mock/gomock"
)

// --- helpers ---

// fakeTextInput implements botinput.InputMessage and botinput.TextMessage
type fakeTextInput struct {
	text string
}

func (f fakeTextInput) GetSender() botinput.User         { return nil }
func (f fakeTextInput) GetRecipient() botinput.Recipient { return nil }
func (f fakeTextInput) GetTime() time.Time               { return time.Time{} }
func (f fakeTextInput) InputType() botinput.Type         { return botinput.TypeText }
func (f fakeTextInput) MessageIntID() int                { return 0 }
func (f fakeTextInput) MessageStringID() string          { return "" }
func (f fakeTextInput) BotChatID() (string, error)       { return "chat1", nil }
func (f fakeTextInput) Chat() botinput.Chat              { return nil }
func (f fakeTextInput) LogRequest()                      {}
func (f fakeTextInput) Text() string                     { return f.text }
func (f fakeTextInput) IsEdited() bool                   { return false }

var _ botinput.TextMessage = fakeTextInput{}
var _ botinput.InputMessage = fakeTextInput{}

// --- TextCommand ---

func TestTextCommand(t *testing.T) {
	cmd := TextCommand("on", []string{"msg1", "msg2"}, "icon", "replyIcon", true)
	if cmd.Code == "" {
		t.Error("expected non-empty code")
	}
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard")
	}
}

func TestTextCommand_noReplyIcon(t *testing.T) {
	cmd := TextCommand("on", []string{"msg1"}, "icon", "", true)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})
	_, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- NewMistypedCommand ---

func TestNewMistypedCommand_withMessage(t *testing.T) {
	cmd := NewMistypedCommand("some extra message")
	if cmd.Code == "" {
		t.Error("expected non-empty code")
	}
}

func TestNewMistypedCommand_noMessage(t *testing.T) {
	cmd := NewMistypedCommand("")
	if cmd.Code == "" {
		t.Error("expected non-empty code")
	}
}

// --- CommandText ---

func TestContactsChannelCommandText(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("result")
	cmd := ContactsChannelCommand{icon: "🔑", title: "title"}
	result := cmd.CommandText(whc)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// --- OnboardingOnUserContactReceivedCommand ---

func TestOnboardingOnUserContactReceivedCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := OnboardingOnUserContactReceivedCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- settingsCommand init closures ---

func TestSettingsCommand_action(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	orig := settingsMainAction
	settingsMainAction = func(_ botsfw.WebhookContext, _ string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, nil
	}
	defer func() { settingsMainAction = orig }()

	_, err := settingsCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSettingsCommand_callbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	orig := settingsMainAction
	settingsMainAction = func(_ botsfw.WebhookContext, _ string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, nil
	}
	defer func() { settingsMainAction = orig }()

	u, _ := url.Parse("settings")
	_, err := settingsCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- StartInBotAction ---

func TestStartInBotAction_noParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := StartInBotAction(whc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartInBotAction_unmatched(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := StartInBotAction(whc, []string{"nomatch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartInBotAction_tooManyParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	_, err := StartInBotAction(whc, []string{"a", "b"})
	if err != cmds4anybot.ErrUnknownStartParam {
		t.Fatalf("expected ErrUnknownStartParam, got: %v", err)
	}
}

func TestStartInBotAction_inviteMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetPreferredLanguage(gomock.Any())
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origGetInvite := getInvite
	getInvite = func(_ context.Context, _ string) (models4debtus.Invite, error) {
		return models4debtus.Invite{}, errors.New("db error")
	}
	defer func() { getInvite = origGetInvite }()

	origDelay := delaySetUserPreferredLocale
	delaySetUserPreferredLocale = func(_ context.Context, _ time.Duration, _, _ string) error { return nil }
	defer func() { delaySetUserPreferredLocale = origDelay }()

	_, err := StartInBotAction(whc, []string{"invite-MYCODE_en"})
	if err == nil {
		t.Fatal("expected error from getInvite")
	}
}

// --- startByLinkCode ---

func TestStartByLinkCode_localeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	whc.EXPECT().SetLocale(gomock.Any()).Return(errors.New("locale error"))

	matches := []string{"invite-CODE_en", "invite", "CODE", "", "", "_en", "en", "", "", ""}
	_, err := startByLinkCode(whc, matches)
	if err == nil {
		t.Fatal("expected error from SetLocale")
	}
}

func TestStartByLinkCode_delayError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetPreferredLanguage(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	whc.EXPECT().SetLocale(gomock.Any()).Return(nil)

	origDelay := delaySetUserPreferredLocale
	delaySetUserPreferredLocale = func(_ context.Context, _ time.Duration, _, _ string) error {
		return errors.New("delay error")
	}
	defer func() { delaySetUserPreferredLocale = origDelay }()

	matches := []string{"invite-CODE_en", "invite", "CODE", "", "", "_en", "en", "", "", ""}
	_, err := startByLinkCode(whc, matches)
	if err == nil {
		t.Fatal("expected error from delaySetUserPreferredLocale")
	}
}

func TestStartByLinkCode_unknownEntityType(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	matches := []string{"unknown-CODE", "unknown", "CODE", "", "", "", "", "", "", ""}
	_, err := startByLinkCode(whc, matches)
	if err != cmds4anybot.ErrUnknownStartParam {
		t.Fatalf("expected ErrUnknownStartParam, got: %v", err)
	}
}

// --- startInvite ---

// notFoundErr wraps dal.ErrRecordNotFound so dal.IsNotFound returns true
type notFoundErr struct{}

func (notFoundErr) Error() string { return "not found" }
func (notFoundErr) Unwrap() error { return dal.ErrRecordNotFound }

func TestStartInvite_getInviteNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origGetInvite := getInvite
	getInvite = func(_ context.Context, _ string) (models4debtus.Invite, error) {
		return models4debtus.Invite{}, notFoundErr{}
	}
	defer func() { getInvite = origGetInvite }()

	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := startInvite(whc, "CODE", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartInvite_getInviteOtherError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origGetInvite := getInvite
	getInvite = func(_ context.Context, _ string) (models4debtus.Invite, error) {
		return models4debtus.Invite{}, errors.New("db error")
	}
	defer func() { getInvite = origGetInvite }()

	_, err := startInvite(whc, "CODE", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartInvite_alreadyClaimed(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origGetInvite := getInvite
	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.MaxClaimsCount = 1
	invite.Data.ClaimedCount = 1
	getInvite = func(_ context.Context, _ string) (models4debtus.Invite, error) {
		return invite, nil
	}
	defer func() { getInvite = origGetInvite }()

	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := startInvite(whc, "CODE", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartInvite_handleInviteOnStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()

	origGetInvite := getInvite
	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.MaxClaimsCount = 0
	invite.Data.CreatedByUserID = "other-user"
	getInvite = func(_ context.Context, _ string) (models4debtus.Invite, error) {
		return invite, nil
	}
	defer func() { getInvite = origGetInvite }()

	origClaim := claimInvite2
	claimInvite2 = func(_ context.Context, _ string, _ models4debtus.Invite, _, _, _ string) error {
		return nil
	}
	defer func() { claimInvite2 = origClaim }()

	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := startInvite(whc, "CODE", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- startReceipt ---

func TestStartReceipt_emptyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	_, err := startReceipt(whc, "", "", "")
	if err == nil {
		t.Fatal("expected error for empty receiptID")
	}
}

func TestStartReceipt_getError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origGet := getReceiptByID
	getReceiptByID = func(_ context.Context, _ string) error {
		return errors.New("db error")
	}
	defer func() { getReceiptByID = origGet }()

	_, err := startReceipt(whc, "RID", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartReceipt_viewOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(nil)

	origGet := getReceiptByID
	getReceiptByID = func(_ context.Context, _ string) error { return nil }
	defer func() { getReceiptByID = origGet }()

	origShow := showReceipt
	showReceipt = func(_ botsfw.WebhookContext, _ string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, nil
	}
	defer func() { showReceipt = origShow }()

	_, err := startReceipt(whc, "RID", "view", "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartReceipt_viewLocaleError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(errors.New("locale error"))

	origGet := getReceiptByID
	getReceiptByID = func(_ context.Context, _ string) error { return nil }
	defer func() { getReceiptByID = origGet }()

	_, err := startReceipt(whc, "RID", "view", "en-US")
	if err == nil {
		t.Fatal("expected error from SetLocale")
	}
}

func TestStartReceipt_defaultOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	origGet := getReceiptByID
	getReceiptByID = func(_ context.Context, _ string) error { return nil }
	defer func() { getReceiptByID = origGet }()

	origAck := acknowledgeReceipt
	acknowledgeReceipt = func(_ botsfw.WebhookContext, _, _ string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, nil
	}
	defer func() { acknowledgeReceipt = origAck }()

	_, err := startReceipt(whc, "RID", "accept", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- startByLinkCode receipt-with-locale path ---

func TestStartByLinkCode_receiptWithLocale(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(nil).Times(2)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetPreferredLanguage(gomock.Any())
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	origDelay := delaySetUserPreferredLocale
	delaySetUserPreferredLocale = func(_ context.Context, _ time.Duration, _, _ string) error { return nil }
	defer func() { delaySetUserPreferredLocale = origDelay }()

	origGet := getReceiptByID
	getReceiptByID = func(_ context.Context, _ string) error { return nil }
	defer func() { getReceiptByID = origGet }()

	origShow := showReceipt
	showReceipt = func(_ botsfw.WebhookContext, _ string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, nil
	}
	defer func() { showReceipt = origShow }()

	// matches: full, entityType, entityCode, -op, op, _locale, locale, _, gaClientId, _
	matches := []string{"receipt-RID-view_en", "receipt", "RID", "-view", "view", "_en", "en", "", "", ""}
	_, err := startByLinkCode(whc, matches)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- handleInviteOnStart ---

func TestHandleInviteOnStart_onboardingWrongUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = InviteIsRelatedToOnboarding
	invite.Data.CreatedByUserID = "other-user"

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
}

func TestHandleInviteOnStart_ownInvite(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = ""
	invite.Data.CreatedByUserID = "user1"

	origSetMenu := setMainMenuKeyboard
	setMainMenuKeyboard = func(_ botsfw.WebhookContext, _ *botmsg.MessageFromBot) error { return nil }
	defer func() { setMainMenuKeyboard = origSetMenu }()

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleInviteOnStart_ownInviteMenuError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = ""
	invite.Data.CreatedByUserID = "user1"

	origSetMenu := setMainMenuKeyboard
	setMainMenuKeyboard = func(_ botsfw.WebhookContext, _ *botmsg.MessageFromBot) error {
		return errors.New("menu error")
	}
	defer func() { setMainMenuKeyboard = origSetMenu }()

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err == nil {
		t.Fatal("expected error from setMainMenuKeyboard")
	}
}

func TestHandleInviteOnStart_unknownRelated(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = "unknown-related"
	invite.Data.CreatedByUserID = "other"

	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleInviteOnStart_onboardingClaimError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()

	origClaim := claimInvite2
	claimInvite2 = func(_ context.Context, _ string, _ models4debtus.Invite, _, _, _ string) error {
		return errors.New("claim error")
	}
	defer func() { claimInvite2 = origClaim }()

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = InviteIsRelatedToOnboarding
	invite.Data.CreatedByUserID = "user1"

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err == nil {
		t.Fatal("expected error from claimInvite2")
	}
}

func TestHandleInviteOnStart_emptyRelatedClaimOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any()).Return(botmsg.MessageFromBot{})

	origClaim := claimInvite2
	claimInvite2 = func(_ context.Context, _ string, _ models4debtus.Invite, _, _, _ string) error {
		return nil
	}
	defer func() { claimInvite2 = origClaim }()

	invite := models4debtus.Invite{}
	invite.Data = new(models4debtus.InviteData)
	invite.Data.Related = ""
	invite.Data.CreatedByUserID = "other"

	_, err := handleInviteOnStart(whc, "CODE", invite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- loginPinCommand ---

func TestLoginPinCommand_matcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	result := loginPinCommand.Matcher(loginPinCommand, whc)
	if result {
		t.Error("expected matcher to return false")
	}
}

func TestLoginPinCommand_badContextParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "onlyone"}).AnyTimes()

	_, err := loginPinCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error for bad context params")
	}
}

func TestLoginPinCommand_badLoginID(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "login-abc_extra"}).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).Return("bad integer")

	_, err := loginPinCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error for non-integer loginID")
	}
}

func TestLoginPinCommand_langSetLocaleError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "login-42_lang-en"}).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(errors.New("locale error"))

	_, err := loginPinCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from SetLocale")
	}
}

func TestLoginPinCommand_assignPinError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "login-42_extra"}).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	origAssign := assignPinCode
	assignPinCode = func(_ context.Context, _ int, _ string) (models4auth.LoginPin, error) {
		return models4auth.LoginPin{}, errors.New("assign error")
	}
	defer func() { assignPinCode = origAssign }()

	_, err := loginPinCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from assignPinCode")
	}
}

func TestLoginPinCommand_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "login-42_extra"}).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	origAssign := assignPinCode
	assignPinCode = func(_ context.Context, _ int, _ string) (models4auth.LoginPin, error) {
		return models4auth.LoginPin{}, nil
	}
	defer func() { assignPinCode = origAssign }()

	_, err := loginPinCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoginPinCommand_langSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Input().Return(fakeTextInput{text: "login-42_lang-en"}).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(nil)

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetPreferredLanguage(gomock.Any()).AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{})

	origAssign := assignPinCode
	assignPinCode = func(_ context.Context, _ int, _ string) (models4auth.LoginPin, error) {
		return models4auth.LoginPin{}, nil
	}
	defer func() { assignPinCode = origAssign }()

	_, err := loginPinCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- fixBalanceCommand ---

func TestFixBalanceCommand_getUserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	origGetUser := getUser
	getUser = func(_ context.Context, _ dal.ReadSession, _ dbo4userus.UserEntry) error {
		return errors.New("db error")
	}
	defer func() { getUser = origGetUser }()

	_, err := fixBalanceCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from getUser")
	}
}

func TestFixBalanceCommand_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().NewMessage("Balance fixed").Return(botmsg.MessageFromBot{})

	origGetUser := getUser
	getUser = func(_ context.Context, _ dal.ReadSession, user dbo4userus.UserEntry) error {
		user.Data.Spaces = map[string]*dbo4userus.UserSpaceBrief{
			"family1": {SpaceBrief: dbo4spaceus.SpaceBrief{Type: coretypes.SpaceTypeFamily}},
		}
		return nil
	}
	defer func() { getUser = origGetUser }()

	db := sneattesting.SetupMemoryDB(t)

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(ctx context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return db.RunReadwriteTransaction(ctx, f)
	}
	defer func() { runReadwriteTransaction = origTx }()

	_, err := fixBalanceCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFixBalanceCommand_txError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	origGetUser := getUser
	getUser = func(_ context.Context, _ dal.ReadSession, _ dbo4userus.UserEntry) error {
		return nil
	}
	defer func() { getUser = origGetUser }()

	origTx := runReadwriteTransaction
	runReadwriteTransaction = func(_ context.Context, _ func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return errors.New("tx error")
	}
	defer func() { runReadwriteTransaction = origTx }()

	_, err := fixBalanceCommand.Action(whc)
	if err == nil {
		t.Fatal("expected error from runReadwriteTransaction")
	}
}

// --- SetPrimaryCurrency ---

func TestSetPrimaryCurrency_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Input().Return(fakeTextInput{text: "USD"}).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{})

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	origRun := runUserWorker
	runUserWorker = func(ctx facade.ContextWithUser, _ bool, worker func(facade.ContextWithUser, dal.ReadwriteTransaction, *dal4userus.UserWorkerParams) error) error {
		params := &dal4userus.UserWorkerParams{User: dbo4userus.NewUserEntry("user1")}
		return worker(ctx, nil, params)
	}
	defer func() { runUserWorker = origRun }()

	_, err := SetPrimaryCurrency.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetPrimaryCurrency_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Input().Return(fakeTextInput{text: "USD"}).AnyTimes()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo("")
	whc.EXPECT().ChatData().Return(chatData)

	origRun := runUserWorker
	runUserWorker = func(_ facade.ContextWithUser, _ bool, _ func(facade.ContextWithUser, dal.ReadwriteTransaction, *dal4userus.UserWorkerParams) error) error {
		return errors.New("worker error")
	}
	defer func() { runUserWorker = origRun }()

	_, err := SetPrimaryCurrency.Action(whc)
	if err == nil {
		t.Fatal("expected error from RunUserWorker")
	}
}

// --- default seam closures (getInvite / getReceiptByID / claimInvite2) ---

// fakeInviteDal implements dal4debtus.InviteDal; embedding the interface
// satisfies unused methods (they would nil-panic, but tests never call them).
type fakeInviteDal struct {
	dal4debtus.InviteDal
	getErr   error
	claimErr error
}

func (f fakeInviteDal) GetInvite(context.Context, dal.ReadSession, string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, f.getErr
}

func (f fakeInviteDal) ClaimInvite2(context.Context, string, models4debtus.Invite, string, string, string) error {
	return f.claimErr
}

// fakeReceiptDal implements dal4debtus.ReceiptDal via embedding.
type fakeReceiptDal struct {
	dal4debtus.ReceiptDal
	getErr error
}

func (f fakeReceiptDal) GetReceiptByID(context.Context, dal.ReadSession, string) (models4debtus.ReceiptEntry, error) {
	return models4debtus.ReceiptEntry{}, f.getErr
}

func TestDefaultSeamClosures(t *testing.T) {
	origDefault := dal4debtus.Default
	t.Cleanup(func() { dal4debtus.Default = origDefault })

	dal4debtus.Default.Invite = fakeInviteDal{getErr: errors.New("invite err"), claimErr: errors.New("claim err")}
	dal4debtus.Default.Receipt = fakeReceiptDal{getErr: errors.New("receipt err")}

	ctx := context.Background()

	if _, err := getInvite(ctx, "CODE"); err == nil {
		t.Error("expected error from getInvite default closure")
	}
	if err := getReceiptByID(ctx, "RID"); err == nil {
		t.Error("expected error from getReceiptByID default closure")
	}
	if err := claimInvite2(ctx, "CODE", models4debtus.Invite{}, "u1", "telegram", "bot"); err == nil {
		t.Error("expected error from claimInvite2 default closure")
	}
}
