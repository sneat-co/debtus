package debtusviberbots

import (
	"context"
	"testing"
)

func TestBots(t *testing.T) {
	result := Bots(context.Background())
	// The function body is entirely commented out — always returns empty BotSettingsBy.
	if result.ByCode != nil {
		t.Error("expected nil ByCode for stub Bots()")
	}
}
