package debtusdal

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/delayer4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/delaying"
	"github.com/strongo/strongoapp"
)

// ---- AdminDal.DeleteAll ----

func TestAdminDal_DeleteAll_panics(t *testing.T) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from DeleteAll, got none")
		}
	}()
	_ = NewAdminDal().DeleteAll(ctx, "bot", "chat1")
}

// ---- NewDAL.HttpClient closure ----

func TestNewDAL_HttpClient_returns_non_nil(t *testing.T) {
	d := NewDAL()
	client := d.HttpClient(context.Background())
	if client == nil {
		t.Error("HttpClient() returned nil")
	}
}

// ---- ApiBotHost ----

func TestApiBotHost_Context(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	host := ApiBotHost{}
	ctx := host.Context(r)
	if ctx == nil {
		t.Error("Context() returned nil")
	}
}

func TestApiBotHost_GetHTTPClient(t *testing.T) {
	sneattesting.SetupMemoryDB(t)
	RegisterDal()
	client := ApiBotHost{}.GetHTTPClient(context.Background())
	if client == nil {
		t.Error("GetHTTPClient() returned nil")
	}
}

func TestApiBotHost_DB(t *testing.T) {
	sneattesting.SetupMemoryDB(t)
	db, err := ApiBotHost{}.DB(context.Background())
	if err != nil {
		t.Fatalf("DB() returned error: %v", err)
	}
	if db == nil {
		t.Error("DB() returned nil")
	}
}

// ---- ContactDal.SaveContact error path ----

type failingWriteTx struct {
	dal.ReadwriteTransaction
}

func (failingWriteTx) Set(_ context.Context, _ dal.Record) error {
	return errFakeSet
}

func (failingWriteTx) Get(_ context.Context, _ dal.Record) error {
	return errFakeSet
}

var errFakeSet = fmt.Errorf("fake set error: %w", dal.ErrRecordNotFound)

func TestContactDal_SaveContact_error_path(t *testing.T) {
	ctx := context.Background()
	contact := models4debtus.NewDebtusSpaceContactEntry("s1", "c1", &models4debtus.DebtusSpaceContactDbo{})
	err := NewContactDal().SaveContact(ctx, failingWriteTx{}, contact)
	if err == nil {
		t.Error("expected error when tx.Set fails, got nil")
	}
}

// ---- query builder helpers (just call them to get coverage) ----

func TestNewUserContactsQuery(t *testing.T) {
	q := newUserContactsQuery("u1")
	if q == nil {
		t.Error("newUserContactsQuery returned nil")
	}
}

func TestNewUserActiveContactsQuery(t *testing.T) {
	q := newUserActiveContactsQuery("u1")
	if q == nil {
		t.Error("newUserActiveContactsQuery returned nil")
	}
}

// ---- NewReminderKey panic path ----

func TestNewReminderKey_panics_on_empty_id(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when reminderID is empty, got none")
		}
	}()
	_ = NewReminderKey("")
}

// ---- reminderIDsFromRecords type assertion failure ----

func TestReminderIDsFromRecords_type_assertion_failure(t *testing.T) {
	// Create a record whose key has a non-string ID
	key := dal.NewKeyWithID(dbo4reminders.ReminderKind, int64(42))
	record := dal.NewRecordWithData(key, &dbo4reminders.ReminderDbo{})
	_, err := reminderIDsFromRecords([]dal.Record{record})
	if err == nil {
		t.Error("expected error when key ID is not a string, got nil")
	}
}

// ---- DelayedSetReminderIsSent ----

func TestDelayedSetReminderIsSent(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	const reminderID = "rem-set-sent"
	sentAt := time.Now()

	// Seed a reminder
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		key := NewReminderKey(reminderID)
		return tx.Set(ctx, dal.NewRecordWithData(key, &dbo4reminders.ReminderDbo{
			TargetID: "t1",
		}))
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Call DelayedSetReminderIsSent — messageIntID=123, messageStrID="" is valid
	err = DelayedSetReminderIsSent(ctx, reminderID, sentAt, 123, "", "en", "")
	if err != nil {
		t.Fatalf("DelayedSetReminderIsSent() returned error: %v", err)
	}
}

// ---- ReminderDal.DelayCreateReminderForTransferUser enqueue path ----

func TestReminderDal_DelayCreateReminderForTransferUser_enqueues(t *testing.T) {
	ctx := context.Background()
	// CreateReminderForTransferUser is not registered by RegisterDelayers4Debtus;
	// set it directly as a no-op for this test.
	orig := delayer4debtus.CreateReminderForTransferUser
	delayer4debtus.CreateReminderForTransferUser = delaying.VoidWithLog("create-reminder-4-transfer-user", func(_ context.Context, _, _ string) error { return nil })
	t.Cleanup(func() { delayer4debtus.CreateReminderForTransferUser = orig })

	err := NewReminderDal().DelayCreateReminderForTransferUser(ctx, "t1", "u1")
	if err != nil {
		t.Errorf("DelayCreateReminderForTransferUser() returned unexpected error: %v", err)
	}
}

// ---- ReminderDal.DelayDiscardRemindersForTransfers non-empty path ----

func TestReminderDal_DelayDiscardRemindersForTransfers_non_empty(t *testing.T) {
	ctx := context.Background()
	// DiscardRemindersForTransfers is not registered by RegisterDelayers4Debtus;
	// set it directly as a no-op for this test.
	orig := delayer4debtus.DiscardRemindersForTransfers
	delayer4debtus.DiscardRemindersForTransfers = delaying.VoidWithLog("discard-reminders", func(_ context.Context, _ []string, _ string) error { return nil })
	t.Cleanup(func() { delayer4debtus.DiscardRemindersForTransfers = orig })

	err := NewReminderDal().DelayDiscardRemindersForTransfers(ctx, []string{"t1"}, "rt1")
	if err != nil {
		t.Errorf("DelayDiscardRemindersForTransfers() returned unexpected error: %v", err)
	}
}

// ---- TransferDal.GetTransfersByID happy path (covers the GetMulti branch) ----

func TestTransferDal_GetTransfersByID_happy_path(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// Seed one transfer so GetMulti succeeds on the happy path
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewTransfer("t1", &models4debtus.TransferData{}).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	transfers, err := NewTransferDal().GetTransfersByID(ctx, db, []string{"t1"})
	if err != nil {
		t.Fatalf("GetTransfersByID() returned error: %v", err)
	}
	if len(transfers) != 1 {
		t.Errorf("got %d transfers, want 1", len(transfers))
	}
}

// ---- TransferDal.LoadLatestTransfers ----

func TestTransferDal_LoadLatestTransfers(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	// Seed two transfers
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		for _, id := range []string{"t1", "t2"} {
			if err := tx.Set(ctx, models4debtus.NewTransfer(id, &models4debtus.TransferData{}).Record); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	transfers, err := NewTransferDal().LoadLatestTransfers(ctx, 0, 10)
	if err != nil {
		t.Fatalf("LoadLatestTransfers() returned error: %v", err)
	}
	if len(transfers) != 2 {
		t.Errorf("got %d transfers, want 2", len(transfers))
	}
}

// ---- TransferDal.DelayUpdateTransferWithCreatorReceiptTgMessageID ----

func TestTransferDal_DelayUpdateTransferWithCreatorReceiptTgMessageID(t *testing.T) {
	ctx := context.Background()
	// UpdateTransferWithCreatorReceiptTgMessageID is not in RegisterDelayers4Debtus; set directly.
	orig := delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID
	delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID = delaying.VoidWithLog(
		"update-transfer-with-creator-receipt-tg-message-id",
		func(_ context.Context, _, _ string, _, _ int64) error { return nil },
	)
	t.Cleanup(func() { delayer4debtus.UpdateTransferWithCreatorReceiptTgMessageID = orig })

	err := NewTransferDal().DelayUpdateTransferWithCreatorReceiptTgMessageID(ctx, "tgbot", "t1", 123, 456)
	if err != nil {
		t.Errorf("DelayUpdateTransferWithCreatorReceiptTgMessageID() returned error: %v", err)
	}
}

// ---- delayedUpdateTransfersOnReturn valid path ----

func TestDelayedUpdateTransfersOnReturn_valid(t *testing.T) {
	ctx := context.Background()
	RegisterDelayers4Debtus(delaying.VoidWithLog)

	err := delayedUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
		{TransferID: "t1", ReturnedAmount: 100},
	})
	if err != nil {
		t.Errorf("delayedUpdateTransfersOnReturn() returned error: %v", err)
	}
}

// ---- ReceiptDal.DelayCreateAndSendReceiptToCounterpartyByTelegram ----

func TestReceiptDal_DelayCreateAndSendReceiptToCounterpartyByTelegram(t *testing.T) {
	ctx := context.Background()
	// CreateAndSendReceiptToCounterpartyByTelegram is not in RegisterDelayers4Debtus; set directly.
	orig := delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram
	delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram = delaying.VoidWithLog(
		"create-and-send-receipt-for-counterparty-by-telegram",
		func(_ context.Context, _, _, _ string) error { return nil },
	)
	t.Cleanup(func() { delayer4debtus.CreateAndSendReceiptToCounterpartyByTelegram = orig })

	err := NewReceiptDal().DelayCreateAndSendReceiptToCounterpartyByTelegram(ctx, "prod", "t1", "u1")
	if err != nil {
		t.Errorf("DelayCreateAndSendReceiptToCounterpartyByTelegram() returned error: %v", err)
	}
}

// ---- TwilioDal.GetLastTwilioSmsesForUser ----

func TestTwilioDal_GetLastTwilioSmsesForUser(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	result, err := NewTwilioDal().GetLastTwilioSmsesForUser(ctx, db, "u1", "", 10)
	if err != nil {
		t.Fatalf("GetLastTwilioSmsesForUser() returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

// ---- TransferFixer unit tests ----

func newTestTransferData() *models4debtus.TransferData {
	// ContactName intentionally left empty so needFixCounterpartyCounterpartyName returns true.
	return models4debtus.NewTransferData(
		"u1", false,
		money.Amount{Value: 100, Currency: "USD"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2"},
	)
}

func TestTransferFixer_needFixCounterpartyCounterpartyName(t *testing.T) {
	t.Run("needs_fix_when_contact_name_empty", func(t *testing.T) {
		fixer := NewTransferFixer(models4debtus.NewTransferKey("t1"), newTestTransferData())
		if !fixer.needFixCounterpartyCounterpartyName() {
			t.Error("expected needFix=true when ContactName is empty")
		}
	})
}

func TestTransferFixer_needFixes(t *testing.T) {
	ctx := context.Background()
	fixer := NewTransferFixer(models4debtus.NewTransferKey("t1"), newTestTransferData())
	if !fixer.needFixes(ctx) {
		t.Error("expected needFixes=true when ContactName is empty")
	}
}

// ---- CreateInvite invalid code path ----

func TestCreateMassInvite_invalid_code(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	// Invite code with only excluded chars (0, 1) is invalid per InviteCodeRegex (which uses [ABCDEFGHJKLMNPQRSTUVWXYZ23456789]+)
	_, err := NewInviteDal().CreateMassInvite(ec, "u1", "000", 1, "telegram")
	if err == nil {
		t.Error("expected error for invalid invite code, got nil")
	}
}

// ---- delayedUpdateInviteClaimedCount - invite not found path ----

func TestDelayedUpdateInviteClaimedCount_invite_not_found(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	// Seed an invite claim that references a non-existent invite code
	claimID := "claim-orphan"
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		claimData := models4debtus.NewInviteClaimData("NOEXIST", "u1", "telegram", "DebtusBot")
		claimData.DtClaimed = time.Now()
		claim := models4debtus.NewInviteClaim(claimID, claimData)
		return tx.Set(ctx, claim.Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Should return nil (not retry) when invite is not found
	if err = delayedUpdateInviteClaimedCount(ctx, claimID); err != nil {
		t.Errorf("expected nil when invite missing, got: %v", err)
	}
}

// ---- delayedUpdateInviteClaimedCount - claim already processed (LastClaimIDs contains claimID) ----

func TestDelayedUpdateInviteClaimedCount_already_processed(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	const (
		inviteCode = "CLMTEST"
		claimID    = "claim-dup"
	)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		// Seed invite that already has this claimID in LastClaimIDs
		inviteData := &models4debtus.InviteData{
			CreatedByUserID: "u-creator",
			LastClaimIDs:    []string{claimID},
		}
		if err := tx.Set(ctx, models4debtus.NewInvite(inviteCode, inviteData).Record); err != nil {
			return err
		}
		// Seed the claim
		claimData := models4debtus.NewInviteClaimData(inviteCode, "u1", "telegram", "DebtusBot")
		claimData.DtClaimed = time.Now()
		return tx.Set(ctx, models4debtus.NewInviteClaim(claimID, claimData).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err = delayedUpdateInviteClaimedCount(ctx, claimID); err != nil {
		t.Errorf("expected nil when claim already processed, got: %v", err)
	}
}

// ---- ContactDal.GetContactIDsByTitle case-insensitive path ----

func TestContactDal_GetContactIDsByTitle_case_insensitive(t *testing.T) {
	ctx := context.Background()

	// GetContactIDsByTitle loads a contactus space; returns error when space is missing.
	// Exercises the caseSensitive=false branch guard (returns before the loop due to error).
	db := sneattesting.SetupMemoryDB(t)
	_, err := NewContactDal().GetContactIDsByTitle(ctx, db, "s1", "u1", "Alice", false)
	if err == nil {
		t.Error("expected error when contactus space is missing, got nil")
	}
}

// ---- ClaimInvite2 path - MaxClaimsCount > 1, skips counterparty lookup ----

func TestClaimInvite2_basic(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	const (
		inviteCode    = "CI2TST"
		claimedUserID = "u-claimant"
		creatorUserID = "u-creator"
	)

	// Seed invite and user
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		inviteData := &models4debtus.InviteData{
			CreatedByUserID: creatorUserID,
			MaxClaimsCount:  2, // >1 to skip counterparty lookup path
		}
		if err := tx.Set(ctx, models4debtus.NewInvite(inviteCode, inviteData).Record); err != nil {
			return err
		}
		user := dbo4userus.NewUserEntry(claimedUserID)
		return tx.Set(ctx, user.Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	invite := models4debtus.NewInvite(inviteCode, &models4debtus.InviteData{
		CreatedByUserID: creatorUserID,
		MaxClaimsCount:  2,
	})
	err = NewInviteDal().ClaimInvite2(ctx, inviteCode, invite, claimedUserID, "telegram", "DebtusBot")
	if err != nil {
		t.Fatalf("ClaimInvite2() returned error: %v", err)
	}
}

// ---- delayedFixTransfersIsOutstanding with empty slice ----

func TestDelayedFixTransfersIsOutstanding_empty(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	RegisterDal()

	// Empty slice: the loop body never executes, just returns nil
	err := delayedFixTransfersIsOutstanding(ctx, []string{})
	if err != nil {
		t.Errorf("expected nil for empty slice, got: %v", err)
	}
}

// ---- delayedUpdateInviteClaimedCount - trim to 10 path ----

func TestDelayedUpdateInviteClaimedCount_trim_to_10(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	const (
		inviteCode = "TRIM10T"
		claimID    = "claim-new"
	)

	// Build an invite with 11 existing claim IDs (triggers trim to 10)
	existing := make([]string, 11)
	for i := range existing {
		existing[i] = fmt.Sprintf("old-claim-%d", i)
	}

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		inviteData := &models4debtus.InviteData{
			CreatedByUserID: "u-creator",
			LastClaimIDs:    existing,
		}
		if err := tx.Set(ctx, models4debtus.NewInvite(inviteCode, inviteData).Record); err != nil {
			return err
		}
		claimData := models4debtus.NewInviteClaimData(inviteCode, "u1", "telegram", "DebtusBot")
		claimData.DtClaimed = time.Now()
		return tx.Set(ctx, models4debtus.NewInviteClaim(claimID, claimData).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err = delayedUpdateInviteClaimedCount(ctx, claimID); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

// ---- delayedMarkReceiptAsSent not-found path (covers line 108) ----

func TestDelayedMarkReceiptAsSent_not_found(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	RegisterDal()
	RegisterDelayers4Debtus(delaying.VoidWithLog)

	// Receipt doesn't exist → MarkReceiptAsSent returns not-found → delayedMarkReceiptAsSent returns nil
	err := delayedMarkReceiptAsSent(ctx, "no-such-receipt", "no-such-transfer", time.Now())
	if err != nil {
		t.Errorf("expected nil when receipt not found, got: %v", err)
	}
}

// ---- delayedMarkReceiptAsSent success path (covers line 110) ----

func TestDelayedMarkReceiptAsSent_success(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()
	RegisterDelayers4Debtus(delaying.VoidWithLog)

	const (
		receiptID  = "r-test-sent"
		transferID = "t-test-sent"
	)

	// Seed a receipt and transfer so MarkReceiptAsSent can succeed
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		receiptData := &models4debtus.ReceiptDbo{
			TransferID:         transferID,
			CreatorUserID:      "u1",
			CounterpartyUserID: "u2",
		}
		if err := tx.Set(ctx, models4debtus.NewReceipt(receiptID, receiptData).Record); err != nil {
			return err
		}
		transferData := newTestTransferData()
		return tx.Set(ctx, models4debtus.NewTransfer(transferID, transferData).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	err = delayedMarkReceiptAsSent(ctx, receiptID, transferID, time.Now())
	if err != nil {
		t.Errorf("delayedMarkReceiptAsSent() returned error: %v", err)
	}
}
