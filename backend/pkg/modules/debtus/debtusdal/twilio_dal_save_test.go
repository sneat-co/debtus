package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/gotwilio"
)

func TestTwilioDalGae_SaveTwilioSms(t *testing.T) {
	ctx := context.Background()
	const userID = "u1"
	price := float32(0.05)
	smsResponse := &gotwilio.SmsResponse{
		Sid:    "SM123",
		To:     "+353857000000",
		From:   "+15005550006",
		Body:   "You owe me",
		Status: "queued",
		Price:  &price,
	}
	phoneContact := dto4contactus.PhoneContact{PhoneNumber: 353857000000}

	seed := func(t *testing.T, db dal.DB, transfer models4debtus.TransferEntry, existingSms *models4debtus.TwilioSmsDbo) {
		t.Helper()
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			debtusUser := models4debtus.NewDebtusUserEntry(userID)
			if err := tx.Set(ctx, debtusUser.Record); err != nil {
				return err
			}
			if err := tx.Set(ctx, transfer.Record); err != nil {
				return err
			}
			if existingSms != nil {
				if err := tx.Set(ctx, models4debtus.NewTwilioSms(smsResponse.Sid, existingSms).Record); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to seed records: %v", err)
		}
	}

	t.Run("saves_new_sms_and_increments_stats", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		transfer := models4debtus.NewTransfer("t1", nil)
		seed(t, db, transfer, nil)

		twilioSms, err := NewTwilioDal().SaveTwilioSms(ctx, smsResponse, transfer, phoneContact, userID, 4242, 7)
		if err != nil {
			t.Fatalf("SaveTwilioSms() returned error: %v", err)
		}
		if twilioSms.ID != "SM123" {
			t.Errorf("twilioSms.ID = %q, want SM123", twilioSms.ID)
		}

		saved := models4debtus.NewTwilioSms("SM123", nil)
		if err = db.Get(ctx, saved.Record); err != nil {
			t.Fatalf("saved sms not found: %v", err)
		}
		if saved.Data.Body != "You owe me" {
			t.Errorf("saved sms Body = %q", saved.Data.Body)
		}
		if saved.Data.CreatorTgChatID != 4242 || saved.Data.CreatorTgSmsStatusMessageID != 7 {
			t.Errorf("saved sms tg fields = (%d, %d), want (4242, 7)", saved.Data.CreatorTgChatID, saved.Data.CreatorTgSmsStatusMessageID)
		}

		debtusUser := models4debtus.NewDebtusUserEntry(userID)
		if err = db.Get(ctx, debtusUser.Record); err != nil {
			t.Fatalf("debtus user not found: %v", err)
		}
		if debtusUser.Data.SmsCount != 1 {
			t.Errorf("debtusUser.SmsCount = %d, want 1", debtusUser.Data.SmsCount)
		}
		if debtusUser.Data.SmsCost != float64(price) {
			t.Errorf("debtusUser.SmsCost = %v, want %v", debtusUser.Data.SmsCost, float64(price))
		}

		savedTransfer := models4debtus.NewTransfer("t1", nil)
		if err = db.Get(ctx, savedTransfer.Record); err != nil {
			t.Fatalf("transfer not found: %v", err)
		}
		if savedTransfer.Data.SmsCount != 1 {
			t.Errorf("transfer.SmsCount = %d, want 1", savedTransfer.Data.SmsCount)
		}
	})

	t.Run("noop_when_sms_already_saved", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		transfer := models4debtus.NewTransfer("t1", nil)
		seed(t, db, transfer, &models4debtus.TwilioSmsDbo{TwilioSmsData: models4debtus.TwilioSmsData{UserID: userID, Body: "original"}})

		if _, err := NewTwilioDal().SaveTwilioSms(ctx, smsResponse, transfer, phoneContact, userID, 4242, 7); err != nil {
			t.Fatalf("SaveTwilioSms() returned error: %v", err)
		}

		saved := models4debtus.NewTwilioSms("SM123", nil)
		if err := db.Get(ctx, saved.Record); err != nil {
			t.Fatalf("saved sms not found: %v", err)
		}
		if saved.Data.Body != "original" {
			t.Errorf("existing sms was overwritten: Body = %q, want original", saved.Data.Body)
		}
		debtusUser := models4debtus.NewDebtusUserEntry(userID)
		if err := db.Get(ctx, debtusUser.Record); err != nil {
			t.Fatalf("debtus user not found: %v", err)
		}
		if debtusUser.Data.SmsCount != 0 {
			t.Errorf("debtusUser.SmsCount = %d, want 0 (no double count)", debtusUser.Data.SmsCount)
		}
	})

	t.Run("error_when_user_missing", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		transfer := models4debtus.NewTransfer("t1", nil)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, transfer.Record)
		})
		if err != nil {
			t.Fatalf("failed to seed transfer: %v", err)
		}
		if _, err = NewTwilioDal().SaveTwilioSms(ctx, smsResponse, transfer, phoneContact, userID, 4242, 7); err == nil {
			t.Error("expected error when debtus user is missing, got nil")
		}
	})
}
