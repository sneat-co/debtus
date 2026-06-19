package delayed4debtus

import (
	"context"
	"errors"
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func GetTelegramChatByUserID(ctx context.Context, userID string) (entityID string, chat botsfwtgmodels.TgChatData, err error) {
	tgChatQuery := dal.From(dal.NewRootCollectionRef(botsfwtgmodels.TgChatCollection, "")).
		NewQuery().
		WhereField("appUserID", dal.Equal, userID).
		OrderBy(dal.DescendingField("dtUpdated")).
		Limit(1).
		SelectIntoRecord(debtusbot.NewDebtusTelegramChatRecord)

	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return
	}

	var tgChatRecords []dal.Record
	if tgChatRecords, err = dal.ExecuteQueryAndReadAllToRecords(ctx, tgChatQuery, db); err != nil {
		err = fmt.Errorf("failed to load telegram chat by app user id=%v: %w", userID, err)
		return
	}
	switch len(tgChatRecords) {
	case tgChatQuery.Limit():
		entityID = fmt.Sprintf("%v", tgChatRecords[0].Key().ID)
		tgChatBase := tgChatRecords[0].Data().(anybot.SneatAppTgChatDbo).TgChatBaseData
		chat = &tgChatBase
		return
	case 0:
		err = fmt.Errorf("%w: telegram chat not found by userID=%s:%T", dal.ErrRecordNotFound, userID, userID)
		return
	default:
		err = fmt.Errorf("%w: too many telegram chats found by userID=%s:%T: %d", dal.ErrRecordNotFound, userID, userID, len(tgChatRecords))
		return
	}
}

// func getTranslatorAndTgChatID(ctx context.Context, userID int64) (translator i18n.SingleLocaleTranslator, tgChatID int64, err error) {
// 	var (
// 		//transfer models.TransferEntry
// 		user models.AppUserOBSOLETE
// 	)
// 	if user, err = dal4userus.GetUserByID(c, userID); err != nil {
// 		return
// 	}
// 	if user.TelegramUserID == 0 {
// 		err = errors.New("user.TelegramUserID == 0")
// 		return
// 	}
// 	var tgChat models.DebtusTelegramChat
// 	if tgChat, err = dal4debtus.TgChat.GetTgChatByID(c, user.TelegramUserID); err != nil {
// 		return
// 	}
// 	localeCode := tgChat.PreferredLanguage
// 	if localeCode == "" {
// 		localeCode = user.GetPreferredLocale()
// 	}
// 	if translator, err = getTranslator(ctx, localeCode); err != nil {
// 		return
// 	}
// 	return
// }

func getTranslator(ctx context.Context, localeCode string) (translator i18n.SingleLocaleTranslator, err error) {
	logus.Debugf(ctx, "getTranslator(localeCode=%v)", localeCode)
	locale, ok := i18n.LocalesByCode5[localeCode]
	if !ok {
		locale = i18n.LocalesByCode5[i18n.LocaleCodeEnUK]
	}
	return i18n.NewSingleMapTranslator(locale, i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS)), nil
}

func editTgMessageText(ctx context.Context, tgBotID string, tgChatID int64, tgMsgID int, text string) (err error) {
	msg := tgbotapi.NewEditMessageText(tgChatID, tgMsgID, "", text)
	botSettings, err := botsettings.GetBotSettingsByCode(ctx, tgBotID)
	if err != nil {
		return fmt.Errorf("bot settings not found by tgChat.BotID=%v: %w", tgBotID, err)
	}
	if err = sendToTelegramFn(ctx, msg, *botSettings); err != nil {
		return
	}
	return
}

func getTelegramBotApiByBotCode(ctx context.Context, code string) (*tgbotapi.BotAPI, error) {
	botSettings, err := botsettings.GetBotSettingsByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return tgbotapi.NewBotAPIWithClient(botSettings.Token, dal4debtus.Default.HttpClient(ctx)), nil
}

func sendToTelegram(ctx context.Context, msg tgbotapi.Sendable, botSettings botsfw.BotSettings) (err error) { // TODO: Merge with same in API package
	tgApi := tgbotapi.NewBotAPIWithClient(botSettings.Token, dal4debtus.Default.HttpClient(ctx))
	if _, err = tgApi.Send(msg); err != nil {
		return
	}
	return
}

var errReceiptStatusIsNotCreated = errors.New("receipt is not in 'created' status")

var editTgMessageTextFn = editTgMessageText
var getTelegramBotApiFn = getTelegramBotApiByBotCode
var sendToTelegramFn = sendToTelegram
var getTelegramChatByUserIDFn = GetTelegramChatByUserID

func updateReceiptStatus(ctx context.Context, tx dal.ReadwriteTransaction, receiptID string, expectedCurrentStatus, newStatus string) (receipt models4debtus.ReceiptEntry, err error) {

	if err = func() (err error) {
		if receipt, err = dal4debtus.Default.Receipt.GetReceiptByID(ctx, tx, receiptID); err != nil {
			return
		}
		if receipt.Data.Status != expectedCurrentStatus {
			return errReceiptStatusIsNotCreated
		}
		receipt.Data.Status = newStatus
		if err = tx.Set(ctx, receipt.Record); err != nil {
			return
		}
		return
	}(); err != nil {
		err = fmt.Errorf("failed to update receipt status from %v to %v: %w", expectedCurrentStatus, newStatus, err)
	}
	return
}
