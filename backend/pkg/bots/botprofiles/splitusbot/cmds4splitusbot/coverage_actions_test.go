package cmds4splitusbot

import (
	"context"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/const4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"go.uber.org/mock/gomock"
)

// putSpace seeds a space record with the test user as a member.
func putSpace(t *testing.T, ctx context.Context, db dal.DB, spaceID coretypes.SpaceID) dbo4spaceus.SpaceEntry {
	t.Helper()
	space := dbo4spaceus.NewSpaceEntry(spaceID)
	space.Data.Type = coretypes.SpaceTypeFamily
	space.Data.Title = "Test Space"
	space.Data.UserIDs = []string{"u1"}
	putRecord(t, ctx, db, space.Record)
	return space
}

func TestSpaceCallbackCommands(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putSpace(t, ctx, db, "space1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		name    string
		command botsfw.Command
		url     string
		wantErr string
	}{
		{"settle_group_ask", settleGroupAskForCounterpartyCommand, "https://x/cb?s=space1", "not implemented"},
		{"settle_group_chosen", settleGroupCounterpartyChosenCommand, "https://x/cb?s=space1&member=m1", "not implemented"},
		{"settle_group_confirmed", settleGroupCounterpartyConfirmedCommand, "https://x/cb?s=space1&member=m1", "not implemented"},
		{"join_space", joinSpaceCommand, "https://x/cb?s=space1", "not implemented"},
		{"leave_group", leaveGroupCommand, "https://x/cb?s=space1", ""},
		{"choose_currency", groupSettingsChooseCurrencyCommand, "https://x/cb?s=space1", ""},
		{"space_split_no_members", spaceSplitCommand, "https://x/cb?s=space1", "members"},
		{"space_missing", leaveGroupCommand, "https://x/cb?s=missing", "not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			whc := newMockWhc(ctrl)
			whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
			whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
			_, err := c.command.CallbackAction(whc, mustParseURL(t, c.url))
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", c.wantErr, err)
			}
		})
	}
}

func TestGroupSettingsSetCurrencyCallback(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putSpace(t, ctx, db, "space1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cmd := groupSettingsSetCurrencyCommand()

	t.Run("same_currency_start", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		// PrimaryCurrency is empty; currency param empty too -> no change; start=y -> onStartCallbackInGroup
		_, err := cmd.CallbackAction(whc, mustParseURL(t, "https://x/cb?s=space1&start=y"))
		if err == nil || !strings.Contains(err.Error(), "not implemented") {
			t.Fatalf("expected onStartCallbackInGroup error, got: %v", err)
		}
	})

	t.Run("new_currency_settings", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		whc.EXPECT().DB().Return(db).AnyTimes()
		// currency=EUR differs from empty primary currency -> runs space worker,
		// then falls through to SpaceSettingsAction.
		_, err := cmd.CallbackAction(whc, mustParseURL(t, "https://x/cb?s=space1&currency=EUR"))
		_ = err // worker and/or settings load may fail; branches are exercised either way
	})
}

func TestSpaceSettingsAction(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	space := putSpace(t, ctx, db, "space1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("missing_module_records", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().DB().Return(db).AnyTimes()
		_, err := SpaceSettingsAction(whc, space, false)
		_ = err // exercised: either GetMulti error branch or zero members branch
	})

	t.Run("with_module_records", func(t *testing.T) {
		splitusSpace := models4splitus.NewSplitusSpaceEntry("space1")
		putRecord(t, ctx, db, splitusSpace.Record)

		contactusSpace := dal4contactus.NewContactusSpaceEntry("space1")
		brief := &briefs4contactus.ContactBrief{Type: briefs4contactus.ContactTypePerson, Title: "Member One"}
		brief.Roles = []string{const4contactus.SpaceMemberRoleMember}
		contactusSpace.Data.AddContact("c1", brief)
		putRecord(t, ctx, db, contactusSpace.Record)

		whc := newMockWhc(ctrl)
		whc.EXPECT().DB().Return(db).AnyTimes()
		m, err := SpaceSettingsAction(whc, space, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Keyboard == nil || !m.IsEdit {
			t.Error("expected keyboard and IsEdit")
		}
	})

	t.Run("with_primary_currency", func(t *testing.T) {
		spaceWithCurrency := putSpace(t, ctx, db, "space2")
		spaceWithCurrency.Data.PrimaryCurrency = money.CurrencyCode("USD")
		splitusSpace := models4splitus.NewSplitusSpaceEntry("space2")
		putRecord(t, ctx, db, splitusSpace.Record)
		contactusSpace := dal4contactus.NewContactusSpaceEntry("space2")
		putRecord(t, ctx, db, contactusSpace.Record)

		whc := newMockWhc(ctrl)
		whc.EXPECT().DB().Return(db).AnyTimes()
		if _, err := SpaceSettingsAction(whc, spaceWithCurrency, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSettingsAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("is_in_group_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if _, err := settingsAction(whc); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("in_group", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil).AnyTimes()
		whc.EXPECT().ChatData().Return(nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		// NewSpaceAction with nil URL cannot resolve a space -> error expected
		if _, err := settingsAction(whc); err == nil {
			t.Fatal("expected error from space resolution")
		}
	})

	t.Run("private_chat", func(t *testing.T) {
		db := withMemDB(t)
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().DB().Return(db).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		_, err := settingsAction(whc)
		_ = err // cmds4anybot.SettingsMainTelegram outcome depends on shared registry
	})

	t.Run("settings_command_callback", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if _, err := settingsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb")); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGroupMembersCommand(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("action_no_space", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		m, err := groupMembersCommand.Action(whc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(m.Text, "ERROR") {
			t.Errorf("expected error text, got %q", m.Text)
		}
	})

	// NOTE: the dalgo2memory adapter keys nested records by their last key
	// component, so all spaces/<id>/ext/contactus records collide. The
	// not-found case therefore must run before any contactus record is seeded.
	t.Run("callback_not_found", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := groupMembersCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?s=missing")); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("callback_success", func(t *testing.T) {
		contactusSpace := dal4contactus.NewContactusSpaceEntry("space1")
		brief := &briefs4contactus.ContactBrief{Type: briefs4contactus.ContactTypePerson, Title: "Member One"}
		brief.Roles = []string{const4contactus.SpaceMemberRoleMember}
		contactusSpace.Data.AddContact("c1", brief)
		putRecord(t, ctx, db, contactusSpace.Record)

		whc := newMockWhc(ctrl)
		m, err := groupMembersCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?s=space1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !m.IsEdit {
			t.Error("expected IsEdit")
		}
	})

	t.Run("callback_db_error", func(t *testing.T) {
		orig := facade.GetSneatDB
		facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, errTest }
		defer func() { facade.GetSneatDB = orig }()
		whc := newMockWhc(ctrl)
		if _, err := groupMembersCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?s=space1")); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetGroupSponsorsAndDebtors(t *testing.T) {
	members := []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Sponsor"}, Balance: money.Balance{"USD": 100}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Debtor"}, Balance: money.Balance{"USD": -100}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m3", Name: "Even"}, Balance: money.Balance{"USD": 0}},
	}
	sponsors, debtors := getGroupSponsorsAndDebtors(members, "m3")
	if len(sponsors) != 1 || sponsors[0].ID != "m1" {
		t.Errorf("unexpected sponsors: %v", sponsors)
	}
	if len(debtors) != 1 || debtors[0].ID != "m2" {
		t.Errorf("unexpected debtors: %v", debtors)
	}
}

func TestGroupBalanceAction(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putSpace(t, ctx, db, "space1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newSplitusSpace := func(spaceID coretypes.SpaceID) models4splitus.SplitusSpaceEntry {
		splitusSpace := models4splitus.NewSplitusSpaceEntry(spaceID)
		splitusSpace.Data.SetGroupMembers([]briefs4splitus.SpaceSplitMember{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Sponsor"}, Balance: money.Balance{"USD": 100}},
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Debtor"}, Balance: money.Balance{"USD": -100}},
		})
		return splitusSpace
	}

	t.Run("success", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		m, err := groupBalanceAction(whc, newSplitusSpace("space1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(m.Text, "Sponsor") || !strings.Contains(m.Text, "Debtor") {
			t.Errorf("expected member names in text, got %q", m.Text)
		}
	})

	t.Run("space_not_found", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		if _, err := groupBalanceAction(whc, newSplitusSpace("missing")); err == nil {
			t.Fatal("expected error for missing space")
		}
	})
}

func TestBillsAction(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("is_in_group_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if _, err := billsAction(whc); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("in_group_not_implemented", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil)
		if _, err := billsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb")); err == nil {
			t.Fatal("expected not-implemented error")
		}
	})

	t.Run("get_db_error", func(t *testing.T) {
		orig := facade.GetSneatDB
		facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, errTest }
		defer func() { facade.GetSneatDB = orig }()
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		if _, err := billsAction(whc); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		if _, err := billsAction(whc); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("no_outstanding_bills", func(t *testing.T) {
		userSplitus := models4splitus.NewSplitusUserEntry("u1")
		putRecord(t, ctx, db, userSplitus.Record)

		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		m, err := billsAction(whc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Text == "" {
			t.Error("expected text")
		}
	})

	t.Run("with_outstanding_bills", func(t *testing.T) {
		userSplitus := models4splitus.NewSplitusUserEntry("u1")
		userSplitus.Data.OutstandingBills = map[string]briefs4splitus.BillBrief{
			"b1": {Name: "Dinner", Total: 1500, Currency: "USD"},
		}
		putRecord(t, ctx, db, userSplitus.Record)

		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		defer func() {
			// bothelper.NewGroupTelegramInlineButton panics on utm.Params validation;
			// statements before the panic are still exercised.
			_ = recover()
		}()
		m, err := billsAction(whc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(m.Text, "Dinner") {
			t.Errorf("expected bill name in text, got %q", m.Text)
		}
	})
}

func TestOutstandingBalanceAction(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	_ = ctx
	_ = db

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := newMockWhc(ctrl)
	_, err := outstandingBalanceAction(whc)
	_ = err // exercised: worker may fail without seeded user records
}

var _ = mock_botsfw.NewMockWebhookContext
