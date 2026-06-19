package splitusbot

import (
	"testing"

	"github.com/sneat-co/sneat-bots/pkg/bots/botinitparams"
)

func TestGetProfile(t *testing.T) {
	botProfile = nil // reset singleton
	p := GetProfile(botinitparams.BotProfileParams{})
	if p == nil {
		t.Fatal("expected non-nil profile")
	}
	// second call returns cached
	p2 := GetProfile(botinitparams.BotProfileParams{})
	if p2 == nil {
		t.Fatal("expected non-nil profile on second call")
	}
}
