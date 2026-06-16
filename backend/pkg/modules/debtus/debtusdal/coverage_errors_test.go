package debtusdal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/gotwilio"
)

// withFailingFacadeDB overrides facade.GetSneatDB to return a failingDB wrapping
// the supplied real DB, restoring the original on cleanup.
func withFailingFacadeDB(t *testing.T, real dal.DB, fault txFault) {
	t.Helper()
	original := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return failingDB{DB: real, fault: fault}, nil
	}
	t.Cleanup(func() { facade.GetSneatDB = original })
}

// withErroringFacadeDB makes facade.GetSneatDB itself return an error.
func withErroringFacadeDB(t *testing.T) {
	t.Helper()
	original := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return nil, errInjected
	}
	t.Cleanup(func() { facade.GetSneatDB = original })
}

func TestGetFeedbackByID_facadeDBError(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	withErroringFacadeDB(t)
	if _, err := NewFeedbackDal().GetFeedbackByID(ctx, nil, "1"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestGetFeedbackByID_getError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// Seed so Get would otherwise succeed; the failing tx forces Get to error.
	failing := failingDB{DB: db, fault: faultGet}
	err := failing.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, e := NewFeedbackDal().GetFeedbackByID(ctx, tx, "1")
		return e
	})
	if !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestGetInvite_facadeDBError(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	withErroringFacadeDB(t)
	if _, err := NewInviteDal().GetInvite(ctx, nil, "code1"); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestCreateReceipt_setDebtusUserError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// Seed debtus user so GetDebtusUser succeeds, then tx.Set fails.
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewDebtusUserEntry("u1").Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withFailingFacadeDB(t, db, faultSet)
	if _, err := NewReceiptDal().CreateReceipt(ctx, &models4debtus.ReceiptDbo{CreatorUserID: "u1"}); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestCreateReceipt_insertError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewDebtusUserEntry("u1").Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withFailingFacadeDB(t, db, faultInsert)
	if _, err := NewReceiptDal().CreateReceipt(ctx, &models4debtus.ReceiptDbo{CreatorUserID: "u1"}); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestMarkReceiptAsSent_getTransferError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	// Seed a receipt so GetReceiptByID succeeds; then the transfer Get fails.
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{TransferID: "t1"}).Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withFailingFacadeDB(t, db, faultGet)
	if err := NewReceiptDal().MarkReceiptAsSent(ctx, "r1", "t1", time.Now()); !errors.Is(err, errInjected) {
		t.Errorf("expected errInjected, got %v", err)
	}
}

func TestSaveTwilioSms_transferNotFound(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	const userID = "u1"
	// Seed debtus user only — transfer is absent, hitting the "transfer not found" branch.
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewDebtusUserEntry(userID).Record)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	price := float32(0.05)
	smsResponse := &gotwilio.SmsResponse{Sid: "SM999", Price: &price}
	transfer := models4debtus.NewTransfer("missing-transfer", nil)
	_, err := NewTwilioDal().SaveTwilioSms(ctx, smsResponse, transfer, dto4contactus.PhoneContact{}, userID, 1, 1)
	if err == nil {
		t.Error("expected error when transfer is missing, got nil")
	}
}

func TestQueryErrorBranches(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		call func(db dal.DB) error
	}{
		{"GetSentReminderIDsByTransferID", func(db dal.DB) error {
			_, err := NewReminderDal().GetSentReminderIDsByTransferID(ctx, db, "t1")
			return err
		}},
		{"GetActiveReminderIDsByTransferID", func(db dal.DB) error {
			_, err := NewReminderDal().GetActiveReminderIDsByTransferID(ctx, db, "t1")
			return err
		}},
		{"GetLastTwilioSmsesForUser", func(db dal.DB) error {
			_, err := NewTwilioDal().GetLastTwilioSmsesForUser(ctx, db, "u1", "", 10)
			return err
		}},
		{"GetContactsWithDebts", func(db dal.DB) error {
			_, err := NewContactDal().GetContactsWithDebts(ctx, db, "space1", "u1")
			return err
		}},
		{"GetLatestContacts", func(db dal.DB) error {
			_, err := NewContactDal().GetLatestContacts(ctx, "u1", db, "space1", 10, 0)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			real := sneattesting.SetupMemoryDB(t)
			db := failingDB{DB: real, fault: faultQuery}
			if err := tc.call(db); !errors.Is(err, errInjected) {
				t.Errorf("%s: expected errInjected, got %v", tc.name, err)
			}
		})
	}
}
