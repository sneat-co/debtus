package facade4debtus

import (
	"context"
	"errors"
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/errors4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
)

func setReceiptDalFake(t *testing.T) {
	t.Helper()
	old := dal4debtus.Default.Receipt
	t.Cleanup(func() { dal4debtus.Default.Receipt = old })
	dal4debtus.Default.Receipt = fakeReceiptDal{}
}

func TestAcknowledgeReceipt(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_operation_returns_error", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		_, _, _, err := AcknowledgeReceipt(ctx, facade.NewUserContext("u2"), "r1", "bogus-op")
		if !errors.Is(err, errors4debtus.ErrInvalidAcknowledgeType) {
			t.Fatalf("expected ErrInvalidAcknowledgeType, got: %v", err)
		}
	})

	t.Run("receipt_not_found_is_wrapped", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		setReceiptDalFake(t)
		_, _, _, err := AcknowledgeReceipt(ctx, facade.NewUserContext("u2"), "missing", dal4debtus.AckAccept)
		if err == nil {
			t.Fatal("expected error for missing receipt")
		}
	})

	t.Run("self_acknowledgement_is_swallowed", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setReceiptDalFake(t)

		// transfer created by u1; the acknowledging user is also u1 => self-ack.
		transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		})
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1",
			TransferID:    "t1",
		})
		creatorUser := dbo4userus.NewUserEntry("u1")
		creatorDebtusUser := models4debtus.NewDebtusUserEntry("u1")
		cpUser := dbo4userus.NewUserEntry("u2")
		cpDebtusUser := models4debtus.NewDebtusUserEntry("u2")
		seedRecords(t, db, transfer.Record, receipt.Record,
			creatorUser.Record, creatorDebtusUser.Record, cpUser.Record, cpDebtusUser.Record)

		// Acknowledged by the creator (u1) => ErrSelfAcknowledgement, swallowed to nil.
		_, _, _, err := AcknowledgeReceipt(ctx, facade.NewUserContext("u1"), "r1", dal4debtus.AckAccept)
		if err != nil {
			t.Fatalf("expected self-acknowledgement to be swallowed, got: %v", err)
		}
	})

	t.Run("already_linked_transfer_is_acknowledged", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		setReceiptDalFake(t)

		// Both transfer sides already carry user IDs => no linking needed; the
		// receipt is acknowledged, transfer status updated, and records saved.
		transfer := models4debtus.NewTransfer("t1", &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      money.CurrencyEUR,
			AmountInCents: 100,
			FromJson:      `{"userID":"u1","contactID":"c1"}`,
			ToJson:        `{"userID":"u2","contactID":"c2"}`,
		})
		receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
			CreatorUserID: "u1",
			TransferID:    "t1",
			SpaceID:       testSpaceID,
		})
		creatorUser := dbo4userus.NewUserEntry("u1")
		giveFamilySpace(creatorUser, testSpaceID)
		creatorDebtusUser := models4debtus.NewDebtusUserEntry("u1")
		cpUser := dbo4userus.NewUserEntry("u2")
		giveFamilySpace(cpUser, testSpaceID)
		cpDebtusUser := models4debtus.NewDebtusUserEntry("u2")
		debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)
		// Seed the contacts referenced by the transfer so the post-tx verify finds them.
		c1 := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})
		c2 := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c2", &models4debtus.DebtusSpaceContactDbo{})
		seedRecords(t, db, transfer.Record, receipt.Record,
			creatorUser.Record, creatorDebtusUser.Record, cpUser.Record, cpDebtusUser.Record,
			debtusSpace.Record, c1.Record, c2.Record)

		got, _, justConnected, err := AcknowledgeReceipt(ctx, facade.NewUserContext("u2"), "r1", dal4debtus.AckAccept)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Data.Status != models4debtus.ReceiptStatusAcknowledged {
			t.Errorf("receipt status = %q, want acknowledged", got.Data.Status)
		}
		if got.Data.AcknowledgedByUserID != "u2" {
			t.Errorf("AcknowledgedByUserID = %q, want u2", got.Data.AcknowledgedByUserID)
		}
		if justConnected {
			t.Error("expected justConnected=false for an already-linked transfer")
		}
	})
}
