package debtusfbmbots

import (
	"context"
	"testing"
)

func TestBots(t *testing.T) {
	result := Bots(context.Background())
	// The function body is entirely commented out — it always returns an empty BotSettingsBy.
	if result.ByCode != nil {
		t.Error("expected nil ByCode for stub Bots()")
	}
}
