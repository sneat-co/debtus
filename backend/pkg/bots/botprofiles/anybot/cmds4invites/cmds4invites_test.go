package cmds4invites

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botinput"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfwmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/const4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/invitus/dbo4invitus"
	"github.com/sneat-co/sneat-core-modules/invitus/facade4invitus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp"
	"github.com/strongo/strongoapp/person"
	"github.com/strongo/validation"
	"go.uber.org/mock/gomock"
)

// --- RegisterCommands ---

func TestRegisterCommands(t *testing.T) {
	var registered []botsfw.Command
	router := &fakeRegisterer{&registered}
	RegisterCommands("test_bot", router)
	if len(registered) == 0 {
		t.Fatal("expected commands to be registered")
	}
}

type fakeRegisterer struct {
	commands *[]botsfw.Command
}

func (f *fakeRegisterer) RegisterCommands(commands ...botsfw.Command) {
	*f.commands = append(*f.commands, commands...)
}

// --- chosenInlineResultAction ---

func TestChosenInlineResultAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	m, err := chosenInlineResultAction(whc, nil, &url.URL{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

// --- startInviteCommandAction guard clauses ---

func TestStartInviteCommandAction_nilUrl(t *testing.T) {
	handled, _, err := startInviteCommandAction(nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
}

func TestStartInviteCommandAction_missingSpaceParam(t *testing.T) {
	u, _ := url.Parse("?invite=inv1&pin=1234&o=accept")
	handled, m, err := startInviteCommandAction(nil, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_missingInviteParam(t *testing.T) {
	u, _ := url.Parse("?s=family!fam1&pin=1234&o=accept")
	handled, m, err := startInviteCommandAction(nil, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_missingPinParam(t *testing.T) {
	u, _ := url.Parse("?s=family!fam1&invite=inv1&o=accept")
	handled, m, err := startInviteCommandAction(nil, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_missingOpParam(t *testing.T) {
	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234")
	handled, m, err := startInviteCommandAction(nil, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

// --- startInviteCommandAction view path (DB via facade.GetSneatDB seam) ---

func TestStartInviteCommandAction_viewOp_dbError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	origGetSneatDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origGetSneatDB }()
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) {
		return nil, errors.New("db not available")
	}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=view")
	_, _, err := startInviteCommandAction(whc, "", u)
	if err == nil {
		t.Fatal("expected error from GetSneatDB")
	}
}

func TestStartInviteCommandAction_viewOp_getMultiError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Return(errors.New("not found")).AnyTimes()

	origGetSneatDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origGetSneatDB }()
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) {
		return mockDB, nil
	}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=view")
	_, _, err := startInviteCommandAction(whc, "", u)
	if err == nil {
		t.Fatal("expected error from GetMulti")
	}
}

func TestStartInviteCommandAction_viewOp_spaceIDMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().GetMulti(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, records []dal.Record) error {
			// Set invite SpaceID to something different from request
			for _, r := range records {
				_ = r
			}
			return nil
		}).AnyTimes()

	origGetSneatDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origGetSneatDB }()
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) {
		return mockDB, nil
	}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=view")
	_, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// invite.Data.SpaceID == "" != "fam1" => mismatch warning
	if m.Text == "" {
		t.Error("expected warning text for space ID mismatch")
	}
}

func TestStartInviteCommandAction_viewOp_pinMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().GetMulti(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, records []dal.Record) error {
			// Mark all records as retrieved, then set invite fields
			for _, r := range records {
				r.SetError(nil)
			}
			if inviteDbo, ok := records[0].Data().(*dbo4invitus.InviteDbo); ok {
				inviteDbo.SpaceID = "fam1"
				inviteDbo.Pin = "wrong-pin"
			}
			return nil
		}).AnyTimes()

	origGetSneatDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origGetSneatDB }()
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) {
		return mockDB, nil
	}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=view")
	_, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected warning text for pin mismatch")
	}
}

// --- startInviteCommandAction claim path ---

func TestStartInviteCommandAction_claimDeclineAlreadyAccepted(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return facade4invitus.ClaimPersonalInviteResponse{}, facade4invitus.ErrInviteAlreadyAccepted
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=decline")
	handled, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_claimExpired(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return facade4invitus.ClaimPersonalInviteResponse{}, facade4invitus.ErrInviteExpired
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=accept")
	_, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_claimPinMismatch(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return facade4invitus.ClaimPersonalInviteResponse{}, facade4invitus.ErrInvitePinDoesNotMatch
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=accept")
	_, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected warning text")
	}
}

func TestStartInviteCommandAction_claimOtherError(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return facade4invitus.ClaimPersonalInviteResponse{}, errors.New("unknown error")
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=accept")
	_, _, err := startInviteCommandAction(whc, "", u)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartInviteCommandAction_claimSuccess_familyAccepted(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	space.Data.Type = coretypes.SpaceTypeFamily
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Status = dbo4invitus.InviteStatusAccepted
	// IsClaimed() returns true when Status == Accepted
	contactusSpace := dal4contactus.NewContactusSpaceEntry("fam1")
	resp := facade4invitus.ClaimPersonalInviteResponse{
		Invite:         invite,
		Space:          space,
		ContactusSpace: contactusSpace,
	}
	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return resp, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=accept")
	handled, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if m.Text == "" {
		t.Error("expected message text")
	}
}

func TestStartInviteCommandAction_claimSuccess_nonFamilyDeclined(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	space := dbo4spaceus.NewSpaceEntry("cmp1")
	space.Data.Type = coretypes.SpaceTypeCompany
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Status = dbo4invitus.InviteStatusDeclined
	// IsClaimed() returns true when Status == Declined

	// Add members to cover sort.Slice + member loop with all gender branches
	cs := dal4contactus.NewContactusSpaceEntry("cmp1")
	maleMember := &briefs4contactus.ContactBrief{Gender: dbmodels.GenderMale}
	maleMember.Names = &person.NameFields{FullName: "Alice Male"}
	maleMember.Roles = []string{const4contactus.SpaceMemberRoleMember}
	cs.Data.AddContact("m1", maleMember)
	femaleMember := &briefs4contactus.ContactBrief{Gender: dbmodels.GenderFemale}
	femaleMember.Names = &person.NameFields{FullName: "Bob Female"}
	femaleMember.Roles = []string{const4contactus.SpaceMemberRoleMember}
	cs.Data.AddContact("f1", femaleMember)
	unknownMember := &briefs4contactus.ContactBrief{}
	unknownMember.Names = &person.NameFields{FullName: "Charlie Unknown"}
	unknownMember.Roles = []string{const4contactus.SpaceMemberRoleMember}
	cs.Data.AddContact("u1", unknownMember)

	resp := facade4invitus.ClaimPersonalInviteResponse{
		Invite:         invite,
		Space:          space,
		ContactusSpace: cs,
	}
	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return resp, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("").AnyTimes()

	u, _ := url.Parse("?s=company!cmp1&invite=inv1&pin=1234&o=decline")
	handled, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if m.Text == "" {
		t.Error("expected message text")
	}
}

func TestStartInviteCommandAction_claimSuccess_defaultStatus_notClaimed(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	space.Data.Type = coretypes.SpaceTypeFamily
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Status = "pending"
	invite.Data.From.Title = "Alice"
	// IsClaimed() = false when Status is "pending" and Claimed is zero
	resp := facade4invitus.ClaimPersonalInviteResponse{
		Invite:         invite,
		Space:          space,
		ContactusSpace: dal4contactus.NewContactusSpaceEntry("fam1"),
	}
	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return resp, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("Accept").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=accept")
	handled, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard for unclaimed invite")
	}
}

// --- AskInviteAddress ---

func TestAskInviteAddress_initialAsk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(false).AnyTimes()
	chatData.EXPECT().PushStepToAwaitingReplyTo(gomock.Any()).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()

	cmd := AskInviteAddress("email", "✉️", "Send by email", "msg_code", "invalid_code")
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

func TestAskInviteAddress_awaitingReply_invalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(true).AnyTimes()

	mockTextMsg := mock_botinput.NewMockTextMessage(ctrl)
	mockTextMsg.EXPECT().Text().Return("not-an-email").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(mockTextMsg).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("text").AnyTimes()

	cmd := AskInviteAddress("email", "✉️", "Send by email", "msg_code", "invalid_code")
	m, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = m
}

func TestAskInviteAddress_awaitingReply_validEmail_sendError(t *testing.T) {
	origSend := sendInviteByEmail
	defer func() { sendInviteByEmail = origSend }()
	sendInviteByEmail = func(ec strongoapp.ExecutionContext, translator i18n.SingleLocaleTranslator, fromName, toEmail, toName, inviteCode, telegramBotID, utmSource string) (string, error) {
		return "", errors.New("send error")
	}

	origDefault := dal4debtus.Default.Invite
	defer func() { dal4debtus.Default.Invite = origDefault }()
	dal4debtus.Default.Invite = &fakeInviteDal{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(true).AnyTimes()

	mockUser := mock_botinput.NewMockUser(ctrl)
	mockUser.EXPECT().GetFirstName().Return("Tester").AnyTimes()

	mockTextMsg := mock_botinput.NewMockTextMessage(ctrl)
	mockTextMsg.EXPECT().Text().Return("test@example.com").AnyTimes()
	mockTextMsg.EXPECT().GetSender().Return(mockUser).AnyTimes()
	mockPlatform := &fakeBotPlatform{id: "telegram"}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(mockTextMsg).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().BotPlatform().Return(mockPlatform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().ExecutionContext().Return(nil).AnyTimes()

	cmd := AskInviteAddress("email", "✉️", "Send by email", "msg_code", "invalid_code")
	_, err := cmd.Action(whc)
	if err == nil {
		t.Fatal("expected error from sendInviteByEmail")
	}
}

// --- inviteContactToJoinInlineQueryAction ---

// TestInviteContactToJoinInlineQueryAction_getClientInfo covers the getClientInfo closure body (lines 68-76)
// by having the stub actually call getClientInfo() before returning.
// Two sub-tests: one with non-empty Code (HostOrApp != hostPlatform), one with empty Code (triggers the inner if branch).
func TestInviteContactToJoinInlineQueryAction_getClientInfo(t *testing.T) {
	for _, tc := range []struct {
		name    string
		botCode string
		botID   string
	}{
		{"with_code", "testbot", "12345"}, // HostOrApp = "telegram@testbot" != "telegram@"
		{"empty_code", "", "bot-id-456"},  // HostOrApp = "telegram@" == hostPlatform → appends ID
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origCreate := createOrReuseInviteToContact
			defer func() { createOrReuseInviteToContact = origCreate }()
			createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
				_ = getClientInfo() // exercise the closure body
				return facade4invitus.CreateInviteResponse{}, errors.New("db error")
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
			mockInlineQuery.EXPECT().GetInlineQueryID().Return("query1").AnyTimes()

			httpReq, _ := http.NewRequest("GET", "/", nil)

			botCtx := botsfw.BotContext{
				BotSettings: &botsfw.BotSettings{
					Code: tc.botCode,
					ID:   tc.botID,
				},
			}

			whc := mock_botsfw.NewMockWebhookContext(ctrl)
			whc.EXPECT().AppUserID().Return("user1").AnyTimes()
			whc.EXPECT().Context().Return(context.Background()).AnyTimes()
			whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
			whc.EXPECT().Request().Return(httpReq).AnyTimes()
			whc.EXPECT().BotContext().Return(botCtx).AnyTimes()
			whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()

			u, _ := url.Parse("?s=family!fam1&c=contact1&l=en-US")
			_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestInviteContactToJoinInlineQueryAction_error(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{}, errors.New("db error")
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("query1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1&l=en-US")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- inviteContactToJoinInlineQueryAction success ---

func TestInviteContactToJoinInlineQueryAction_badRequestError(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{}, validation.NewBadRequestError(errors.New("missing field"))
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("query1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInviteContactToJoinInlineQueryAction_success(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Pin = "4321"
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{
			Invite: invite,
			Space:  space,
		}, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("query1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(nil).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1&l=en-US")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteContactToJoinInlineQueryAction_localeFromSpace(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	space.Data.PreferredLocale = "fr"
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Pin = "4321"
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{Invite: invite, Space: space}, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("q1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().SetLocale("fr").Return(nil).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()

	// No l= in URL → locale from space.Data.GetPreferredLocale()
	u, _ := url.Parse("?s=family!fam1&c=contact1")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteContactToJoinInlineQueryAction_setLocaleError(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	space.Data.PreferredLocale = "fr"
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Pin = "4321"
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{Invite: invite, Space: space}, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("q1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().SetLocale(gomock.Any()).Return(errors.New("unsupported locale")).AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err == nil {
		t.Fatal("expected error from SetLocale")
	}
}

func TestInviteContactToJoinInlineQueryAction_appUserDataLocale(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()

	// Space has no preferred locale, no l= in URL → falls through to AppUserData
	space := dbo4spaceus.NewSpaceEntry("fam1")
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Pin = "4321"
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{Invite: invite, Space: space}, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("q1").AnyTimes()

	mockAdapter := &fakeAppUserAdapter{locale: "de"}
	mockAppUserData := &fakeAppUserData{adapter: mockAdapter}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().AppUserData().Return(mockAppUserData, nil).AnyTimes()
	whc.EXPECT().SetLocale("de").Return(nil).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteContactToJoinInlineQueryAction_appUserDataError(t *testing.T) {
	origCreate := createOrReuseInviteToContact
	defer func() { createOrReuseInviteToContact = origCreate }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Pin = "4321"
	createOrReuseInviteToContact = func(ctx facade.ContextWithUser, req facade4invitus.InviteContactRequest, getClientInfo func() dbmodels.RemoteClientInfo) (facade4invitus.CreateInviteResponse, error) {
		return facade4invitus.CreateInviteResponse{Invite: invite, Space: space}, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	mockInlineQuery.EXPECT().GetInlineQueryID().Return("q1").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Request().Return(nil).AnyTimes()
	whc.EXPECT().BotContext().Return(botsfw.BotContext{}).AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().AppUserData().Return(nil, errors.New("db error")).AnyTimes()

	u, _ := url.Parse("?s=family!fam1&c=contact1")
	_, err := inviteContactToJoinInlineQueryAction(whc, mockInlineQuery, u)
	if err == nil {
		t.Fatal("expected error from AppUserData")
	}
}

// --- askInviteAddressCallbackCommand ---

func TestAskInviteAddressCallbackCommand_emailBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockResponder := mock_botsfw.NewMockWebhookResponder(ctrl)
	mockResponder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(mock_botsfwmodels.NewMockBotChatData(ctrl)).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}, nil).AnyTimes()
	whc.EXPECT().Responder().Return(mockResponder).AnyTimes()

	// echoSelection always returns non-nil error (wraps nil with fmt.Errorf),
	// so the callback returns after echoSelection without reaching Action(whc).
	u, _ := url.Parse("?by=email")
	_, err := askInviteAddressCallbackCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from echoSelection wrapping nil")
	}
}

func TestAskInviteAddressCallbackCommand_smsBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockResponder := mock_botsfw.NewMockWebhookResponder(ctrl)
	mockResponder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(mock_botsfwmodels.NewMockBotChatData(ctrl)).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}, nil).AnyTimes()
	whc.EXPECT().Responder().Return(mockResponder).AnyTimes()

	// echoSelection always returns non-nil error (wraps nil with fmt.Errorf),
	// so the callback returns after echoSelection without reaching Action(whc).
	u, _ := url.Parse("?by=sms")
	_, err := askInviteAddressCallbackCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from echoSelection wrapping nil")
	}
}

func TestAskInviteAddressCallbackCommand_editMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(mock_botsfwmodels.NewMockBotChatData(ctrl)).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}, errors.New("edit error")).AnyTimes()

	u, _ := url.Parse("?by=email")
	_, err := askInviteAddressCallbackCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error from NewEditMessage")
	}
}

func TestAskInviteAddressCallbackCommand_emptyBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(mock_botsfwmodels.NewMockBotChatData(ctrl)).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("not implemented").AnyTimes()

	u, _ := url.Parse("?by=")
	_, err := askInviteAddressCallbackCommand.CallbackAction(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAskInviteAddressCallbackCommand_defaultBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(mock_botsfwmodels.NewMockBotChatData(ctrl)).AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	u, _ := url.Parse("?by=unknown")
	_, err := askInviteAddressCallbackCommand.CallbackAction(whc, u)
	if err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

// --- startInviteCommandAction additional branches ---

func TestStartInviteCommandAction_claimSuccess_familyDeclined(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	space := dbo4spaceus.NewSpaceEntry("fam1")
	space.Data.Type = coretypes.SpaceTypeFamily
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Status = dbo4invitus.InviteStatusDeclined
	resp := facade4invitus.ClaimPersonalInviteResponse{
		Invite:         invite,
		Space:          space,
		ContactusSpace: dal4contactus.NewContactusSpaceEntry("fam1"),
	}
	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return resp, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("").AnyTimes()

	u, _ := url.Parse("?s=family!fam1&invite=inv1&pin=1234&o=decline")
	handled, m, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if m.Text == "" {
		t.Error("expected message text")
	}
}

func TestStartInviteCommandAction_claimSuccess_nonFamilyAccepted(t *testing.T) {
	origClaim := claimPersonalInvite
	defer func() { claimPersonalInvite = origClaim }()

	space := dbo4spaceus.NewSpaceEntry("cmp1")
	space.Data.Type = coretypes.SpaceTypeCompany
	invite := facade4invitus.NewInviteEntry("inv1")
	invite.Data.Status = dbo4invitus.InviteStatusAccepted
	resp := facade4invitus.ClaimPersonalInviteResponse{
		Invite:         invite,
		Space:          space,
		ContactusSpace: dal4contactus.NewContactusSpaceEntry("cmp1"),
	}
	claimPersonalInvite = func(ctx facade.ContextWithUser, req facade4invitus.ClaimPersonalInviteRequest) (facade4invitus.ClaimPersonalInviteResponse, error) {
		return resp, nil
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("").AnyTimes()

	u, _ := url.Parse("?s=company!cmp1&invite=inv1&pin=1234&o=accept")
	handled, _, err := startInviteCommandAction(whc, "", u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
}

// --- command action bodies ---

func TestInviteCommandAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().SetAwaitingReplyTo(gomock.Any()).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("text").AnyTimes()
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()

	_, err := inviteCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateMassInviteCommandAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().NewMessage(gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()

	_, err := createMassInviteCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- AskInviteAddress success path ---

func TestAskInviteAddress_awaitingReply_validEmail_success(t *testing.T) {
	origSend := sendInviteByEmail
	defer func() { sendInviteByEmail = origSend }()
	sendInviteByEmail = func(ec strongoapp.ExecutionContext, translator i18n.SingleLocaleTranslator, fromName, toEmail, toName, inviteCode, telegramBotID, utmSource string) (string, error) {
		return "email123", nil
	}

	origDefault := dal4debtus.Default.Invite
	defer func() { dal4debtus.Default.Invite = origDefault }()
	dal4debtus.Default.Invite = &fakeInviteDal{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(true).AnyTimes()

	mockUser := mock_botinput.NewMockUser(ctrl)
	mockUser.EXPECT().GetFirstName().Return("Tester").AnyTimes()

	mockTextMsg := mock_botinput.NewMockTextMessage(ctrl)
	mockTextMsg.EXPECT().Text().Return("test@example.com").AnyTimes()
	mockTextMsg.EXPECT().GetSender().Return(mockUser).AnyTimes()
	mockPlatform := &fakeBotPlatform{id: "telegram"}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(mockTextMsg).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().BotPlatform().Return(mockPlatform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().ExecutionContext().Return(nil).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}).AnyTimes()

	cmd := AskInviteAddress("email", "✉️", "Send by email", "msg_code", "invalid_code")
	_, err := cmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAskInviteAddress_awaitingReply_createInviteError(t *testing.T) {
	origDefault := dal4debtus.Default.Invite
	defer func() { dal4debtus.Default.Invite = origDefault }()
	dal4debtus.Default.Invite = &fakeInviteDalWithError{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatData := mock_botsfwmodels.NewMockBotChatData(ctrl)
	chatData.EXPECT().IsAwaitingReplyTo(gomock.Any()).Return(true).AnyTimes()

	mockUser := mock_botinput.NewMockUser(ctrl)
	mockUser.EXPECT().GetFirstName().Return("Tester").AnyTimes()

	mockTextMsg := mock_botinput.NewMockTextMessage(ctrl)
	mockTextMsg.EXPECT().Text().Return("test@example.com").AnyTimes()
	mockTextMsg.EXPECT().GetSender().Return(mockUser).AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(mockTextMsg).AnyTimes()
	whc.EXPECT().AppUserID().Return("user1").AnyTimes()
	whc.EXPECT().BotPlatform().Return(&fakeBotPlatform{id: "telegram"}).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	cmd := AskInviteAddress("email", "✉️", "Send by email", "msg_code", "invalid_code")
	_, err := cmd.Action(whc)
	if err == nil {
		t.Fatal("expected error from CreatePersonalInvite")
	}
}

// --- fake helpers ---

type fakeBotPlatform struct {
	id string
}

func (f *fakeBotPlatform) ID() string      { return f.id }
func (f *fakeBotPlatform) Version() string { return "1.0" }

type fakeInviteDalWithError struct{}

func (f *fakeInviteDalWithError) GetInvite(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, nil
}
func (f *fakeInviteDalWithError) ClaimInvite(_ context.Context, _ string, _, _, _ string) error {
	return nil
}
func (f *fakeInviteDalWithError) ClaimInvite2(_ context.Context, _ string, _ models4debtus.Invite, _ string, _, _ string) error {
	return nil
}
func (f *fakeInviteDalWithError) CreatePersonalInvite(_ strongoapp.ExecutionContext, _ string, _ models4debtus.InviteBy, _, _, _, _ string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, errors.New("create invite failed")
}
func (f *fakeInviteDalWithError) CreateMassInvite(_ strongoapp.ExecutionContext, _ string, _ string, _ int32, _ string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, nil
}

type fakeInviteDal struct{}

func (f *fakeInviteDal) GetInvite(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, nil
}
func (f *fakeInviteDal) ClaimInvite(_ context.Context, _ string, _, _, _ string) error {
	return nil
}
func (f *fakeInviteDal) ClaimInvite2(_ context.Context, _ string, _ models4debtus.Invite, _ string, _, _ string) error {
	return nil
}
func (f *fakeInviteDal) CreatePersonalInvite(_ strongoapp.ExecutionContext, _ string, _ models4debtus.InviteBy, _, _, _, _ string) (models4debtus.Invite, error) {
	inv := models4debtus.Invite{}
	inv.ID = "fake-invite"
	inv.Data = &models4debtus.InviteData{}
	inv.Record = dal.NewRecordWithData(dal.NewKeyWithID("invites", "fake-invite"), inv.Data)
	return inv, nil
}
func (f *fakeInviteDal) CreateMassInvite(_ strongoapp.ExecutionContext, _ string, _ string, _ int32, _ string) (models4debtus.Invite, error) {
	return models4debtus.Invite{}, nil
}

type fakeAppUserAdapter struct {
	locale string
}

func (f *fakeAppUserAdapter) SetBotUserID(_, _, _ string)       {}
func (f *fakeAppUserAdapter) SetNames(_, _, _ string) error     { return nil }
func (f *fakeAppUserAdapter) SetPreferredLocale(_ string) error { return nil }
func (f *fakeAppUserAdapter) GetPreferredLocale() string        { return f.locale }

type fakeAppUserData struct {
	adapter botsfwmodels.AppUserAdapter
}

func (f *fakeAppUserData) BotsFwAdapter() botsfwmodels.AppUserAdapter { return f.adapter }
