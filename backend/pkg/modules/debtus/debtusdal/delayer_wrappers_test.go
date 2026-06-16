package debtusdal

import (
	"context"
	"testing"
	"time"
)

// Tests for thin enqueue-wrapper functions that call a registered delayer.
// Only delayers registered by RegisterDelayers4Debtus can be exercised here;
// delayer vars that are declared but never registered (CreateAndSendReceipt...,
// CreateReminderForTransferUser, DiscardRemindersForTransfers,
// UpdateTransferWithCreatorReceiptTgMessageID) stay nil and would panic.

func TestDelayedMarkReceiptAsSent_enqueue(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)
	err := NewReceiptDal().DelayedMarkReceiptAsSent(ctx, "r1", "t1", time.Now())
	if err != nil {
		t.Errorf("DelayedMarkReceiptAsSent() returned error: %v", err)
	}
}

func TestDelayUpdateTransfersWithCreatorName(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)
	err := DelayUpdateTransfersWithCreatorName(ctx, "u1")
	if err != nil {
		t.Errorf("DelayUpdateTransfersWithCreatorName() returned error: %v", err)
	}
}

func TestDelayUpdateInviteClaimedCount(t *testing.T) {
	ctx := context.Background()
	setupDelayers(t)
	err := DelayUpdateInviteClaimedCount(ctx, "42")
	if err != nil {
		t.Errorf("DelayUpdateInviteClaimedCount() returned error: %v", err)
	}
}

func TestDelayCreateReminderForTransferUser_panics_on_empty_transferID(t *testing.T) {
	ctx := context.Background()
	// No delayers needed — panics before reaching EnqueueWork.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when transferID is empty")
		}
	}()
	_ = NewReminderDal().DelayCreateReminderForTransferUser(ctx, "", "u1")
}

func TestDelayCreateReminderForTransferUser_panics_on_empty_userID(t *testing.T) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when userID is empty")
		}
	}()
	_ = NewReminderDal().DelayCreateReminderForTransferUser(ctx, "t1", "")
}
