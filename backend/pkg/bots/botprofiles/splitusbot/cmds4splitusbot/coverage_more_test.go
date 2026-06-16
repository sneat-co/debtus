package cmds4splitusbot

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// newMockWhc returns a MockWebhookContext with the most common expectations
// preset, including a nil ChatData.
func newMockWhc(ctrl *gomock.Controller) *mock_botsfw.MockWebhookContext {
	whc := newMockWhcNoChatData(ctrl)
	whc.EXPECT().ChatData().Return(nil).AnyTimes()
	return whc
}

// newMockWhcNoChatData returns a MockWebhookContext with common expectations
// preset but without a ChatData expectation, so tests can set their own.
// Translate delegates to the real translator so that the shared common4all
// template cache (keyed by template name + locale) is not polluted with raw keys.
func newMockWhcNoChatData(ctrl *gomock.Controller) *mock_botsfw.MockWebhookContext {
	translator := newTestTranslator(context.Background())
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().AppUserID().Return("u1").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
		if s := translator.Translate(key, args...); s != "" {
			return s
		}
		return key
	}).AnyTimes()
	whc.EXPECT().TranslateNoWarning(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
		if s := translator.TranslateNoWarning(key, args...); s != "" {
			return s
		}
		return key
	}).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("cmd-text").AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) (m botmsg.MessageFromBot) {
		m.Text = text
		return m
	}).AnyTimes()
	whc.EXPECT().NewMessageByCode(gomock.Any(), gomock.Any()).DoAndReturn(func(code string, _ ...any) (m botmsg.MessageFromBot) {
		m.Text = code
		return m
	}).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(text string, format botmsg.Format) (m botmsg.MessageFromBot, err error) {
		m.Text = text
		m.Format = format
		m.IsEdit = true
		return m, nil
	}).AnyTimes()
	return whc
}

// fakeBadTemplateTranslator returns an unparsable template for any key,
// using a unique locale code so common4all template cache is not polluted.
type fakeBadTemplateTranslator struct{}

func (fakeBadTemplateTranslator) Locale() i18n.Locale { return i18n.Locale{Code5: "xx-XX"} }
func (fakeBadTemplateTranslator) Translate(_ string, _ ...any) string {
	return "{{.Bad"
}
func (fakeBadTemplateTranslator) TranslateWithMap(_ string, _ map[string]string) string {
	return "{{.Bad"
}
func (fakeBadTemplateTranslator) TranslateNoWarning(_ string, _ ...any) string {
	return "{{.Bad"
}

func TestCallbackLinkToGroup(t *testing.T) {
	if got := CallbackLink.ToGroup("g1", false); got != groupCommandCode+"?id=g1" {
		t.Errorf("unexpected: %q", got)
	}
	if got := CallbackLink.ToGroup("g1", true); !strings.HasSuffix(got, "&edit=1") {
		t.Errorf("expected edit suffix, got %q", got)
	}
}

func TestGetWhoPaidInlineKeyboard(t *testing.T) {
	kb := getWhoPaidInlineKeyboard(newTestTranslator(context.Background()), "b1")
	if kb == nil || len(kb.InlineKeyboard) != 3 {
		t.Fatalf("expected 3 keyboard rows, got %v", kb)
	}
}

func TestGetGroupBillCardInlineKeyboard(t *testing.T) {
	bill := models4splitus.NewBillEntry("b1", nil)
	kb := getGroupBillCardInlineKeyboard(newTestTranslator(context.Background()), bill)
	if kb == nil || len(kb.InlineKeyboard) == 0 {
		t.Fatal("expected non-empty keyboard")
	}
}

func TestGetPrivateBillCardInlineKeyboard(t *testing.T) {
	bill := models4splitus.NewBillEntry("b1", nil)
	kb := getPrivateBillCardInlineKeyboard(newTestTranslator(context.Background()), "testbot", bill)
	if kb == nil || len(kb.InlineKeyboard) == 0 {
		t.Fatal("expected non-empty keyboard")
	}
}

func TestBillCardCallbackCommandData(t *testing.T) {
	if got := billCardCallbackCommandData("b1"); got != billCardCommandCode+"?bill=b1" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestWriteSplitInstructions(t *testing.T) {
	var buf bytes.Buffer
	writeSplitInstructions(&buf, "", "Alice")
	if !strings.Contains(buf.String(), "Alice") {
		t.Errorf("expected member name, got %q", buf.String())
	}
	buf.Reset()
	writeSplitInstructions(&buf, "12345", "Bob")
	if !strings.Contains(buf.String(), "tg://user?id=12345") {
		t.Errorf("expected tg user link, got %q", buf.String())
	}
}

func TestWriteSplitMembers(t *testing.T) {
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 1}, Owes: 100},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
	}
	var buf bytes.Buffer
	writeSplitMembers(&buf, members, "m1", "USD")
	out := buf.String()
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Bob") {
		t.Errorf("expected member names, got %q", out)
	}

	// totalShares == 0 path
	buf.Reset()
	writeSplitMembers(&buf, []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
	}, "other", "EUR")
	if buf.Len() == 0 {
		t.Error("expected output for zero-shares members")
	}
}

func TestAddEditSplitInlineKeyboardButtons(t *testing.T) {
	translator := newTestTranslator(context.Background())

	t.Run("multiple_members", func(t *testing.T) {
		kb := addEditSplitInlineKeyboardButtons(nil, translator, 2, "b1", "pfx?", "back")
		if len(kb) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(kb))
		}
	})

	t.Run("single_member", func(t *testing.T) {
		kb := addEditSplitInlineKeyboardButtons(nil, translator, 1, "b1", "pfx?", "back")
		if len(kb) != 1 {
			t.Fatalf("expected 1 row, got %d", len(kb))
		}
	})
}

func TestGetSplitParamsAndCurrentMember(t *testing.T) {
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice"}},
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "Bob"}},
	}
	cases := []struct {
		name       string
		query      string
		members    []*briefs4splitus.BillMemberBrief
		wantErr    bool
		wantMember string
		wantAdd    int
	}{
		{"no_members", "", nil, true, "", 0},
		{"no_m_param", "", members, false, "m1", 0},
		{"m_zero", "m=0", members, true, "", 0},
		{"move_up", "m=m2&move=up", members, false, "m1", 0},
		{"move_up_wrap", "m=m1&move=up", members, false, "m2", 0},
		{"move_down", "m=m1&move=down", members, false, "m2", 0},
		{"move_down_wrap", "m=m2&move=down", members, false, "m1", 0},
		{"move_unknown", "m=m1&move=sideways", members, true, "", 0},
		{"add", "m=m1&add=5", members, false, "m1", 5},
		{"add_invalid", "m=m1&add=abc", members, true, "", 0},
		{"member_no_extra", "m=m2", members, false, "m2", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := url.ParseQuery(c.query)
			if err != nil {
				t.Fatal(err)
			}
			member, add, err := getSplitParamsAndCurrentMember(q, c.members)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if member.ID != c.wantMember {
				t.Errorf("member ID: got %q, want %q", member.ID, c.wantMember)
			}
			if add != c.wantAdd {
				t.Errorf("add: got %d, want %d", add, c.wantAdd)
			}
		})
	}
}

func newBillWithMembers(id string, members []*briefs4splitus.BillMemberBrief, currency money.CurrencyCode) models4splitus.BillEntry {
	bill := models4splitus.NewBillEntry(id, nil)
	bill.Data.Name = "Lunch"
	bill.Data.Currency = currency
	bill.Data.AmountTotal = 1000
	bill.Data.Members = members
	return bill
}

func TestWriteBillMembersList(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)

	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "PaidAll", Shares: 1}, Paid: 1000}, // paid == total -> bold
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m2", Name: "PartPaid"}, Owes: 300, Paid: 200}, // owes & paid
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m3", Name: "OwesOnly"}, Owes: 500},            // owes only
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m4", Name: "PaidSome"}, Paid: 100},            // paid only
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m5", Name: "Plain"}},                          // default
	}

	t.Run("all_branches", func(t *testing.T) {
		bill := newBillWithMembers("b1", members, "USD")
		var buf bytes.Buffer
		writeBillMembersList(ctx, &buf, translator, bill, "")
		out := buf.String()
		for _, name := range []string{"PaidAll", "PartPaid", "OwesOnly", "PaidSome", "Plain"} {
			if !strings.Contains(out, name) {
				t.Errorf("expected %q in output", name)
			}
		}
		if !strings.Contains(out, "<b>") {
			t.Error("expected bold markup for member who paid in full")
		}
	})

	t.Run("selected_member", func(t *testing.T) {
		bill := newBillWithMembers("b2", members, "USD")
		var buf bytes.Buffer
		writeBillMembersList(ctx, &buf, translator, bill, "m2")
		if buf.Len() == 0 {
			t.Error("expected output")
		}
	})

	t.Run("zero_shares", func(t *testing.T) {
		bill := newBillWithMembers("b3", []*briefs4splitus.BillMemberBrief{
			{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "NoShares"}},
		}, "USD")
		var buf bytes.Buffer
		writeBillMembersList(ctx, &buf, translator, bill, "")
		if !strings.Contains(buf.String(), "NoShares") {
			t.Errorf("expected member name, got %q", buf.String())
		}
	})

	t.Run("render_error", func(t *testing.T) {
		bill := newBillWithMembers("b4", members, "USD")
		var buf bytes.Buffer
		writeBillMembersList(ctx, &buf, fakeBadTemplateTranslator{}, bill, "")
		// render fails on first member title template -> early return
	})
}

func TestWriteBillCardTitle(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)

	t.Run("no_currency", func(t *testing.T) {
		bill := newBillWithMembers("b1", nil, "")
		var buf bytes.Buffer
		if err := writeBillCardTitle(ctx, bill, "testbot", &buf, translator); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("expected output")
		}
	})

	t.Run("with_currency", func(t *testing.T) {
		bill := newBillWithMembers("b1", nil, "USD")
		var buf bytes.Buffer
		if err := writeBillCardTitle(ctx, bill, "testbot", &buf, translator); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGetBillCardMessageText(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)
	members := []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 1}},
	}

	cases := []struct {
		name        string
		members     []*briefs4splitus.BillMemberBrief
		showMembers bool
		footer      string
	}{
		{"members_footer", members, true, "footer-text"},
		{"members_no_footer", members, true, ""},
		{"no_members_footer", nil, true, "footer-text"},
		{"hide_members_footer", members, false, "footer-text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bill := newBillWithMembers("b1", c.members, "USD")
			text, err := getBillCardMessageText(ctx, "testbot", translator, bill, c.showMembers, c.footer)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if text == "" {
				t.Error("expected non-empty text")
			}
			if c.footer != "" && !strings.Contains(text, c.footer) {
				t.Errorf("expected footer in text, got %q", text)
			}
		})
	}
}

func TestSetMainMenuAndMenuCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)

	var m botmsg.MessageFromBot
	SetMainMenu(whc, &m)
	if m.Keyboard == nil {
		t.Error("expected keyboard to be set")
	}

	m, err := menuCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil || m.Text == "" {
		t.Error("expected text and keyboard")
	}
}

func TestStartInGroupAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)

	m, err := StartInGroupAction(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected greeting text")
	}
}

func TestSettleBillsCommandActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)

	if _, err := settleBillsCommand.Action(whc); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected not-implemented error, got: %v", err)
	}
	if _, err := settleBillsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb")); err == nil {
		t.Error("expected not-implemented error from callback action")
	}
}

func TestGroupCommandCallbackAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)

	m, err := groupCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?id=g1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected text")
	}
}

func TestGroupsAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("in_group", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil)
		m, err := groupsAction(whc, false, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(m.Text, "private chat") {
			t.Errorf("expected private-chat hint, got %q", m.Text)
		}
	})

	t.Run("is_in_group_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if _, err := groupsAction(whc, false, 0); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not_in_group_via_action", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		m, err := groupsCommand.Action(whc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Text == "" {
			t.Error("expected text")
		}
	})

	t.Run("callback_edit", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		if _, err := groupsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?edit=1")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStartInBotAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("no_params", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := StartInBotAction(whc, nil); err == nil {
			t.Fatal("expected ErrUnknownStartParam")
		}
	})

	t.Run("unknown_param", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := StartInBotAction(whc, []string{"something-else"}); err == nil {
			t.Fatal("expected ErrUnknownStartParam")
		}
	})

	t.Run("settle_group", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := StartInBotAction(whc, []string{SettleGroupAskForCounterpartyCommandCode, "space=s1"}); err == nil {
			t.Fatal("expected not-implemented error")
		}
	})

	t.Run("empty_bill_id", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		if _, err := StartInBotAction(whc, []string{"bill-"}); err == nil {
			t.Fatal("expected invalid bill parameter error")
		}
	})

	t.Run("bill_not_found", func(t *testing.T) {
		withMemDB(t)
		whc := newMockWhc(ctrl)
		if _, err := StartInBotAction(whc, []string{"bill-missing"}); err == nil {
			t.Fatal("expected not-found error")
		}
	})
}

func TestShowBillCard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bill := newBillWithMembers("b1", nil, "USD")

	t.Run("in_group", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil)
		m, err := ShowBillCard(whc, true, bill, "footer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Keyboard == nil {
			t.Error("expected group keyboard")
		}
	})

	t.Run("is_in_group_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest)
		if _, err := ShowBillCard(whc, false, bill, ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("private_chat", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		input := &fakeInlineInput{chat: fakeChat{}}
		whc.EXPECT().Input().Return(input).AnyTimes()
		m, err := ShowBillCard(whc, false, bill, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Keyboard == nil {
			t.Error("expected private keyboard")
		}
	})

	t.Run("nil_chat", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil)
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		m, err := ShowBillCard(whc, false, bill, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Keyboard == nil {
			t.Error("expected group keyboard for nil chat")
		}
	})
}

var errTest = errTestType{}

type errTestType struct{}

func (errTestType) Error() string { return "test error" }

var _ botsfw.WebhookContext = (*mock_botsfw.MockWebhookContext)(nil)
