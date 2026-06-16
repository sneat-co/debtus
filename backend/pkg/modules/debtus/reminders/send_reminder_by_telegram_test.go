package reminders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/i18n"
)

func testReminder(id string) dbo4reminders.Reminder {
	return dbo4reminders.NewReminder(id, &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
}

func testTransfer() models4debtus.TransferEntry {
	return newValidTransfer("txid")
}

func overrideGetLocale(t *testing.T, locale i18n.Locale, err error) {
	t.Helper()
	orig := getLocale
	getLocale = func(_ context.Context, _ string, _ int64, _ string) (i18n.Locale, error) {
		return locale, err
	}
	t.Cleanup(func() { getLocale = orig })
}

func overrideGetBotSettingsByCode(t *testing.T, settings *botsfw.BotSettings, err error) {
	t.Helper()
	orig := getBotSettingsByCode
	getBotSettingsByCode = func(_ context.Context, _ string) (*botsfw.BotSettings, error) {
		return settings, err
	}
	t.Cleanup(func() { getBotSettingsByCode = orig })
}

func overrideNewTgBotAPIFromSettings(t *testing.T) {
	t.Helper()
	orig := newTgBotAPIFromSettings
	newTgBotAPIFromSettings = func(_ context.Context, token string) *tgbotapi.BotAPI {
		return tgbotapi.NewBotAPIWithClient(token, nil)
	}
	t.Cleanup(func() { newTgBotAPIFromSettings = orig })
}

func overrideTgBotAPISend(t *testing.T, msg tgbotapi.Message, err error) {
	t.Helper()
	orig := tgBotAPISend
	tgBotAPISend = func(_ *tgbotapi.BotAPI, _ tgbotapi.Sendable) (tgbotapi.Message, error) {
		return msg, err
	}
	t.Cleanup(func() { tgBotAPISend = orig })
}

func overrideReminderSent(t *testing.T) {
	t.Helper()
	orig := reminderSent
	reminderSent = func(_ context.Context, _, _, _ string) {}
	t.Cleanup(func() { reminderSent = orig })
}

func TestSendReminderByTelegram_ZeroChatID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for tgChatID==0")
		}
	}()
	_, _, _ = sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 0, "testbot")
}

func TestSendReminderByTelegram_EmptyBot_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty tgBot")
		}
	}()
	_, _, _ = sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "")
}

func TestSendReminderByTelegram_GetLocalError(t *testing.T) {
	overrideGetLocale(t, i18n.Locale{}, errors.New("locale error"))
	_, _, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err == nil {
		t.Error("expected error from GetLocale")
	}
}

func TestSendReminderByTelegram_GetReminderError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t) // reminder not seeded → GetReminderByID returns not-found
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, &botsfw.BotSettings{Token: "fake:TOKEN"}, nil)
	overrideNewTgBotAPIFromSettings(t)

	_, _, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err == nil {
		t.Error("expected error when reminder not in DB")
	}
}

func TestSendReminderByTelegram_BotSettingsError(t *testing.T) {
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, nil, errors.New("bot not found"))
	_, _, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err == nil {
		t.Error("expected error from GetBotSettingsByCode")
	}
}

func overrideDelaySetChatIsForbiddenFn(t *testing.T, err error) {
	t.Helper()
	orig := delaySetChatIsForbiddenFn
	delaySetChatIsForbiddenFn = func(_ context.Context, _ string, _ int64, _ time.Time) error {
		return err
	}
	t.Cleanup(func() { delaySetChatIsForbiddenFn = orig })
}

func overrideSetReminderIsSentInTx(t *testing.T, err error) {
	t.Helper()
	orig := setReminderIsSentInTx
	setReminderIsSentInTx = func(_ context.Context, _ dal.ReadwriteTransaction, _ dbo4reminders.Reminder, _ time.Time, _ int64, _, _, _ string) error {
		return err
	}
	t.Cleanup(func() { setReminderIsSentInTx = orig })
}

func overrideDelaySetReminderIsSent(t *testing.T, err error) {
	t.Helper()
	orig := delaySetReminderIsSent
	delaySetReminderIsSent = func(_ context.Context, _ string, _ time.Time, _ int64, _, _, _ string) error {
		return err
	}
	t.Cleanup(func() { delaySetReminderIsSent = orig })
}

func TestSendReminderByTelegram_SendForbidden(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "r1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, &botsfw.BotSettings{Token: "fake:TOKEN"}, nil)
	overrideNewTgBotAPIFromSettings(t)
	overrideTgBotAPISend(t, tgbotapi.Message{}, tgbotapi.ErrAPIForbidden{})
	overrideDelaySetChatIsForbiddenFn(t, nil)

	_, channelDisabled, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !channelDisabled {
		t.Error("expected channelDisabledByUser=true for ErrAPIForbidden")
	}
}

func TestSendReminderByTelegram_SendForbidden_DelayError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "r1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, &botsfw.BotSettings{Token: "fake:TOKEN"}, nil)
	overrideNewTgBotAPIFromSettings(t)
	overrideTgBotAPISend(t, tgbotapi.Message{}, tgbotapi.ErrAPIForbidden{})
	overrideDelaySetChatIsForbiddenFn(t, errors.New("delay error"))

	_, channelDisabled, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !channelDisabled {
		t.Error("expected channelDisabledByUser=true even when delay fails")
	}
}

func TestSendReminderByTelegram_SetReminderSentError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "r1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, &botsfw.BotSettings{Token: "fake:TOKEN"}, nil)
	overrideNewTgBotAPIFromSettings(t)
	overrideTgBotAPISend(t, tgbotapi.Message{MessageID: 42}, nil)
	overrideSetReminderIsSentInTx(t, errors.New("db error"))
	overrideDelaySetReminderIsSent(t, nil)
	overrideReminderSent(t)

	sent, _, _ := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if !sent {
		t.Error("expected sent=true even when marking as sent fails")
	}
}

func TestSendReminderByTelegram_SendSuccess(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "r1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	overrideGetLocale(t, i18n.LocaleEnUS, nil)
	overrideGetBotSettingsByCode(t, &botsfw.BotSettings{Token: "fake:TOKEN"}, nil)
	overrideNewTgBotAPIFromSettings(t)
	overrideTgBotAPISend(t, tgbotapi.Message{MessageID: 42}, nil)
	overrideReminderSent(t)

	sent, channelDisabled, err := sendReminderByTelegram(context.Background(), testTransfer(), testReminder("r1"), 12345, "testbot")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected sent=true")
	}
	if channelDisabled {
		t.Error("expected channelDisabledByUser=false")
	}
}
