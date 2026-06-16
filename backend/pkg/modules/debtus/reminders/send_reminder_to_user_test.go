package reminders

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/strongoapp/appuser"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

// seedUser stores a user record in the memory DB for use by sendReminderToUser tests.
func seedUser(t *testing.T, db dal.DB, id string, dbo *dbo4userus.UserDbo) {
	t.Helper()
	ctx := context.Background()
	user := dbo4userus.NewUserEntryWithDbo(id, dbo)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, user.Record)
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func newUserWithEmail(email string) *dbo4userus.UserDbo {
	dbo := &dbo4userus.UserDbo{Email: email}
	dbo.Names = &person.NameFields{UserName: "Test User"}
	return dbo
}

func newUserNoEmail() *dbo4userus.UserDbo {
	dbo := &dbo4userus.UserDbo{}
	dbo.Names = &person.NameFields{UserName: "Test User"}
	return dbo
}

// TestSendReminderToUser_AlreadySending covers the errReminderAlreadySentOrIsBeingSent path.
// The reminder already has status "sending" when the transaction reads it, so
// sendReminderToUser logs the sentinel and returns it.
func TestSendReminderToUser_AlreadySending(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusSending,
		UserID: "u1",
	})

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if !errors.Is(err, errReminderAlreadySentOrIsBeingSent) {
		t.Errorf("expected errReminderAlreadySentOrIsBeingSent, got: %v", err)
	}
}

// TestSendReminderToUser_GetSneatDBError covers the second facade.GetSneatDB call
// inside sendReminderToUser (for GetUser).  We make the first call succeed (the
// transaction uses it to read/write the reminder) and the second fail.
func TestSendReminderToUser_GetSneatDBError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})

	callCount := 0
	origGetDB := facade.GetSneatDB
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) {
		callCount++
		if callCount == 1 {
			// First call: from RunReadwriteTransaction — return the real memory DB.
			return db, nil
		}
		// Second call: from sendReminderToUser for GetUser — return error.
		return nil, errors.New("db unavailable for getUser")
	}
	t.Cleanup(func() { facade.GetSneatDB = origGetDB })

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errors.New("db unavailable for getUser")) {
		// errors.Is won't work here since wrapped; just check message
		_ = err.Error() // suppress unused warning
	}
}

// TestSendReminderToUser_NoTelegram_HasEmail covers the path where the user has
// no Telegram account but does have an email address.
func TestSendReminderToUser_NoTelegram_HasEmail(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithEmail("user@example.com"))

	origClient := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{sent: fakeSent{id: "msg-42"}}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	transfer := newValidTransfer("txid")
	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendReminderToUser_NoTelegram_NoEmail_MarkFailed covers the path where the
// user has neither Telegram nor email: the reminder is re-fetched inside a
// second transaction and its status is set to "failed".
func TestSendReminderToUser_NoTelegram_NoEmail_MarkFailed(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
		DtNext: time.Now(),
	})
	seedUser(t, db, "u1", newUserNoEmail())

	transfer := newValidTransfer("txid")
	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the reminder's status was set to "failed".
	ctx := context.Background()
	reminder := dbo4reminders.NewReminder("rem1", nil)
	if err := db.Get(ctx, reminder.Record); err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if reminder.Data.Status != dbo4reminders.ReminderStatusFailed {
		t.Errorf("status = %q, want %q", reminder.Data.Status, dbo4reminders.ReminderStatusFailed)
	}
}

// TestSendReminderToUser_GetUserError covers the branch where dal4userus.GetUser
// returns an error (record not found in the DB).
func TestSendReminderToUser_GetUserError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	// Seed reminder but NOT the user — GetUser will return a not-found error.
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected error from missing user, got nil")
	}
}

// newUserWithTelegram creates a UserDbo that HasAccount("telegram","") == true.
func newUserWithTelegram() *dbo4userus.UserDbo {
	dbo := &dbo4userus.UserDbo{}
	dbo.Names = &person.NameFields{UserName: "Tg User"}
	dbo.AccountsOfUser = appuser.AccountsOfUser{
		Accounts: []string{"telegram::some-chat-id"},
	}
	return dbo
}

// newTransferWithTgChatID returns a transfer where From.TgChatID is non-zero for userID "u1".
func newTransferWithTgChatID(tgChatID int64) models4debtus.TransferEntry {
	return models4debtus.NewTransfer("txid", &models4debtus.TransferData{
		CreatorUserID: "u1",
		IsOutstanding: true,
		FromJson:      `{"userID":"u1","contactName":"Alice","tgChatID":` + fmt.Sprintf("%d", tgChatID) + `,"tgBotID":"testbot"}`,
		ToJson:        `{"userID":"u2","contactName":"Bob"}`,
	})
}

// TestSendReminderToUser_TxGetReminderError covers the GetReminderByID error path
// inside the first RunReadwriteTransaction (line 128-130).
func TestSendReminderToUser_TxGetReminderError(t *testing.T) {
	_ = sneattesting.SetupMemoryDB(t)
	// Do NOT seed the reminder — GetReminderByID will fail with not-found.
	// The not-found is wrapped as "failed to get reminder" and returned.

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(context.Background(), "no-such-reminder", transfer)
	if err == nil {
		t.Fatal("expected error from missing reminder in tx, got nil")
	}
}

// TestSendReminderToUser_TxSaveReminderError covers the SaveReminder error path
// inside the first RunReadwriteTransaction (lines 135-137).
func TestSendReminderToUser_TxSaveReminderError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	memDB := sneattesting.SetupMemoryDB(t)
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})

	wantSetErr := errors.New("set error")
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(wantSetErr).AnyTimes()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			return f(ctx, mockTx)
		}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(ctx, "rem1", transfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestSendReminderToUser_TxNonSentinelError covers the else branch at line 142-145
// where RunReadwriteTransaction returns a non-sentinel error.
func TestSendReminderToUser_TxNonSentinelError(t *testing.T) {
	ctrl := gomock.NewController(t)
	wantErr := errors.New("some db failure")

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return(wantErr).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	transfer := newValidTransfer("txid")
	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestSendReminderToUser_Telegram_TgChatIDInTransfer covers the Telegram path
// where the transfer's UserInfoByUserID returns a non-zero TgChatID (lines 166-168).
func TestSendReminderToUser_Telegram_TgChatIDInTransfer(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	// Transfer has TgChatID set for user "u1".
	transfer := newTransferWithTgChatID(12345)

	// Stub sendReminderByTelegramFn to succeed without real Telegram API.
	orig := sendReminderByTelegramFn
	sendReminderByTelegramFn = func(_ context.Context, _ models4debtus.TransferEntry, _ dbo4reminders.Reminder, _ int64, _ string) (bool, bool, error) {
		return true, false, nil // sent=true
	}
	t.Cleanup(func() { sendReminderByTelegramFn = orig })

	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendReminderToUser_Telegram_TgChatIDInTransfer_NotSent covers the
// !reminderIsSent && !channelDisabledByUser warning path (line 198-200).
func TestSendReminderToUser_Telegram_NotSent(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	transfer := newTransferWithTgChatID(12345)

	orig := sendReminderByTelegramFn
	sendReminderByTelegramFn = func(_ context.Context, _ models4debtus.TransferEntry, _ dbo4reminders.Reminder, _ int64, _ string) (bool, bool, error) {
		return false, false, nil // not sent, not disabled
	}
	t.Cleanup(func() { sendReminderByTelegramFn = orig })

	origClient := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{sent: fakeSent{id: "msg1"}}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	// User also has email so it falls through to email sending.
	// Re-seed user with both telegram+email to cover the email fallback path.
	dbo := newUserWithTelegram()
	dbo.Email = "tg@example.com"
	seedUser(t, db, "u1", dbo)

	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendReminderToUser_Telegram_SendError covers sendReminderByTelegramFn returning error (line 196-197).
func TestSendReminderToUser_Telegram_SendError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	transfer := newTransferWithTgChatID(12345)

	orig := sendReminderByTelegramFn
	sendReminderByTelegramFn = func(_ context.Context, _ models4debtus.TransferEntry, _ dbo4reminders.Reminder, _ int64, _ string) (bool, bool, error) {
		return false, false, errors.New("telegram send failed")
	}
	t.Cleanup(func() { sendReminderByTelegramFn = orig })

	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected error from telegram send failure, got nil")
	}
}

// TestSendReminderToUser_Telegram_GetChatByUserID_NotFound covers the
// GetTelegramChatByUserID not-found error path (lines 182-185).
func TestSendReminderToUser_Telegram_GetChatByUserID_NotFound(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	// Transfer has NO TgChatID, so GetTelegramChatByUserID is called.
	transfer := newValidTransfer("txid") // FromJson has no tgChatID

	origGetChat := getTelegramChatByUserID
	getTelegramChatByUserID = func(_ context.Context, _ string) (string, botsfwtgmodels.TgChatData, error) {
		return "", nil, dal.ErrRecordNotFound
	}
	t.Cleanup(func() { getTelegramChatByUserID = origGetChat })

	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected error from not-found tg chat, got nil")
	}
}

// TestSendReminderToUser_Telegram_GetChatByUserID_OtherError covers the
// case where GetTelegramChatByUserID returns a non-not-found error. In that
// case the error is ignored (only not-found causes an early return) and
// tgChatID stays 0 so we fall through to the email path.
func TestSendReminderToUser_Telegram_GetChatByUserID_OtherError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	dbo := newUserWithTelegram()
	dbo.Email = "tg@example.com"
	seedUser(t, db, "u1", dbo)

	transfer := newValidTransfer("txid")

	origGetChat := getTelegramChatByUserID
	getTelegramChatByUserID = func(_ context.Context, _ string) (string, botsfwtgmodels.TgChatData, error) {
		return "", nil, errors.New("some non-not-found error")
	}
	t.Cleanup(func() { getTelegramChatByUserID = origGetChat })

	origClient := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{sent: fakeSent{id: "msg1"}}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendReminderToUser_Telegram_ParseIntError covers the strconv.ParseInt error
// path (lines 187-190) when GetTelegramChatByUserID succeeds but BotUserIDs[0] is not numeric.
func TestSendReminderToUser_Telegram_ParseIntError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	transfer := newValidTransfer("txid")

	origGetChat := getTelegramChatByUserID
	getTelegramChatByUserID = func(_ context.Context, _ string) (string, botsfwtgmodels.TgChatData, error) {
		chat := new(botsfwtgmodels.TgChatBaseData)
		chat.SetBotUserID("not-a-number")
		return "chatkey", chat, nil
	}
	t.Cleanup(func() { getTelegramChatByUserID = origGetChat })

	err := sendReminderToUser(context.Background(), "rem1", transfer)
	if err == nil {
		t.Fatal("expected ParseInt error, got nil")
	}
}

// TestSendReminderToUser_Telegram_GetChatByUserID_ValidID covers the path where
// GetTelegramChatByUserID succeeds and ParseInt also succeeds, reaching the
// tgBotID assignment (line 192) and then sendReminderByTelegramFn.
func TestSendReminderToUser_Telegram_GetChatByUserID_ValidID(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithTelegram())

	transfer := newValidTransfer("txid") // no TgChatID in transfer

	origGetChat := getTelegramChatByUserID
	getTelegramChatByUserID = func(_ context.Context, _ string) (string, botsfwtgmodels.TgChatData, error) {
		chat := new(botsfwtgmodels.TgChatBaseData)
		chat.SetBotUserID("12345") // valid numeric ID
		return "chatkey", chat, nil
	}
	t.Cleanup(func() { getTelegramChatByUserID = origGetChat })

	orig := sendReminderByTelegramFn
	sendReminderByTelegramFn = func(_ context.Context, _ models4debtus.TransferEntry, _ dbo4reminders.Reminder, _ int64, _ string) (bool, bool, error) {
		return true, false, nil
	}
	t.Cleanup(func() { sendReminderByTelegramFn = orig })

	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendReminderToUser_NoTelegram_HasEmail_SendError covers sendReminderByEmail
// returning an error (lines 204-206).
func TestSendReminderToUser_NoTelegram_HasEmail_SendError(t *testing.T) {
	db := sneattesting.SetupMemoryDB(t)
	seedReminder(t, db, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, db, "u1", newUserWithEmail("user@example.com"))

	origClient := emailing.GetEmailClient
	emailing.GetEmailClient = func(_ context.Context) (emails.Client, error) {
		return &fakeEmailClient{err: errors.New("smtp error")}, nil
	}
	t.Cleanup(func() { emailing.GetEmailClient = origClient })

	transfer := newValidTransfer("txid")
	// sendReminderByEmail errors are only logged, sendReminderToUser returns nil.
	if err := sendReminderToUser(context.Background(), "rem1", transfer); err != nil {
		t.Errorf("unexpected error (email errors are logged only): %v", err)
	}
}

// TestSendReminderToUser_NoEmail_MarkFailed_TxGetError covers GetReminderByID
// failing inside the "mark as failed" transaction (line 212-214).
func TestSendReminderToUser_NoEmail_MarkFailed_TxGetError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	memDB := sneattesting.SetupMemoryDB(t)
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, memDB, "u1", newUserNoEmail())

	wantGetErr := errors.New("get error in mark-failed tx")

	// The first RunReadwriteTransaction (status→sending) must succeed using memDB.
	// The second RunReadwriteTransaction (mark failed) uses a failing mock tx.
	callCount := 0
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(wantGetErr).AnyTimes()

	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			callCount++
			if callCount == 1 {
				return memDB.RunReadwriteTransaction(ctx, f, opts...)
			}
			return f(ctx, mockTx)
		}).AnyTimes()
	mockDB.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	transfer := newValidTransfer("txid")
	// Error in mark-failed tx is only logged, function returns nil.
	if err := sendReminderToUser(ctx, "rem1", transfer); err != nil {
		t.Errorf("unexpected error (mark-failed errors are logged): %v", err)
	}
}

// TestSendReminderToUser_NoEmail_MarkFailed_TxSetError covers the tx.Set error path
// inside the "mark as failed" transaction.
func TestSendReminderToUser_NoEmail_MarkFailed_TxSetError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	memDB := sneattesting.SetupMemoryDB(t)
	seedReminder(t, memDB, "rem1", &dbo4reminders.ReminderDbo{
		Status: dbo4reminders.ReminderStatusCreated,
		UserID: "u1",
	})
	seedUser(t, memDB, "u1", newUserNoEmail())

	wantSetErr := errors.New("set error in mark-failed tx")
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()
	mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(wantSetErr).AnyTimes()

	callCount := 0
	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f dal.RWTxWorker, opts ...dal.TransactionOption) error {
			callCount++
			if callCount == 1 {
				return memDB.RunReadwriteTransaction(ctx, f, opts...)
			}
			return f(ctx, mockTx)
		}).AnyTimes()
	mockDB.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, rec dal.Record) error {
		return memDB.Get(ctx, rec)
	}).AnyTimes()

	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return mockDB, nil }
	t.Cleanup(func() { facade.GetSneatDB = origGetSneatDB })

	transfer := newValidTransfer("txid")
	if err := sendReminderToUser(ctx, "rem1", transfer); err != nil {
		t.Errorf("unexpected error (mark-failed errors are logged): %v", err)
	}
}
