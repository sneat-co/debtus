package debtusdal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/strongo/gotwilio"
	"github.com/strongo/strongoapp"
)

// TestLoadTransfers_queryErrorBranches covers the `loadTransfers` error returns of
// LoadTransfersByUserID and LoadTransfersByContactID, plus the
// ExecuteQueryToRecordsReader error of LoadTransferIDsByContactID. All three go
// through facade.GetSneatDB, so a faultQuery failingDB forces the query to fail
// after input validation has passed.
func TestLoadTransfers_queryErrorBranches(t *testing.T) {
	ctx := context.Background()

	cases := map[string]func() error{
		"LoadTransfersByUserID": func() error {
			_, _, err := NewTransferDal().LoadTransfersByUserID(ctx, "u1", 0, 10)
			return err
		},
		"LoadTransfersByContactID": func() error {
			_, _, err := NewTransferDal().LoadTransfersByContactID(ctx, "c1", 0, 10)
			return err
		},
		"LoadTransferIDsByContactID": func() error {
			_, _, err := NewTransferDal().LoadTransferIDsByContactID(ctx, "c1", 10, "")
			return err
		},
	}

	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			real := sneattesting.SetupMemoryDB(t)
			withFailingFacadeDB(t, real, faultQuery)
			if err := call(); !errors.Is(err, errInjected) {
				t.Errorf("%s: expected errInjected, got %v", name, err)
			}
		})
	}
}

// TestCreatePersonalInvite_smsChannel covers the InviteBySms branch of
// createInvite: a valid numeric address is parsed and stored as ToPhoneNumber.
func TestCreatePersonalInvite_smsChannel(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	invite, err := NewInviteDal().CreatePersonalInvite(ec, "u1", models4debtus.InviteBySms, "353857000000", "telegram", "DebtusBot", "")
	if err != nil {
		t.Fatalf("CreatePersonalInvite(sms) error: %v", err)
	}
	stored := models4debtus.NewInvite(invite.ID, nil)
	if err = db.Get(ctx, stored.Record); err != nil {
		t.Fatalf("stored invite not found: %v", err)
	}
	if stored.Data.ToPhoneNumber != 353857000000 {
		t.Errorf("stored ToPhoneNumber = %d, want 353857000000", stored.Data.ToPhoneNumber)
	}
	if stored.Data.Channel != string(models4debtus.InviteBySms) {
		t.Errorf("stored Channel = %q, want %q", stored.Data.Channel, models4debtus.InviteBySms)
	}
}

// TestDeleteContact_delayerError covers the error return in DeleteContact after
// tx.Delete succeeds but delayDeleteContactTransfers fails to enqueue work.
func TestDeleteContact_delayerError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	orig := delayer4debtus.DeleteContactTransfersDelayFunc
	delayer4debtus.DeleteContactTransfersDelayFunc = failingDelayer{id: "delete-contact-transfers"}
	t.Cleanup(func() { delayer4debtus.DeleteContactTransfersDelayFunc = orig })

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return NewContactDal().DeleteContact(ctx, tx, "space1", "c1")
	})
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestClaimInvite_insertError covers the "failed to insert invite claim" branch:
// the invite and user load successfully, then tx.Insert fails.
func TestClaimInvite_insertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()
	setupDelayers(t)

	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("creator1").Record)
	seedRecord(t, ctx, db, models4debtus.NewInvite("INV-INS", &models4debtus.InviteData{
		CreatedByUserID: "creator1",
		MaxClaimsCount:  1,
	}).Record)
	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("claimer1").Record)

	withFailingFacadeDB(t, db, faultInsert)
	if err := NewInviteDal().ClaimInvite(ctx, "claimer1", "INV-INS", "Telegram", "DebtusBot"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestClaimInvite_setUserError covers the "failed to save user" branch: invite,
// user load and the claim insert succeed, then tx.Set on the user record fails.
func TestClaimInvite_setUserError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()
	setupDelayers(t)

	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("creator1").Record)
	seedRecord(t, ctx, db, models4debtus.NewInvite("INV-SET", &models4debtus.InviteData{
		CreatedByUserID: "creator1",
		MaxClaimsCount:  1,
	}).Record)
	seedRecord(t, ctx, db, dbo4userus.NewUserEntry("claimer1").Record)

	withFailingFacadeDB(t, db, faultSet)
	if err := NewInviteDal().ClaimInvite(ctx, "claimer1", "INV-SET", "Telegram", "DebtusBot"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestDelayedUpdateTransfersOnReturn_enqueuesEach covers the success path of
// delayedUpdateTransfersOnReturn: it enqueues one delayed task per update and
// returns nil.
func TestDelayedUpdateTransfersOnReturn_enqueuesEach(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)

	err := delayedUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
		{TransferID: "t1", ReturnedAmount: 100},
		{TransferID: "t2", ReturnedAmount: 200},
	})
	if err != nil {
		t.Errorf("delayedUpdateTransfersOnReturn() returned error: %v", err)
	}
}

// TestDelayedUpdateTransfersOnReturn_enqueueError covers the error return when
// DelayUpdateTransferOnReturn fails to enqueue work.
func TestDelayedUpdateTransfersOnReturn_enqueueError(t *testing.T) {
	ctx := context.Background()

	orig := delayer4debtus.UpdateTransferOnReturn
	delayer4debtus.UpdateTransferOnReturn = failingDelayer{id: "update-transfer-on-return"}
	t.Cleanup(func() { delayer4debtus.UpdateTransferOnReturn = orig })

	err := delayedUpdateTransfersOnReturn(ctx, "rt1", []dal4debtus.TransferReturnUpdate{
		{TransferID: "t1", ReturnedAmount: 100},
	})
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestInsertReward_insertError covers the error return of InsertReward when the
// underlying tx.Insert fails.
func TestInsertReward_insertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, e := NewRewardDal().InsertReward(ctx, failingTx{ReadwriteTransaction: tx, fault: faultInsert}, &models4debtus.RewardDbo{})
		return e
	})
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestLoadTransferIDsByContactID_facadeDBError covers the early return when
// facade.GetSneatDB itself fails, after input validation has passed.
func TestLoadTransferIDsByContactID_facadeDBError(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	withErroringFacadeDB(t)
	if _, _, err := NewTransferDal().LoadTransferIDsByContactID(ctx, "c1", 10, ""); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestDelayedUpdateInviteClaimedCount_inviteNotFound covers the branch where the
// claim exists but the invite it references does not: the handler logs and
// returns nil (deliberately, to avoid retries) without error.
func TestDelayedUpdateInviteClaimedCount_inviteNotFound(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	seedRecord(t, ctx, db, models4debtus.NewInviteClaim("clm-orphan",
		models4debtus.NewInviteClaimData("MISSING-INVITE", "u2", "Telegram", "bot")).Record)

	if err := delayedUpdateInviteClaimedCount(ctx, "clm-orphan"); err != nil {
		t.Errorf("expected nil when invite not found, got %v", err)
	}
}

// TestDelayedUpdateInviteClaimedCount_getClaimError covers the non-NotFound
// error return when reading the claim fails (line 31), and the outer error log.
func TestDelayedUpdateInviteClaimedCount_getClaimError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	withFailingFacadeDB(t, db, faultGet)
	if err := delayedUpdateInviteClaimedCount(ctx, "clm-x"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestDelayedUpdateInviteClaimedCount_setInviteError covers the "failed to save
// invite" branch: claim and invite load successfully, then tx.Set fails.
func TestDelayedUpdateInviteClaimedCount_setInviteError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	seedRecord(t, ctx, db, models4debtus.NewInvite("INV-SETERR", &models4debtus.InviteData{CreatedByUserID: "u1"}).Record)
	seedRecord(t, ctx, db, models4debtus.NewInviteClaim("clm-seterr",
		models4debtus.NewInviteClaimData("INV-SETERR", "u2", "Telegram", "bot")).Record)

	withFailingFacadeDB(t, db, faultSet)
	if err := delayedUpdateInviteClaimedCount(ctx, "clm-seterr"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestCreatePersonalInvite_withRelated covers the valid `related` parsing branch
// of createInvite (a single key=value pair) and verifies it is persisted.
func TestCreatePersonalInvite_withRelated(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	invite, err := NewInviteDal().CreatePersonalInvite(ec, "u1", models4debtus.InviteByEmail, "friend@example.com", "telegram", "DebtusBot", "ref=abc")
	if err != nil {
		t.Fatalf("CreatePersonalInvite(related) error: %v", err)
	}
	stored := models4debtus.NewInvite(invite.ID, nil)
	if err = db.Get(ctx, stored.Record); err != nil {
		t.Fatalf("stored invite not found: %v", err)
	}
	if stored.Data.Related != "ref=abc" {
		t.Errorf("stored Related = %q, want ref=abc", stored.Data.Related)
	}
}

// TestCreateInvite_defaultsCodeLen covers the inviteCodeLen == 0 branch of
// createInvite, which falls back to INVITE_CODE_LENGTH for an auto-generated code.
func TestCreateInvite_defaultsCodeLen(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	invite, err := createInvite(ec, models4debtus.InviteTypePersonal, "u1", models4debtus.InviteByEmail,
		"friend@example.com", "telegram", "DebtusBot", 0, AUTO_GENERATE_INVITE_CODE, "", PERSONAL_INVITE)
	if err != nil {
		t.Fatalf("createInvite() error: %v", err)
	}
	if len(invite.ID) != INVITE_CODE_LENGTH {
		t.Errorf("auto code len = %d, want default %d", len(invite.ID), INVITE_CODE_LENGTH)
	}
}

// TestCreateInvite_setError covers the error branch of createInvite when tx.Set
// of the new invite record fails.
func TestCreateInvite_setError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	withFailingFacadeDB(t, db, faultSet)
	_, err := NewInviteDal().CreateMassInvite(ec, "u1", "FIXED", 100, "telegram")
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestClaimInvite2_facadeDBError covers the early return when facade.GetSneatDB
// fails before ClaimInvite2 opens its transaction.
func TestClaimInvite2_facadeDBError(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	withErroringFacadeDB(t)
	invite := models4debtus.NewInvite("INV", &models4debtus.InviteData{MaxClaimsCount: 5})
	if err := NewInviteDal().ClaimInvite2(ctx, "INV", invite, "u1", "Telegram", "DebtusBot"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// seedClaimInvite2 seeds an invite (with the given MaxClaimsCount/ClaimedCount)
// and the claimant user, returning the invite arg for ClaimInvite2.
func seedClaimInvite2(t *testing.T, ctx context.Context, db dal.DB, code, creatorID, claimerID string, maxClaims, claimedCount int32) models4debtus.Invite {
	t.Helper()
	data := &models4debtus.InviteData{
		CreatedByUserID: creatorID,
		MaxClaimsCount:  maxClaims,
		ClaimedCount:    claimedCount,
	}
	seedRecord(t, ctx, db, models4debtus.NewInvite(code, data).Record)
	seedRecord(t, ctx, db, dbo4userus.NewUserEntry(claimerID).Record)
	return models4debtus.NewInvite(code, &models4debtus.InviteData{
		CreatedByUserID: creatorID,
		MaxClaimsCount:  maxClaims,
	})
}

// TestClaimInvite2_getMultiError covers the GetMulti error return: the
// invite/user load via tx.GetMulti fails.
func TestClaimInvite2_getMultiError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	invite := seedClaimInvite2(t, ctx, db, "CI2-GM", "creator", "claimer", 2, 0)
	withFailingFacadeDB(t, db, faultGetMulti)
	if err := NewInviteDal().ClaimInvite2(ctx, "CI2-GM", invite, "claimer", "telegram", "DebtusBot"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestClaimInvite2_exceedsMaxClaims covers the branch where incrementing
// ClaimedCount pushes it above MaxClaimsCount, returning a descriptive error.
func TestClaimInvite2_exceedsMaxClaims(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	// Seeded ClaimedCount == MaxClaimsCount == 2; after +1 → 3 > 2 → error.
	invite := seedClaimInvite2(t, ctx, db, "CI2-MAX", "creator", "claimer", 2, 2)
	err := NewInviteDal().ClaimInvite2(ctx, "CI2-MAX", invite, "claimer", "telegram", "DebtusBot")
	if err == nil {
		t.Fatal("expected error when ClaimedCount exceeds MaxClaimsCount, got nil")
	}
	if !strings.Contains(err.Error(), "MaxClaimsCount") {
		t.Errorf("error %q does not mention MaxClaimsCount", err.Error())
	}
}

// TestClaimInvite2_insertClaimError covers the error return when inserting the
// invite claim fails (MaxClaimsCount > 1 path, so the counterparty lookup is
// skipped).
func TestClaimInvite2_insertClaimError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	RegisterDal()

	invite := seedClaimInvite2(t, ctx, db, "CI2-INS", "creator", "claimer", 2, 0)
	withFailingFacadeDB(t, db, faultInsert)
	if err := NewInviteDal().ClaimInvite2(ctx, "CI2-INS", invite, "claimer", "telegram", "DebtusBot"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestDelayUpdateTransfersWithCounterparty_emptySpaceID covers the spaceID ==
// "" validation branch (the other two empties are covered elsewhere).
func TestDelayUpdateTransfersWithCounterparty_emptySpaceID(t *testing.T) {
	ctx := context.Background()
	if err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "", "cp1", "cp2"); err == nil {
		t.Error("expected error when spaceID is empty, got nil")
	}
}

// TestDelayUpdateTransfersWithCounterparty_enqueueError covers the error return
// when EnqueueWork fails after all inputs validate.
func TestDelayUpdateTransfersWithCounterparty_enqueueError(t *testing.T) {
	ctx := context.Background()

	orig := delayer4debtus.UpdateTransfersWithCounterparty
	delayer4debtus.UpdateTransfersWithCounterparty = failingDelayer{id: "update-transfers-with-cp"}
	t.Cleanup(func() { delayer4debtus.UpdateTransfersWithCounterparty = orig })

	if err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "cp2"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestDelayUpdateTransfersWithCounterparty_success covers the success return of
// DelayUpdateTransfersWithCounterparty when all inputs are valid and the work is
// enqueued.
func TestDelayUpdateTransfersWithCounterparty_success(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)
	if err := NewTransferDal().DelayUpdateTransfersWithCounterparty(ctx, "space1", "cp1", "cp2"); err != nil {
		t.Errorf("expected nil on success, got %v", err)
	}
}

// TestLoadOutstandingTransfers_queryError covers the error return when the
// outstanding-transfers query fails. A faultQuery failingDB is passed as the
// read session (which is also the query executor).
func TestLoadOutstandingTransfers_queryError(t *testing.T) {
	ctx := context.Background()
	real := sneattesting.SetupMemoryDB(t)
	failing := failingDB{DB: real, fault: faultQuery}
	_, err := NewTransferDal().LoadOutstandingTransfers(ctx, failing, time.Now(), "u1", "", "EUR", "")
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

// TestSaveTwilioSms_getMultiError covers the GetMulti error return of
// SaveTwilioSms.
func TestSaveTwilioSms_getMultiError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	withFailingFacadeDB(t, db, faultGetMulti)
	transfer := models4debtus.NewTransfer("t1", nil)
	_, err := NewTwilioDal().SaveTwilioSms(ctx, &gotwilio.SmsResponse{Sid: "SM1"}, transfer, dto4contactus.PhoneContact{}, "u1", 1, 1)
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}
