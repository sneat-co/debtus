package delayed4debtus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
)

// TestDelayedOnReceiptSentSuccess_statusNotSent reaches the main transaction body
// (line 74) where Status != ReceiptStatusSent. Line 74 itself panics because
// transferEntity is a local zero-value TransferData whose Counterparty() panics
// (documented production bug on_receipt_sent_success.go:74). We use recover to confirm
// the path is reached; lines 74-87 cannot be covered without fixing the production bug.
func TestDelayedOnReceiptSentSuccess_statusNotSentPanics(t *testing.T) {
	ctx := context.Background()
	db := setupReceiptSentSuccessTest(t)

	const (
		receiptID  = "r-not-sent"
		transferID = "t-not-sent"
	)
	receiptDbo := &models4debtus.ReceiptDbo{
		Status:     models4debtus.ReceiptStatusSending,
		TransferID: transferID,
	}
	seedRecords(t, db,
		models4debtus.NewReceipt(receiptID, receiptDbo).Record,
		models4debtus.NewTransfer(transferID, &models4debtus.TransferData{
			CreatorUserID: "u1",
			FromJson:      `{"userID":"u1"}`,
			ToJson:        `{"userID":"u2"}`,
		}).Record,
	)
	stubEditTgMessageText(t, nil)

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_ = DelayedOnReceiptSentSuccess(ctx, time.Now(), receiptID, transferID, 123, 1, "bot1", "en-UK")
	}()
	if !panicked {
		t.Error("expected panic from transferEntity.Counterparty() (production bug); got none")
	}
}

// TestDelayedOnReceiptSentSuccess_badRequestLogLevels covers the age-based log-level
// selection (lines 106-109) for the swallowed "Bad Request ... not found" case.
func TestDelayedOnReceiptSentSuccess_badRequestLogLevels(t *testing.T) {
	cases := []struct {
		name      string
		dtCreated time.Duration // offset from now
	}{
		{"between_1h_and_24h_infof", -2 * time.Hour},
		{"between_1m_and_1h_warningf", -10 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := setupReceiptSentSuccessTest(t)

			receiptID := "r-loglevel-" + tc.name
			transferID := "t-loglevel-" + tc.name
			receiptDbo := &models4debtus.ReceiptDbo{
				Status:     models4debtus.ReceiptStatusSent, // already sent → returns nil, then edit fails
				TransferID: transferID,
				DtCreated:  time.Now().Add(tc.dtCreated),
			}
			seedRecords(t, db,
				models4debtus.NewReceipt(receiptID, receiptDbo).Record,
				models4debtus.NewTransfer(transferID, &models4debtus.TransferData{
					CreatorUserID: "u1",
					FromJson:      `{"userID":"u1"}`,
					ToJson:        `{"userID":"u2"}`,
				}).Record,
			)

			stubEditTgMessageText(t, errors.New("Bad Request: message to edit not found"))

			err := DelayedOnReceiptSentSuccess(ctx, time.Now(), receiptID, transferID, 123, 1, "bot1", "en-UK")
			if err != nil {
				t.Errorf("expected nil (Bad Request swallowed), got: %v", err)
			}
		})
	}
}
