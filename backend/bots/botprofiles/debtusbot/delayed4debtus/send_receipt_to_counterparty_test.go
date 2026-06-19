package delayed4debtus

import (
	"context"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botsdal"
	"github.com/bots-go-framework/bots-fw/botsfwconst"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/strongo/delaying"
)

func setupSendReceiptTest(t *testing.T) dal.DB {
	t.Helper()
	db := sneattesting.SetupMemoryDB(t)

	originalReceiptDal := dal4debtus.Default.Receipt
	dal4debtus.Default.Receipt = debtusdal.NewReceiptDal()
	t.Cleanup(func() { dal4debtus.Default.Receipt = originalReceiptDal })

	originalOnReceiptSendFail := delayer4debtus.OnReceiptSendFail
	originalOnReceiptSentSuccess := delayer4debtus.OnReceiptSentSuccess
	delayer4debtus.OnReceiptSendFail = delaying.VoidWithLog("OnReceiptSendFail", DelayedOnReceiptSendFail)
	delayer4debtus.OnReceiptSentSuccess = delaying.VoidWithLog("OnReceiptSentSuccess", DelayedOnReceiptSentSuccess)
	t.Cleanup(func() {
		delayer4debtus.OnReceiptSendFail = originalOnReceiptSendFail
		delayer4debtus.OnReceiptSentSuccess = originalOnReceiptSentSuccess
	})
	return db
}

func stubSendReceiptToTelegramChat(t *testing.T, result error) *int {
	t.Helper()
	calls := new(int)
	original := sendReceiptToTelegramChat
	sendReceiptToTelegramChat = func(ctx context.Context, receipt models4debtus.ReceiptEntry, transfer models4debtus.TransferEntry, tgChat anybot.SneatAppTgChatEntry) error {
		*calls++
		return result
	}
	t.Cleanup(func() { sendReceiptToTelegramChat = original })
	return calls
}

func newTestTgChatEntry(botID, chatID string) anybot.SneatAppTgChatEntry {
	key := botsdal.NewBotChatKey(botsfwconst.PlatformTelegram, botID, chatID)
	data := new(anybot.SneatAppTgChatDbo)
	return anybot.SneatAppTgChatEntry{
		RecordWithID: record.NewWithID(chatID, key, data),
		Data:         data,
	}
}

func TestDelayedSendReceiptToCounterpartyByTelegram(t *testing.T) {
	ctx := context.Background()

	const (
		receiptID     = "r1"
		transferID    = "t1"
		creatorUserID = "u-creator"
		cpUserID      = "u-cp"
		botID         = "bot1"
		tgUserID      = "111"
	)

	newReceiptDbo := func(status string) *models4debtus.ReceiptDbo {
		return &models4debtus.ReceiptDbo{
			Status:             status,
			TransferID:         transferID,
			CreatorUserID:      creatorUserID,
			CounterpartyUserID: cpUserID,
		}
	}
	newTransferData := func() *models4debtus.TransferData {
		return &models4debtus.TransferData{
			CreatorUserID: creatorUserID,
			FromJson:      `{"userID":"` + creatorUserID + `","tgBotID":"` + botID + `","tgChatID":123}`,
			ToJson:        `{"userID":"` + cpUserID + `","contactName":"Counterparty Name"}`,
		}
	}

	seed := func(t *testing.T, db dal.DB, records ...dal.Record) {
		t.Helper()
		if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			for _, r := range records {
				if err := tx.Set(ctx, r); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.Fatalf("failed to seed records: %v", err)
		}
	}

	newCpUserRecord := func() dal.Record {
		user := dbo4userus.NewUserEntry(cpUserID)
		user.Data.Accounts = []string{"telegram:" + botID + ":" + tgUserID}
		return user.Record
	}

	getReceiptStatus := func(t *testing.T, db dal.DB) string {
		t.Helper()
		receipt := models4debtus.NewReceipt(receiptID, nil)
		if err := db.Get(ctx, receipt.Record); err != nil {
			t.Fatalf("failed to get receipt: %v", err)
		}
		return receipt.Data.Status
	}

	t.Run("stops_when_receipt_not_in_created_status", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, nil)
		seed(t, db, models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusSent)).Record)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if got := getReceiptStatus(t, db); got != models4debtus.ReceiptStatusSent {
			t.Errorf("receipt status = %v, want unchanged %v", got, models4debtus.ReceiptStatusSent)
		}
		if *calls != 0 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 0", *calls)
		}
	})

	t.Run("stops_when_transfer_not_found", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, nil)
		seed(t, db, models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if *calls != 0 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 0", *calls)
		}
	})

	t.Run("skips_account_when_tg_chat_record_not_found", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, nil)
		seed(t, db,
			models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record,
			models4debtus.NewTransfer(transferID, newTransferData()).Record,
			newCpUserRecord(),
		)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if *calls != 0 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 0", *calls)
		}
	})

	t.Run("skips_chat_already_marked_forbidden", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, nil)
		tgChat := newTestTgChatEntry(botID, tgUserID)
		tgChat.Data.DtForbiddenLast = time.Now()
		seed(t, db,
			models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record,
			models4debtus.NewTransfer(transferID, newTransferData()).Record,
			newCpUserRecord(),
			tgChat.Record,
		)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if *calls != 0 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 0", *calls)
		}
	})

	t.Run("sends_receipt_to_active_chat", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, nil)
		seed(t, db,
			models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record,
			models4debtus.NewTransfer(transferID, newTransferData()).Record,
			newCpUserRecord(),
			newTestTgChatEntry(botID, tgUserID).Record,
		)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if *calls != 1 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 1", *calls)
		}
	})

	t.Run("marks_chat_forbidden_when_telegram_api_forbids", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, tgbotapi.ErrAPIForbidden{})
		seed(t, db,
			models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record,
			models4debtus.NewTransfer(transferID, newTransferData()).Record,
			newCpUserRecord(),
			newTestTgChatEntry(botID, tgUserID).Record,
		)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if *calls != 1 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 1", *calls)
		}
		tgChat := newTestTgChatEntry(botID, tgUserID)
		if err := db.Get(ctx, tgChat.Record); err != nil {
			t.Fatalf("failed to get tg chat: %v", err)
		}
		if tgChat.Data.DtForbiddenLast.IsZero() {
			t.Error("tgChat.Data.DtForbiddenLast is zero, want it set to time of failure")
		}
	})

	t.Run("returns_error_on_generic_send_failure", func(t *testing.T) {
		db := setupSendReceiptTest(t)
		calls := stubSendReceiptToTelegramChat(t, context.DeadlineExceeded)
		seed(t, db,
			models4debtus.NewReceipt(receiptID, newReceiptDbo(models4debtus.ReceiptStatusCreated)).Record,
			models4debtus.NewTransfer(transferID, newTransferData()).Record,
			newCpUserRecord(),
			newTestTgChatEntry(botID, tgUserID).Record,
		)

		if err := DelayedSendReceiptToCounterpartyByTelegram(ctx, receiptID, 123, "en-UK"); err == nil {
			t.Fatal("expected error on generic send failure, got nil")
		}
		if *calls != 1 {
			t.Errorf("sendReceiptToTelegramChat called %d times, want 1", *calls)
		}
		tgChat := newTestTgChatEntry(botID, tgUserID)
		if err := db.Get(ctx, tgChat.Record); err != nil {
			t.Fatalf("failed to get tg chat: %v", err)
		}
		if !tgChat.Data.DtForbiddenLast.IsZero() {
			t.Error("tgChat.Data.DtForbiddenLast set on generic failure, want zero")
		}
	})
}
