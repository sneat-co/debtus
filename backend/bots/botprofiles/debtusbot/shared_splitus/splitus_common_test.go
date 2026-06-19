package shared_splitus

import (
	"net/url"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
)

func TestGetSplitusSpaceEntryByCallbackUrl(t *testing.T) {
	_, err := GetSplitusSpaceEntryByCallbackUrl(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "func GetSplitusSpaceEntryByCallbackUrl() is not implemented yet" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewSplitusSpaceAction(t *testing.T) {
	called := false
	action := NewSplitusSpaceAction(func(botsfw.WebhookContext, models4splitus.SplitusSpaceEntry) (botmsg.MessageFromBot, error) {
		called = true
		return botmsg.MessageFromBot{}, nil
	})
	_, err := action(nil)
	if err == nil {
		t.Fatal("expected error from unimplemented GetSplitusSpaceEntryByCallbackUrl")
	}
	if called {
		t.Error("inner action f must not be called when lookup errors")
	}
}

func TestNewSplitusSpaceCallbackAction(t *testing.T) {
	called := false
	action := NewSplitusSpaceCallbackAction(func(botsfw.WebhookContext, *url.URL, models4splitus.SplitusSpaceEntry) (botmsg.MessageFromBot, error) {
		called = true
		return botmsg.MessageFromBot{}, nil
	})
	_, err := action(nil, nil)
	if err == nil {
		t.Fatal("expected error from unimplemented GetSplitusSpaceEntryByCallbackUrl")
	}
	if called {
		t.Error("inner action f must not be called when lookup errors")
	}
}
