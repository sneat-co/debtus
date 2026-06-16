package cmds4splitusbot

import (
	"context"
	"net/url"
	"strings"
	"testing"

	tgbotapi "github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"go.uber.org/mock/gomock"
)

func newTestContactusSpace(spaceID coretypes.SpaceID) dal4contactus.ContactusSpaceEntry {
	return dal4contactus.NewContactusSpaceEntry(spaceID)
}

func TestGetBillID_missingParam(t *testing.T) {
	u, _ := url.Parse("http://example.com/callback")
	_, err := GetBillID(u)
	if err == nil {
		t.Fatal("expected error when 'bill' param is missing")
	}
	if !strings.Contains(err.Error(), "bill") {
		t.Errorf("error should mention 'bill', got: %v", err)
	}
}

func TestGetBillID_withParam(t *testing.T) {
	u, _ := url.Parse("http://example.com/callback?bill=bill123")
	billID, err := GetBillID(u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if billID != "bill123" {
		t.Errorf("want 'bill123', got %q", billID)
	}
}

func TestCurrenciesInlineKeyboard(t *testing.T) {
	kb := currenciesInlineKeyboard("test-prefix")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	if len(kb.InlineKeyboard) == 0 {
		t.Fatal("expected keyboard rows")
	}
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "" {
				t.Error("expected non-empty CallbackData on button")
			}
		}
	}
}

func TestCurrenciesInlineKeyboard_withExtraRows(t *testing.T) {
	extra := []tgbotapi.InlineKeyboardButton{{Text: "extra", CallbackData: "extra-data"}}
	kb := currenciesInlineKeyboard("pfx", extra)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	last := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	if len(last) != 1 || last[0].Text != "extra" {
		t.Errorf("expected extra row at end, got %v", last)
	}
}

func TestOnStartCallbackInGroup(t *testing.T) {
	_, err := onStartCallbackInGroup(nil, dbo4spaceus.NewSpaceEntry("space1"))
	if err == nil {
		t.Fatal("expected error from onStartCallbackInGroup")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' in error, got: %v", err)
	}
}

func TestShowGroupMembers_withSpaceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
		return key
	}).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).Return("settings").AnyTimes()

	contactusSpace := newTestContactusSpace("space1")
	m, err := showGroupMembers(whc, "space1", contactusSpace, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Keyboard == nil {
		t.Error("expected keyboard to be set")
	}
}

func TestShowGroupMembers_emptySpaceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	// Pass a placeholder contactusSpace (with a non-empty ID) but empty spaceID
	// to trigger the early-return error path in showGroupMembers
	contactusSpace := newTestContactusSpace("placeholder")
	m, err := showGroupMembers(whc, "", contactusSpace, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "ERROR") {
		t.Errorf("expected ERROR in message text, got: %q", m.Text)
	}
}
