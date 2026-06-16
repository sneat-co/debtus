package cmds4splitusbot

import (
	"context"
	"errors"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/const4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func delayUpdateBillCardOnUserJoin(ctx context.Context, billID string, message string) error {
	if err := delayUpdateBillCards.EnqueueWork(
		ctx,
		delaying.With(const4splitus.QueueSplitus, "update-bill-cards", 0),
		billID,
		message,
	); err != nil {
		logus.Errorf(ctx, "Failed to queue update of bill cards: %v", err)
	}
	return nil
}

func delayedUpdateBillCards(ctx context.Context, billID string, footer string) error {
	logus.Debugf(ctx, "delayedUpdateBillCards(billID=%s)", billID)
	if bill, err := facade4splitus.GetBillByID(ctx, nil, billID); err != nil {
		return err
	} else {
		for _, tgChatMessageID := range bill.Data.TgChatMessageIDs {
			if err = delayUpdateBillTgChatCard.EnqueueWork(ctx, delaying.With(const4splitus.QueueSplitus, "update-bill-tg-chat-card", 0), billID, tgChatMessageID, footer); err != nil {
				logus.Errorf(ctx, "Failed to queue updated for %v: %v", tgChatMessageID, err)
				return err
			}
		}
	}
	return nil
}

func delayedUpdateBillTgChartCard(ctx context.Context, billID string, tgChatMessageID, footer string) (err error) {
	logus.Debugf(ctx, "delayedUpdateBillTgChartCard(billID=%s, tgChatMessageID=%v)", billID, tgChatMessageID)
	var bill models4splitus.BillEntry
	if bill, err = facade4splitus.GetBillByID(ctx, nil, billID); err != nil {
		return err
	} else {
		ids := strings.Split(tgChatMessageID, "@")
		inlineMessageID, botCode, localeCode5 := ids[0], ids[1], ids[2]
		translator := i18n.NewSingleMapTranslator(i18n.GetLocaleByCode5(localeCode5), i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS))

		editMessage := tgbotapi.NewEditMessageText(0, 0, inlineMessageID, "")
		editMessage.ParseMode = "HTML"
		editMessage.DisableWebPagePreview = true

		if err = updateInlineBillCardMessage(ctx, translator, true, editMessage, bill, botCode, footer); err != nil {
			return err
		}
		botSettings, settingsErr := botsettings.GetBotSettingsByCode(ctx, botCode)
		if settingsErr != nil {
			logus.Errorf(ctx, "No bot settings for bot: "+botCode)
			return nil
		}

		tgApi := tgbotapi.NewBotAPIWithClient(botSettings.Token, dal4debtus.Default.HttpClient(ctx))
		if _, err = tgApi.Send(editMessage); err != nil {
			logus.Errorf(ctx, "Failed to send message to Telegram: %v", err)
			return err
		}
		return nil
	}
}

func updateInlineBillCardMessage(ctx context.Context, translator i18n.SingleLocaleTranslator, isGroupChat bool, editedMessage *tgbotapi.EditMessageTextConfig, bill models4splitus.BillEntry, botCode string, footer string) (err error) {
	if bill.ID == "" {
		return errors.New("updateInlineBillCardMessage: bill.ID is empty string")
	}
	if bill.Data == nil {
		return errors.New("updateInlineBillCardMessage: bill.Data == nil")
	}

	if editedMessage.Text, err = getBillCardMessageText(ctx, botCode, translator, bill, true, footer); err != nil {
		return
	}
	if isGroupChat {
		editedMessage.ReplyMarkup = getPublicBillCardInlineKeyboard(translator, botCode, bill.ID)
	} else {
		editedMessage.ReplyMarkup = getPrivateBillCardInlineKeyboard(translator, botCode, bill)
	}

	return
}

func getPublicBillCardInlineKeyboard(translator i18n.SingleLocaleTranslator, botCode string, billID string) *tgbotapi.InlineKeyboardMarkup {
	goToBotLink := func(command botsfw.CommandCode) string {
		return bothelper.StartTelegramBotUrl(botCode, command, "id="+billID)
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{
				Text: translator.Translate(trans.BUTTON_TEXT_JOIN),
				URL:  goToBotLink(joinBillCommandCode),
			},
		},
		[]tgbotapi.InlineKeyboardButton{
			{
				Text: translator.Translate(trans.BUTTON_TEXT_EDIT_BILL),
				URL:  goToBotLink(editBillCommandCode),
			},
			{
				Text:         translator.Translate(trans.BUTTON_TEXT_DUE, translator.Translate(trans.NOT_SET)),
				CallbackData: billCallbackCommandData(setBillDueDateCommandCode, billID),
			},
		},
	)
	return keyboard
}
