package delayed4debtus

import (
	"context"
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

// buildSendReceiptInputs returns a receipt+transfer+tgChat ready for sendReceiptToTelegramChatReal.
func buildSendReceiptInputs(receiptID, transferID, botID, tgUserID string) (models4debtus.ReceiptEntry, models4debtus.TransferEntry, anybot.SneatAppTgChatEntry) {
	receipt := models4debtus.NewReceipt(receiptID, &models4debtus.ReceiptDbo{
		Status:     models4debtus.ReceiptStatusSending,
		TransferID: transferID,
	})
	transfer := models4debtus.NewTransfer(transferID, &models4debtus.TransferData{
		CreatorUserID: "u-creator",
		FromJson:      `{"userID":"u-creator","contactName":"Creator Name"}`,
		ToJson:        `{"userID":"u-cp","contactName":"Counterparty Name"}`,
	})
	tgChat := newTestTgChatEntry(botID, tgUserID)
	tgChat.Data.BotUserIDs = []string{tgUserID}
	return receipt, transfer, tgChat
}

func TestSendReceiptToTelegramChatReal(t *testing.T) {
	ctx := context.Background()
	const (
		receiptID  = "r-real"
		transferID = "t-real"
		botID      = "bot1"
		tgUserID   = "12345"
	)

	t.Run("getBotApi_error", func(t *testing.T) {
		_ = setupSendReceiptTest(t)
		receipt, transfer, tgChat := buildSendReceiptInputs(receiptID, transferID, botID, tgUserID)
		stubGetTelegramBotApi(t, nil, errors.New("no bot api"))

		if err := sendReceiptToTelegramChatReal(ctx, receipt, transfer, tgChat); err == nil {
			t.Fatal("expected error from getTelegramBotApiFn, got nil")
		}
	})

	t.Run("send_error", func(t *testing.T) {
		_ = setupSendReceiptTest(t)
		receipt, transfer, tgChat := buildSendReceiptInputs(receiptID, transferID, botID, tgUserID)
		// Telegram returns ok:false → tgBotApi.Send errors.
		stubGetTelegramBotApi(t, fakeBotAPI(fakeRoundTripper{
			responseBody: `{"ok":false,"error_code":400,"description":"Bad Request"}`,
			statusCode:   400,
		}), nil)

		if err := sendReceiptToTelegramChatReal(ctx, receipt, transfer, tgChat); err == nil {
			t.Fatal("expected error from tgBotApi.Send, got nil")
		}
	})

	t.Run("success_updates_status", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		receipt, transfer, tgChat := buildSendReceiptInputs(receiptID, transferID, botID, tgUserID)
		// Seed receipt so updateReceiptStatus's transaction can load+update it.
		seedRecords(t, db, receipt.Record)
		stubGetTelegramBotApi(t, fakeBotAPI(fakeRoundTripper{}), nil)

		if err := sendReceiptToTelegramChatReal(ctx, receipt, transfer, tgChat); err != nil {
			t.Fatalf("sendReceiptToTelegramChatReal() error = %v", err)
		}
		// Confirm status moved to sent.
		got := models4debtus.NewReceipt(receiptID, nil)
		if err := db.Get(ctx, got.Record); err != nil {
			t.Fatalf("failed to reload receipt: %v", err)
		}
		if got.Data.Status != models4debtus.ReceiptStatusSent {
			t.Errorf("receipt status = %v, want %v", got.Data.Status, models4debtus.ReceiptStatusSent)
		}
	})
}

var _ = tgbotapi.NewEditMessageText
