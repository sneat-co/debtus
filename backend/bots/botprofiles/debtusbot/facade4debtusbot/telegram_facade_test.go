package facade4debtusbot

import (
	"context"
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/i18n"
)

func TestGetLocale(t *testing.T) {
	ctx := context.Background()
	const (
		botID       = "bot1"
		tgChatIntID = int64(123)
	)

	seedTgChat := func(t *testing.T, db dal.DB, preferredLanguage, appUserID string) {
		t.Helper()
		chatID := botsfwmodels.NewChatID(botID, "123")
		key := dal.NewKeyWithID(botsfwtgmodels.TgChatCollection, chatID)
		data := new(anybot.SneatAppTgChatDbo)
		data.PreferredLanguage = preferredLanguage
		data.AppUserID = appUserID
		tgChat := record.NewDataWithID(chatID, key, data)
		if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, tgChat.Record)
		}); err != nil {
			t.Fatalf("failed to seed tg chat: %v", err)
		}
	}

	seedUser := func(t *testing.T, db dal.DB, userID, preferredLocale string) {
		t.Helper()
		user := dbo4userus.NewUserEntry(userID)
		user.Data.PreferredLocale = preferredLocale
		if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, user.Record)
		}); err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
	}

	t.Run("returns_error_when_GetSneatDB_fails", func(t *testing.T) {
		origGetSneatDB := facade.GetSneatDB
		dbErr := errors.New("db connection error")
		facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
			return nil, dbErr
		}
		t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })
		_, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err == nil {
			t.Fatal("expected error when GetSneatDB fails, got nil")
		}
		if !errors.Is(err, dbErr) {
			t.Errorf("expected dbErr, got: %v", err)
		}
	})

	t.Run("returns_error_when_chat_not_found", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		_, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err == nil {
			t.Fatal("expected error when TgChat not found, got nil")
		}
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})

	t.Run("returns_chat_preferred_language", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedTgChat(t, db, i18n.LocaleCodeRuRU, "")
		locale, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err != nil {
			t.Fatalf("GetLocale() returned error: %v", err)
		}
		if locale.Code5 != i18n.LocaleCodeRuRU {
			t.Errorf("locale.Code5 = %v, want %v", locale.Code5, i18n.LocaleCodeRuRU)
		}
	})

	t.Run("returns_error_when_GetUserByID_fails", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedTgChat(t, db, "", "u1")
		userErr := errors.New("user fetch error")
		origGetUserByID := dal4userus.GetUserByID
		dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
			return dbo4userus.UserEntry{}, userErr
		}
		t.Cleanup(func() { dal4userus.GetUserByID = origGetUserByID })
		_, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err == nil {
			t.Fatal("expected error when GetUserByID fails, got nil")
		}
		if !errors.Is(err, userErr) {
			t.Errorf("expected userErr, got: %v", err)
		}
	})

	t.Run("falls_back_to_user_preferred_locale", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedTgChat(t, db, "", "u1")
		seedUser(t, db, "u1", i18n.LocaleCodeRuRU)
		locale, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err != nil {
			t.Fatalf("GetLocale() returned error: %v", err)
		}
		if locale.Code5 != i18n.LocaleCodeRuRU {
			t.Errorf("locale.Code5 = %v, want %v", locale.Code5, i18n.LocaleCodeRuRU)
		}
	})

	t.Run("defaults_to_en_US_when_no_language_known", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedTgChat(t, db, "", "")
		locale, err := GetLocale(ctx, botID, tgChatIntID, "")
		if err != nil {
			t.Fatalf("GetLocale() returned error: %v", err)
		}
		if locale.Code5 != i18n.LocaleCodeEnUS {
			t.Errorf("locale.Code5 = %v, want %v", locale.Code5, i18n.LocaleCodeEnUS)
		}
	})
}
