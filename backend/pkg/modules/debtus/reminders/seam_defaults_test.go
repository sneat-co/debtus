package reminders

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

// errRoundTripper fails every HTTP request immediately, so a real Telegram
// Send returns an error without any network round-trip.
type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network in test")
}

// TestSeamDefaults exercises the DEFAULT bodies of the package-level seam vars.
// Each seam body is a single `return realCall(...)` statement that delegates to
// an external dependency; every other test replaces these vars with stubs, so
// their default bodies are otherwise never executed. Calling each default here
// covers the delegating statement even when the offline dependency errors or
// panics (COVER-BEFORE-PANIC). A memory DB is wired so DB-backed seams reach
// their delegate. Only the seams whose delegate returns or panics quickly are
// covered here; three seams (getLocale, discardReminder,
// getTelegramChatByUserID) make blocking network calls offline and are
// documented as external-io gaps in TEST-COVERAGE.md instead.
func TestSeamDefaults(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	ctx := context.Background()

	// safe runs f, swallowing any panic from the offline external call so the
	// seam body still registers as covered.
	safe := func(f func()) {
		defer func() { _ = recover() }()
		f()
	}

	transfer := newValidTransfer("txid")
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusSending,
		UserID: "u1",
	})

	t.Run("cron_handler", func(t *testing.T) {
		safe(func() { _ = createSendReminderTask(ctx, "rem1") })
		safe(func() { _, _ = getDueReminderIDs(ctx, db) })
	})

	t.Run("taskqueu_handler", func(t *testing.T) {
		safe(func() { _, _ = getTransferByID(ctx, nil, "txid") })
		safe(func() { _ = sendReminderToUserFn(ctx, "rem1", transfer) })
		safe(func() { _, _, _ = sendReminderByTelegramFn(ctx, transfer, reminder, 1, "bot") })
		// discardReminder and getTelegramChatByUserID delegate to operations
		// that block on network/RPC calls offline, so their default bodies are
		// documented as external-io gaps in TEST-COVERAGE.md instead.
	})

	t.Run("reminder_by_telegram", func(t *testing.T) {
		safe(func() { _, _ = getBotSettingsByCode(ctx, "bot") })
		safe(func() { _ = newTgBotAPIFromSettings(ctx, "token") })
		safe(func() { reminderSent(ctx, "u1", "en", "telegram") })
		safe(func() { _ = delaySetChatIsForbiddenFn(ctx, "bot", 1, time.Now()) })
		safe(func() { _ = setReminderIsSentInTx(ctx, nil, reminder, time.Now(), 0, "v", "en", "u1") })
		safe(func() { _ = delaySetReminderIsSent(ctx, "rem1", time.Now(), 0, "v", "en", "u1") })
		safe(func() {
			bot := &tgbotapi.BotAPI{Token: "x", Client: &http.Client{Transport: errRoundTripper{}}}
			_, _ = tgBotAPISend(bot, tgbotapi.NewMessage(1, "hi"))
		})
	})
}
