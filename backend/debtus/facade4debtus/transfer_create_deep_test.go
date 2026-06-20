package facade4debtus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dbo4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/strongoapp/person"
)

// fakeTransferDal is a minimal dal4debtus.TransferDal whose LoadOutstandingTransfers
// returns a configurable set, used to drive checkOutstandingTransfersForReturns.
type fakeTransferDal struct {
	outstanding []models4debtus.TransferEntry
	loadErr     error
}

func (f fakeTransferDal) LoadOutstandingTransfers(_ context.Context, _ dal.ReadSession, _ time.Time, _, _ string, _ money.CurrencyCode, _ models4debtus.TransferDirection) ([]models4debtus.TransferEntry, error) {
	return f.outstanding, f.loadErr
}
func (fakeTransferDal) GetTransfersByID(_ context.Context, _ dal.ReadSession, _ []string) ([]models4debtus.TransferEntry, error) {
	return nil, nil
}
func (fakeTransferDal) LoadTransfersByUserID(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
	return nil, false, nil
}
func (fakeTransferDal) LoadTransfersByContactID(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
	return nil, false, nil
}
func (fakeTransferDal) LoadTransferIDsByContactID(_ context.Context, _ string, _ int, _ string) ([]string, string, error) {
	return nil, "", nil
}
func (fakeTransferDal) LoadOverdueTransfers(_ context.Context, _ dal.ReadSession, _ string, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, nil
}
func (fakeTransferDal) LoadDueTransfers(_ context.Context, _ dal.ReadSession, _ string, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, nil
}
func (fakeTransferDal) LoadLatestTransfers(_ context.Context, _, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, nil
}
func (fakeTransferDal) DelayUpdateTransferWithCreatorReceiptTgMessageID(_ context.Context, _ string, _ string, _, _ int64) error {
	return nil
}
func (fakeTransferDal) DelayUpdateTransfersWithCounterparty(_ context.Context, _ coretypes.SpaceID, _, _ string) error {
	return nil
}
func (fakeTransferDal) DelayUpdateTransfersOnReturn(_ context.Context, _ string, _ []dal4debtus.TransferReturnUpdate) error {
	return nil
}

func setTransferDal(t *testing.T, d dal4debtus.TransferDal) {
	t.Helper()
	orig := dal4debtus.Default.Transfer
	dal4debtus.Default.Transfer = d
	t.Cleanup(func() { dal4debtus.Default.Transfer = orig })
}

// outstandingTransfer builds an outstanding transfer entry with the given
// outstanding amount, owed by u1 to c2 (so the reverse direction is a return).
func outstandingTransfer(id string, amount money.Amount) models4debtus.TransferEntry {
	td := &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      amount.Currency,
		AmountInCents: amount.Value,
		IsOutstanding: true,
		FromJson:      `{"userID":"u1","contactID":"cu1"}`,
		ToJson:        `{"contactID":"c2"}`,
	}
	td.DtCreated = time.Now().Add(-time.Hour)
	return models4debtus.NewTransfer(id, td)
}

func returnInput() CreateTransferInput {
	u1 := dbo4userus.NewUserEntry("u1")
	u1.Data.Names = &person.NameFields{FirstName: "Alice"}
	return CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: u1,
		Request: CreateTransferRequest{
			SpaceRequest: validCreateTransferRequest().SpaceRequest,
			Direction:    models4debtus.TransferDirectionUser2Counterparty,
			Amount:       money.NewAmount(money.CurrencyEUR, 100),
			ToContactID:  "c2",
			IsReturn:     true,
		},
		From: &models4debtus.TransferCounterpartyInfo{UserID: "u1"},
		To:   &models4debtus.TransferCounterpartyInfo{ContactID: "c2"},
	}
}

// ============================================================================
// checkOutstandingTransfersForReturns
// ============================================================================

func TestCheckOutstandingTransfersForReturns(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("load_error_is_wrapped", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setTransferDal(t, fakeTransferDal{loadErr: errors.New("boom")})
		_, err := Transfers.checkOutstandingTransfersForReturns(ctx, now, returnInput())
		if err == nil {
			t.Fatal("expected wrapped load error")
		}
	})

	t.Run("is_return_with_no_outstanding_returns_sentinel", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setTransferDal(t, fakeTransferDal{outstanding: nil})
		_, err := Transfers.checkOutstandingTransfersForReturns(ctx, now, returnInput())
		if !errors.Is(err, ErrNoOutstandingTransfers) {
			t.Fatalf("expected ErrNoOutstandingTransfers, got: %v", err)
		}
	})

	t.Run("exact_amount_match_selects_single_transfer", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setTransferDal(t, fakeTransferDal{outstanding: []models4debtus.TransferEntry{
			outstandingTransfer("t1", money.NewAmount(money.CurrencyEUR, 100)),
		}})
		ids, err := Transfers.checkOutstandingTransfersForReturns(ctx, now, returnInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "t1" {
			t.Errorf("expected [t1], got %v", ids)
		}
	})

	t.Run("partial_amounts_accumulate_across_transfers", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		// Two transfers of 60 each; the requested return is 100, so both are picked
		// (neither is an exact match), accumulating 60+60 >= 100.
		setTransferDal(t, fakeTransferDal{outstanding: []models4debtus.TransferEntry{
			outstandingTransfer("t1", money.NewAmount(money.CurrencyEUR, 60)),
			outstandingTransfer("t2", money.NewAmount(money.CurrencyEUR, 60)),
		}})
		ids, err := Transfers.checkOutstandingTransfersForReturns(ctx, now, returnInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("expected 2 transfer IDs, got %v", ids)
		}
	})

	t.Run("not_enough_outstanding_logs_warning_but_returns_partial", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		// Single transfer of 30; requested 100 — assignedValue(30) < amount(100).
		setTransferDal(t, fakeTransferDal{outstanding: []models4debtus.TransferEntry{
			outstandingTransfer("t1", money.NewAmount(money.CurrencyEUR, 30)),
		}})
		ids, err := Transfers.checkOutstandingTransfersForReturns(ctx, now, returnInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "t1" {
			t.Errorf("expected [t1], got %v", ids)
		}
	})
}

// ============================================================================
// CreateTransfer — reachable early branches (the deep transactional body builds
// records with nil keys and cannot complete on the in-memory DB; documented).
// ============================================================================

func TestCreateTransfer_EarlyBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("return_to_unknown_transfer_errors", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		input := returnInput()
		input.Request.ReturnToTransferID = "missing-transfer"
		_, err := Transfers.CreateTransfer(ctx, input)
		if err == nil {
			t.Fatal("expected error for unknown ReturnToTransferID")
		}
	})

	t.Run("return_to_already_returned_transfer", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		// Seed a transfer whose outstanding value is zero (fully returned).
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			IsOutstanding: true,
			FromJson:      `{"userID":"u1","contactID":"cu1"}`,
			ToJson:        `{"contactID":"c2"}`,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		_ = td.AddReturn(models4debtus.TransferReturnJson{TransferID: "tOld", Time: time.Now(), Amount: 100})
		seeded := models4debtus.NewTransfer("t1", td)
		seedRecords(t, db, seeded.Record)

		input := returnInput()
		input.Request.ReturnToTransferID = "t1"
		_, err := Transfers.CreateTransfer(ctx, input)
		if !errors.Is(err, ErrDebtAlreadyReturned) {
			t.Fatalf("expected ErrDebtAlreadyReturned, got: %v", err)
		}
	})

	t.Run("return_partial_greater_than_outstanding", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		// Outstanding 50, requesting return of 100 (not equal to AmountInCents=50,
		// so it does NOT get capped) => ErrPartialReturnGreaterThenOutstanding.
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 50,
			IsOutstanding: true,
			FromJson:      `{"userID":"u1","contactID":"cu1"}`,
			ToJson:        `{"contactID":"c2"}`,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		seeded := models4debtus.NewTransfer("t1", td)
		seedRecords(t, db, seeded.Record)

		input := returnInput()
		input.Request.ReturnToTransferID = "t1"
		input.Request.Amount = money.NewAmount(money.CurrencyEUR, 100)
		_, err := Transfers.CreateTransfer(ctx, input)
		if !errors.Is(err, ErrPartialReturnGreaterThenOutstanding) {
			t.Fatalf("expected ErrPartialReturnGreaterThenOutstanding, got: %v", err)
		}
	})

	t.Run("return_to_transfer_currency_mismatch_panics", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      "USD", // different from input EUR
			AmountInCents: 100,
			IsOutstanding: true,
			FromJson:      `{"userID":"u1","contactID":"cu1"}`,
			ToJson:        `{"contactID":"c2"}`,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		seeded := models4debtus.NewTransfer("t1", td)
		seedRecords(t, db, seeded.Record)

		input := returnInput()
		input.Request.ReturnToTransferID = "t1"
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for currency mismatch")
			}
		}()
		_, _ = Transfers.CreateTransfer(ctx, input)
	})
}

// guard against unused imports if a branch above is trimmed.
var (
	_ = dal4contactus.NewContactEntryWithData
	_ = dbo4contactus.ContactDbo{}
)
