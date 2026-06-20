package facade4debtus

import (
	"context"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

// linkUsersByReceiptWithinTransaction drives the receipt-linking orchestration
// through getReceiptTransferAndUsers-populated changes down into
// linkUsersWithinTransaction. With a normal already-self-linked inviter contact
// the deep linker terminates in the updateInviterContact data-integrity arm —
// but validateInput, the receipt/transfer consistency checks, the counterparty
// contact load and the new invited-contact creation all run first.
func TestLinkUsersByReceiptWithinTransaction_DeepPath(t *testing.T) {
	initDelayers4Test(t)
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	orig := dal4debtus.Default.Contact
	dal4debtus.Default.Contact = insertingContactDal{newID: "invitedNew"}
	t.Cleanup(func() { dal4debtus.Default.Contact = orig })

	changes := newReceiptDbChanges()
	changes.inviter = newLinkParty("u1", "c1")
	changes.invited = newLinkParty("u2", "c2")
	giveFamilySpace(changes.invited.user, testSpaceID)

	// transfer: created by inviter u1; counterparty (the side without a userID)
	// references inviter's own contact c1.
	td := &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      money.CurrencyEUR,
		AmountInCents: 100,
		FromJson:      `{"userID":"u1","contactID":"c1"}`,
		ToJson:        `{"contactID":"c1"}`,
	}
	changes.transfer = models4debtus.NewTransfer("t1", td)
	changes.receipt = models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		SpaceID:       testSpaceID,
		TransferID:    "t1",
		CreatorUserID: "u1",
	})

	// Seed the transfer-counterparty debtus contact so its load succeeds.
	cpDebtus := models4debtus.NewDebtusSpaceContactEntry(testSpaceID, "c1", &models4debtus.DebtusSpaceContactDbo{})
	seedRecords(t, db, cpDebtus.Record)

	linker := NewReceiptUsersLinker(changes)
	var got error
	_ = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		_, got = linker.linkUsersByReceiptWithinTransaction(ctx, ctx, tx)
		return nil
	}, dal.TxWithCrossGroup())

	if got == nil {
		t.Fatal("expected terminal data-integrity error from deep linker")
	}
}
