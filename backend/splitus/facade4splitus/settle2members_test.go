package facade4splitus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
)

const (
	settleDebtorID  = "md"
	settleSponsorID = "ms"
	settleCurrency  = money.CurrencyCode("EUR")
)

// seedSettleBill stores a bill record directly (bypassing CreateBill) so tests
// can craft exact member balances.
func seedSettleBill(t *testing.T, ctx context.Context, db dal.DB, billID string, members []*briefs4splitus.BillMemberBrief) {
	t.Helper()
	data := &models4splitus.BillDbo{
		BillCommon: models4splitus.BillCommon{
			Status:        models4splitus.BillStatusOutstanding,
			SplitMode:     models4splitus.SplitModeEqually,
			CreatorUserID: "u1",
			AmountTotal:   100,
			Currency:      settleCurrency,
			Name:          "settle bill " + billID,
			SpaceID:       coretypes.SpaceID(spaceID),
			Members:       members,
		},
	}
	key := dal.NewKeyWithID(models4splitus.BillKind, billID)
	rec := dal.NewRecordWithData(key, data)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, rec)
	}); err != nil {
		t.Fatalf("failed to seed bill %s: %v", billID, err)
	}
}

// seedSettleSpace stores a splitus space with debtor/sponsor members carrying
// the given balances.
func seedSettleSpace(t *testing.T, ctx context.Context, db dal.DB, debtorBalance, sponsorBalance money.Balance) {
	t.Helper()
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	splitusSpace.Data.Members = []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleDebtorID, UserID: "u2", Name: "Debtor"}, Balance: debtorBalance},
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleSponsorID, UserID: "u1", Name: "Sponsor"}, Balance: sponsorBalance},
	}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatalf("failed to seed splitus space: %v", err)
	}
}

// settleQueryDB returns a fakeDB whose query execution yields the given bill IDs.
func settleQueryDB(t *testing.T, billIDs ...string) (dal.DB, *fakeDB) {
	t.Helper()
	memDB := sneattesting.SetupMemoryDB(t)
	keys := make([]*dal.Key, len(billIDs))
	for i, id := range billIDs {
		keys[i] = dal.NewKeyWithID(models4splitus.BillKind, id)
	}
	fdb := &fakeDB{
		DB:          memDB,
		useQuery:    true,
		queryReader: &fakeRecordsReader{keys: keys},
	}
	overrideSneatDB(t, fdb)
	return memDB, fdb
}

func TestSettle2members_GetSneatDBError(t *testing.T) {
	wantErr := errors.New("no db")
	failSneatDB(t, wantErr)
	err := Settle2members(context.Background(), spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSettle2members_QueryError(t *testing.T) {
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("query failed")
	overrideSneatDB(t, &fakeDB{DB: memDB, useQuery: true, queryErr: wantErr})
	err := Settle2members(context.Background(), spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSettle2members_ReaderError(t *testing.T) {
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("reader failed")
	overrideSneatDB(t, &fakeDB{DB: memDB, useQuery: true, queryReader: &fakeRecordsReader{nextErr: wantErr}})
	err := Settle2members(context.Background(), spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSettle2members_NoBillsFound(t *testing.T) {
	_, _ = settleQueryDB(t) // no bill IDs
	if err := Settle2members(context.Background(), spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10); err != nil {
		t.Errorf("expected nil error when no bills found, got %v", err)
	}
}

func TestSettle2members_SpaceNotFound(t *testing.T) {
	_, _ = settleQueryDB(t, "sb1")
	err := Settle2members(context.Background(), spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil {
		t.Error("expected error when splitus space is missing")
	}
}

func TestSettle2members_UnknownDebtor(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	// Space exists but has no members at all.
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := memDB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatal(err)
	}
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil || !strings.Contains(err.Error(), "unknown debtorID") {
		t.Errorf("expected unknown debtorID error, got %v", err)
	}
}

func TestSettle2members_UnknownSponsor(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	splitusSpace := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	splitusSpace.Data.Members = []briefs4splitus.SpaceSplitMember{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleDebtorID, UserID: "u2", Name: "Debtor"}},
	}
	if err := memDB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, splitusSpace.Record)
	}); err != nil {
		t.Fatal(err)
	}
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil || !strings.Contains(err.Error(), "unknown sponsorID") {
		t.Errorf("expected unknown sponsorID error, got %v", err)
	}
}

func TestSettle2members_DebtorHasNoBalanceInCurrency(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, nil, money.Balance{settleCurrency: 50})
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil || !strings.Contains(err.Error(), "debtor has no balance") {
		t.Errorf("expected debtor no balance error, got %v", err)
	}
}

func TestSettle2members_SponsorHasNoBalanceInCurrency(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, nil)
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil || !strings.Contains(err.Error(), "sponsor has no balance") {
		t.Errorf("expected sponsor no balance error, got %v", err)
	}
}

func TestSettle2members_NegativeAmount(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	// amount=-10: both balance warnings are skipped (50 < -10 is false), the
	// loop breaks into the amount < 0 branch.
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, decimal.Decimal64p2(-10))
	if err == nil || !strings.Contains(err.Error(), "amount < 0") {
		t.Errorf("expected amount < 0 error, got %v", err)
	}
}

func TestSettle2members_BillNotFound(t *testing.T) {
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "missing-bill")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil {
		t.Error("expected error for missing bill")
	}
}

// settleBillMembers returns bill members where the debtor owes `owes` and the
// sponsor paid `paid`.
func settleBillMembers(debtorPaid, debtorOwes, sponsorPaid, sponsorOwes decimal.Decimal64p2) []*briefs4splitus.BillMemberBrief {
	return []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleDebtorID, UserID: "u2", Name: "Debtor"}, Paid: debtorPaid, Owes: debtorOwes},
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleSponsorID, UserID: "u1", Name: "Sponsor"}, Paid: sponsorPaid, Owes: sponsorOwes},
	}
}

func TestSettle2members_SkipsBills(t *testing.T) {
	// Bills that hit each of the "goto nextBill" branches, followed by no
	// settleable bill, so billsToSave stays empty (the else branch logs).
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "posDebtor", "negSponsor", "noDebtor", "noSponsor")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})

	// Debtor with positive balance.
	seedSettleBill(t, ctx, memDB, "posDebtor", settleBillMembers(60, 10, 40, 90))
	// Sponsor with negative balance (debtor negative as required).
	seedSettleBill(t, ctx, memDB, "negSponsor", settleBillMembers(0, 50, 50, 100))
	// No debtor member in bill.
	seedSettleBill(t, ctx, memDB, "noDebtor", []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleSponsorID, UserID: "u1", Name: "Sponsor"}, Paid: 100, Owes: 50},
	})
	// Debtor present (negative balance) but no sponsor member.
	seedSettleBill(t, ctx, memDB, "noSponsor", []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleDebtorID, UserID: "u2", Name: "Debtor"}, Paid: 0, Owes: 50},
	})

	if err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSettle2members_SuccessFullSettlement(t *testing.T) {
	// Bill where debtor owes 50 and sponsor paid all. Balances on space match
	// the bill, so the settlement zeroes both balances (delete branches) and
	// the second bill iteration hits the amount == 0 break.
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1", "sb2")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))
	seedSettleBill(t, ctx, memDB, "sb2", settleBillMembers(0, 50, 100, 50))

	// amount=60 is more than balances; it gets capped to 50 by both balance
	// checks, the bill diff (50) fully settles, amount becomes 0 and the loop
	// breaks on the second bill.
	if err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 60); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := memDB.Get(ctx, reloaded.Record); err != nil {
		t.Fatalf("failed to reload space: %v", err)
	}
	for _, m := range reloaded.Data.GetGroupMembers() {
		if got := m.Balance[settleCurrency]; got != 0 {
			t.Errorf("expected member %s balance settled to 0, got %v", m.ID, got)
		}
	}
}

func TestSettle2members_PartialSettlement(t *testing.T) {
	// diff > amount branch: bill could settle 50 but amount is only 10.
	// Sponsor bill balance (40) is less than debtor's inverted balance (50),
	// so diff starts at sponsor's balance (the else branch) then is capped to
	// the amount. Both members keep non-zero space balances (no delete).
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 90, 50))

	if err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := memDB.Get(ctx, reloaded.Record); err != nil {
		t.Fatalf("failed to reload space: %v", err)
	}
	members := reloaded.Data.GetGroupMembers()
	byID := map[string]decimal.Decimal64p2{}
	for _, m := range members {
		byID[m.ID] = m.Balance[settleCurrency]
	}
	if byID[settleDebtorID] != -40 {
		t.Errorf("expected debtor balance -40, got %v", byID[settleDebtorID])
	}
	if byID[settleSponsorID] != 40 {
		t.Errorf("expected sponsor balance 40, got %v", byID[settleSponsorID])
	}
}

func TestSettle2members_SponsorBalanceCapsAmount(t *testing.T) {
	// Debtor balance (-60) does not cap the amount (60), but the sponsor
	// balance (50) does — covering the "sponsor balance is less than settling
	// amount" branch.
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -60}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))

	if err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 60); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded := models4splitus.NewSplitusSpaceEntry(coretypes.SpaceID(spaceID))
	if err := memDB.Get(ctx, reloaded.Record); err != nil {
		t.Fatalf("failed to reload space: %v", err)
	}
	members := reloaded.Data.GetGroupMembers()
	byID := map[string]decimal.Decimal64p2{}
	for _, m := range members {
		byID[m.ID] = m.Balance[settleCurrency]
	}
	if byID[settleDebtorID] != -10 {
		t.Errorf("expected debtor balance -10, got %v", byID[settleDebtorID])
	}
	if byID[settleSponsorID] != 0 {
		t.Errorf("expected sponsor balance 0, got %v", byID[settleSponsorID])
	}
}

func TestSettle2members_SetBillMembersError(t *testing.T) {
	// A bill member without a name makes SetBillMembers fail validation.
	ctx := context.Background()
	memDB, _ := settleQueryDB(t, "sb1")
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleDebtorID, UserID: "u2", Name: ""}, Paid: 0, Owes: 50},
		{MemberBrief: briefs4splitus.MemberBrief{ID: settleSponsorID, UserID: "u1", Name: "Sponsor"}, Paid: 100, Owes: 50},
	})
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if err == nil {
		t.Error("expected SetBillMembers validation error")
	}
}

func TestSettle2members_InsertSettlementError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("insert failed")
	fdb := &fakeDB{
		DB:          memDB,
		useQuery:    true,
		queryReader: &fakeRecordsReader{keys: []*dal.Key{dal.NewKeyWithID(models4splitus.BillKind, "sb1")}},
		wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
			return txWrap{ReadwriteTransaction: tx, insert: func(dal.Record) error { return wantErr }}
		},
	}
	overrideSneatDB(t, fdb)
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSettle2members_SetMultiError(t *testing.T) {
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("set multi failed")
	fdb := &fakeDB{
		DB:          memDB,
		useQuery:    true,
		queryReader: &fakeRecordsReader{keys: []*dal.Key{dal.NewKeyWithID(models4splitus.BillKind, "sb1")}},
		wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
			return txWrap{ReadwriteTransaction: tx, setMulti: func([]dal.Record) error { return wantErr }}
		},
	}
	overrideSneatDB(t, fdb)
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSettle2members_SetSettlementError(t *testing.T) {
	// The only tx.Set call in the success path is for the bills settlement
	// record after SetMulti, so failing Set covers that branch.
	ctx := context.Background()
	memDB := sneattesting.SetupMemoryDB(t)
	wantErr := errors.New("set failed")
	fdb := &fakeDB{
		DB:          memDB,
		useQuery:    true,
		queryReader: &fakeRecordsReader{keys: []*dal.Key{dal.NewKeyWithID(models4splitus.BillKind, "sb1")}},
		wrapTx: func(tx dal.ReadwriteTransaction) dal.ReadwriteTransaction {
			return txWrap{ReadwriteTransaction: tx, set: func(dal.Record) error { return wantErr }}
		},
	}
	overrideSneatDB(t, fdb)
	seedSettleSpace(t, ctx, memDB, money.Balance{settleCurrency: -50}, money.Balance{settleCurrency: 50})
	seedSettleBill(t, ctx, memDB, "sb1", settleBillMembers(0, 50, 100, 50))
	err := Settle2members(ctx, spaceID, settleDebtorID, settleSponsorID, settleCurrency, 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}
