package facade4debtusbot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func GetLocale(ctx context.Context, botID string, tgChatIntID int64, userID string) (locale i18n.Locale, err error) {
	chatID := botsfwmodels.NewChatID(botID, strconv.FormatInt(tgChatIntID, 10))
	//var tgChatEntity botsfwtgmodels.ChatEntity
	//tgChatBaseData := botsfwtgmodels.NewTelegramChatBaseData()
	//chatID, new(models.DebtusTelegramChatData)
	key := dal.NewKeyWithID(botsfwtgmodels.TgChatCollection, chatID)
	var tgChat = record.NewDataWithID[string, *anybot.SneatAppTgChatDbo](chatID, key, new(anybot.SneatAppTgChatDbo))
	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return
	}
	if err = db.Get(ctx, tgChat.Record); err != nil {
		err = fmt.Errorf("failed to get TgChat entity by ID=%v: %w", tgChat.ID, err)
		logus.Debugf(ctx, err.Error())
		return
	}
	tgChatPreferredLanguage := tgChat.Data.BaseTgChatData().PreferredLanguage
	if tgChatPreferredLanguage == "" {
		if userID == "" && tgChat.Data.AppUserID != "" {
			userID = tgChat.Data.BaseTgChatData().AppUserID
		}
		if userID != "" {
			var user dbo4userus.UserEntry
			if user, err = dal4userus.GetUserByID(ctx, db, userID); err != nil {
				logus.Errorf(ctx, fmt.Errorf("failed to get user by userID=%s: %w", userID, err).Error())
				return locale, err
			}
			tgChatPreferredLanguage = user.Data.PreferredLocale
		}
		if tgChatPreferredLanguage == "" {
			tgChatPreferredLanguage = i18n.LocaleCodeEnUS
			logus.Warningf(ctx, "tgChat.PreferredLanguage == '' && user.PreferredLanguage == '', set to %v", i18n.LocaleCodeEnUS)
		}
	}
	locale = i18n.LocalesByCode5[tgChatPreferredLanguage]
	return
}
