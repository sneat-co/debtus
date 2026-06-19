package cmds4debtus

import (
	"testing"

	"github.com/sneat-co/sneat-bots/pkg/bots/botinitparams"
)

func TestCreateDebtusBotProfile(t *testing.T) {
	p := CreateDebtusBotProfile(nil)
	if p == nil {
		t.Fatal("expected non-nil BotProfile")
	}
}

func TestGetProfile_createsAndCaches(t *testing.T) {
	// Reset cached profile so we test the creation path.
	botProfile = nil
	p1 := GetProfile(botinitparams.BotProfileParams{ErrFooterText: nil})
	if p1 == nil {
		t.Fatal("expected non-nil BotProfile from GetProfile")
	}
	// Second call should return cached value (botProfile != nil branch).
	p2 := GetProfile(botinitparams.BotProfileParams{ErrFooterText: nil})
	if p2 != p1 {
		t.Error("expected GetProfile to return cached profile on second call")
	}
}
