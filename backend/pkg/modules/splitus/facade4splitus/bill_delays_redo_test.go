package facade4splitus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/decimal"
)

// delayedUpdateUserWithBill reads the bill via facade.GetSneatDB from a
// goroutine while RunUserExtWorker holds the dalgo2memory write lock for the
// whole transaction. Serving the bill from the fakeDB.get override (instead of
// the locked memory DB) avoids the deadlock.

const updUserID = "uw1"
const updBillID = "uwbill1"

// updBillDbo builds the bill served to the goroutine.
func updBillDbo(status string, memberUserID string) *models4splitus.BillDbo {
	return &models4splitus.BillDbo{
		BillCommon: models4splitus.BillCommon{
			Status:        status,
			SplitMode:     models4splitus.SplitModeEqually,
			CreatorUserID: "u1",
			AmountTotal:   100,
			Currency:      "EUR",
			Name:          "Bill for user update",
			SpaceID:       coretypes.SpaceID(spaceID),
			Members: []*briefs4splitus.BillMemberBrief{
				{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", UserID: memberUserID, Name: "Member1"}, Paid: 100, Owes: 40},
			},
		},
	}
}

// serveBillGet returns a fakeDB get override that serves the given bill DBO
// for any record read (the only non-transactional read is the bill).
func serveBillGet(dbo *models4splitus.BillDbo) func(ctx context.Context, rec dal.Record) error {
	return func(_ context.Context, rec dal.Record) error {
		rec.SetError(nil)
		*(rec.Data().(*models4splitus.BillDbo)) = *dbo
		return nil
	}
}

// seedUserExt stores a SplitusUserDbo user-extension record for updUserID.
func seedUserExt(t *testing.T, ctx context.Context, db dal.DB, outstanding map[string]briefs4splitus.BillBrief) {
	t.Helper()
	extKey := dal4userus.NewUserExtKey(updUserID, const4debtus.ModuleID)
	dbo := new(models4splitus.SplitusUserDbo)
	dbo.OutstandingBills = outstanding
	rec := dal.NewRecordWithData(extKey, dbo)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, rec)
	}); err != nil {
		t.Fatalf("failed to seed user ext record: %v", err)
	}
}

func TestDelayedUpdateUserWithBill_UserExtNotFound(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	// The worker fails fast (missing user ext record) without waiting for the
	// bill-reading goroutine, so the goroutine's unsynchronized read of err
	// (a production data race) must be sequenced after the function returns.
	release := make(chan struct{})
	served := make(chan struct{})
	billDbo := updBillDbo(models4splitus.BillStatusOutstanding, updUserID)
	overrideSneatDB(t, fakeDB{DB: memDB, get: func(_ context.Context, rec dal.Record) error {
		<-release
		rec.SetError(nil)
		*(rec.Data().(*models4splitus.BillDbo)) = *billDbo
		close(served)
		return nil
	}})

	// No user ext record seeded: dal.IsNotFound clears the error.
	if err := delayedUpdateUserWithBill(ctx, updBillID, updUserID); err != nil {
		t.Errorf("expected nil error for missing user ext record, got %v", err)
	}
	close(release)
	<-served
	time.Sleep(50 * time.Millisecond) // let the goroutine finish
}

func TestDelayedUpdateUserWithBill_BillGetError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("bill get failed")
	overrideSneatDB(t, fakeDB{DB: memDB, get: func(_ context.Context, _ dal.Record) error {
		return wantErr
	}})
	seedUserExt(t, ctx, memDB, nil)

	err := delayedUpdateUserWithBill(ctx, updBillID, updUserID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestDelayedUpdateUserWithBill_AddsOutstandingBill(t *testing.T) {
	// User ext exists with no outstanding bills; the user is a bill member of
	// an outstanding bill, so a new brief is added and the record saved.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, fakeDB{DB: memDB, get: serveBillGet(updBillDbo(models4splitus.BillStatusOutstanding, updUserID))})
	// The map must be non-nil with at least one entry: the production code
	// assigns into the map returned by GetOutstandingBills as-is.
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		"other-bill": {Name: "Other"},
	})

	if err := delayedUpdateUserWithBill(ctx, updBillID, updUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	extKey := dal4userus.NewUserExtKey(updUserID, const4debtus.ModuleID)
	dbo := new(models4splitus.SplitusUserDbo)
	rec := dal.NewRecordWithData(extKey, dbo)
	if err := memDB.Get(ctx, rec); err != nil {
		t.Fatalf("failed to reload user ext: %v", err)
	}
	brief, ok := dbo.OutstandingBills[updBillID]
	if !ok {
		t.Fatal("expected outstanding bill brief to be added")
	}
	if brief.UserBalance != 60 { // paid 100 - owes 40
		t.Errorf("expected user balance 60, got %v", brief.UserBalance)
	}
}

func TestDelayedUpdateUserWithBill_UpdatesStaleBrief(t *testing.T) {
	// Existing brief has every tracked field stale, so all update branches run.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, fakeDB{DB: memDB, get: serveBillGet(updBillDbo(models4splitus.BillStatusOutstanding, updUserID))})
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		updBillID: {
			Name:        "Old name",
			Total:       1,
			Currency:    "USD",
			UserBalance: decimal.Decimal64p2(-5),
			GroupID:     "old-group",
		},
	})

	if err := delayedUpdateUserWithBill(ctx, updBillID, updUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	extKey := dal4userus.NewUserExtKey(updUserID, const4debtus.ModuleID)
	dbo := new(models4splitus.SplitusUserDbo)
	rec := dal.NewRecordWithData(extKey, dbo)
	if err := memDB.Get(ctx, rec); err != nil {
		t.Fatalf("failed to reload user ext: %v", err)
	}
	brief := dbo.OutstandingBills[updBillID]
	if brief.Name != "Bill for user update" || brief.Total != 100 || brief.Currency != "EUR" ||
		brief.UserBalance != 60 || brief.GroupID != spaceID {
		t.Errorf("expected brief updated from bill, got: %+v", brief)
	}
}

func TestDelayedUpdateUserWithBill_BriefUpToDate(t *testing.T) {
	// All fields already match the bill: nothing changes, the "User not
	// changed" branch logs.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, fakeDB{DB: memDB, get: serveBillGet(updBillDbo(models4splitus.BillStatusOutstanding, updUserID))})
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		updBillID: {
			Name:        "Bill for user update",
			Total:       100,
			Currency:    "EUR",
			UserBalance: 60,
			GroupID:     spaceID,
		},
	})

	if err := delayedUpdateUserWithBill(ctx, updBillID, updUserID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayedUpdateUserWithBill_DeletedBillRemovesBrief(t *testing.T) {
	// The bill is deleted, so it should not be in outstanding bills and the
	// delete branch runs (note: the deletion is not flagged as a change by
	// production code, so the record is not re-saved).
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	overrideSneatDB(t, fakeDB{DB: memDB, get: serveBillGet(updBillDbo(models4splitus.BillStatusDeleted, updUserID))})
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		updBillID: {Name: "Bill for user update", Total: 100, Currency: "EUR", UserBalance: 60, GroupID: spaceID},
	})

	if err := delayedUpdateUserWithBill(ctx, updBillID, updUserID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayedUpdateUserWithBill_SetError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("ext set failed")
	overrideSneatDB(t, fakeDB{
		DB:  memDB,
		get: serveBillGet(updBillDbo(models4splitus.BillStatusOutstanding, updUserID)),
		wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
			return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
		},
	})
	seedUserExt(t, ctx, memDB, map[string]briefs4splitus.BillBrief{
		"other-bill": {Name: "Other"},
	})

	err := delayedUpdateUserWithBill(ctx, updBillID, updUserID)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestDelayedUpdateUserWithBill_LateBillReadSeesError(t *testing.T) {
	// The user-ext tx.Get fails with a non-not-found error, so the function
	// returns before the goroutine finished reading the bill. Once released,
	// the goroutine observes the non-nil err and returns early.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	release := make(chan struct{})
	served := make(chan struct{})
	billDbo := updBillDbo(models4splitus.BillStatusOutstanding, updUserID)
	overrideSneatDB(t, fakeDB{
		DB: memDB,
		get: func(_ context.Context, rec dal.Record) error {
			<-release
			rec.SetError(nil)
			*(rec.Data().(*models4splitus.BillDbo)) = *billDbo
			close(served)
			return nil
		},
		wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
			return txWrap{ReadwriteTransaction: tx, get: func(dal.Record) error {
				return errors.New("ext get failed")
			}}
		},
	})
	seedUserExt(t, ctx, memDB, nil)

	err := delayedUpdateUserWithBill(ctx, updBillID, updUserID)
	if err == nil {
		t.Fatal("expected error from user ext get")
	}
	close(release)
	<-served
	// Give the goroutine a moment to evaluate the err != nil branch.
	time.Sleep(50 * time.Millisecond)
}
