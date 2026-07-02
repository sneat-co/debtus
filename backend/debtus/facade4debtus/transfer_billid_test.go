package facade4debtus

import (
	"context"
	"testing"

	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dbo4contactus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
)

// TestCreateTransfer_PersistsBillID is a regression test for splitus's
// settle-up single-source-of-truth read-through (splitus#ac:settle-up-single-
// source-of-truth): a transfer created with a non-empty CreateTransferRequest.
// BillID must carry that ID onto the stored TransferData.BillIDs so a later
// read (e.g. splitus's getBillTransfers) can find "which Debtus transfers
// back this bill" without splitus persisting any settled/unsettled state of
// its own. Before this fix, CreateTransferRequest.BillID was read only for
// the Direction() 3rd-party heuristic and was never written to TransferData.
func TestCreateTransfer_PersistsBillID(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	u1 := dbo4userus.NewUserEntry("u1")
	contactDbo := &dbo4contactus.ContactDbo{}
	contactDbo.UserID = "u1"
	contact := dal4contactus.NewContactEntryWithData(testSpaceID, "c2", contactDbo)
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)

	seedRecords(t, db, u1.Record, contact.Record, debtusSpace.Record)

	input := sameSpaceCreateTransferInput()
	input.Request.BillID = "bill123"

	output, err := Transfers.CreateTransfer(ctx, input)
	if err != nil {
		t.Fatalf("CreateTransfer() error: %v", err)
	}

	if got := output.Transfer.Data.BillIDs; len(got) != 1 || got[0] != "bill123" {
		t.Errorf("Transfer.Data.BillIDs = %v, want [bill123]", got)
	}
}

// TestCreateTransfer_NoBillID_LeavesBillIDsEmpty documents the complementary
// case: an ordinary (non-split) transfer has no BillID, and must not gain a
// spurious BillIDs entry.
func TestCreateTransfer_NoBillID_LeavesBillIDsEmpty(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	u1 := dbo4userus.NewUserEntry("u1")
	contactDbo := &dbo4contactus.ContactDbo{}
	contactDbo.UserID = "u1"
	contact := dal4contactus.NewContactEntryWithData(testSpaceID, "c2", contactDbo)
	debtusSpace := models4debtus.NewDebtusSpaceEntry(testSpaceID)

	seedRecords(t, db, u1.Record, contact.Record, debtusSpace.Record)

	input := sameSpaceCreateTransferInput()

	output, err := Transfers.CreateTransfer(ctx, input)
	if err != nil {
		t.Fatalf("CreateTransfer() error: %v", err)
	}

	if got := output.Transfer.Data.BillIDs; len(got) != 0 {
		t.Errorf("Transfer.Data.BillIDs = %v, want empty", got)
	}
}
