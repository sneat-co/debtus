package models4debtus

import (
	"testing"

	"github.com/sneat-co/sneat-go-core/coretypes"
)

// These tests verify the "shape" of cross-space lending support at the model
// layer: TransferCounterpartyInfo carries its own SpaceID per side (From/To),
// independent of the other side's space, and that per-side SpaceID survives
// the FromJson/ToJson round trip used for persistence. Direction/Counterparty
// resolution is also verified to be purely UserID-based (i.e. it does not
// assume both sides live in the same space).

// TestTransferCounterpartyInfo_PerSideSpaceID verifies From and To can carry
// two different SpaceIDs at the same time — the basic precondition for a
// transfer between a lender in space A and a borrower/contact tracked in a
// different space B.
func TestTransferCounterpartyInfo_PerSideSpaceID(t *testing.T) {
	const spaceA coretypes.SpaceID = "spaceA"
	const spaceB coretypes.SpaceID = "spaceB"

	from := NewFrom("u1", spaceA, "lender note")
	to := NewTo(spaceB, "c-borrower")

	if from.SpaceID != spaceA {
		t.Errorf("from.SpaceID = %v, want %v", from.SpaceID, spaceA)
	}
	if to.SpaceID != spaceB {
		t.Errorf("to.SpaceID = %v, want %v", to.SpaceID, spaceB)
	}
	if from.SpaceID == to.SpaceID {
		t.Fatalf("test setup bug: from and to must be in different spaces")
	}
}

// TestTransferData_CrossSpace_FromToJsonRoundTrip verifies that distinct
// per-side SpaceID values are preserved when a transfer is (de)serialized via
// FromJson/ToJson, which is how From()/To() are lazily rehydrated after a
// transfer is loaded from the DB (see transfer_fromto.go). This confirms the
// stored representation can genuinely distinguish the two spaces, not just
// the in-memory struct before save.
func TestTransferData_CrossSpace_FromToJsonRoundTrip(t *testing.T) {
	const spaceA coretypes.SpaceID = "spaceA"
	const spaceB coretypes.SpaceID = "spaceB"

	original := &TransferData{
		CreatorUserID: "u1",
		from:          &TransferCounterpartyInfo{UserID: "u1", SpaceID: spaceA},
		to:            &TransferCounterpartyInfo{ContactID: "c-borrower", SpaceID: spaceB},
	}
	if err := original.onSaveSerializeJson(); err != nil {
		t.Fatalf("onSaveSerializeJson() error: %v", err)
	}
	if original.FromJson == "" || original.ToJson == "" {
		t.Fatalf("expected non-empty FromJson/ToJson, got FromJson=%q ToJson=%q", original.FromJson, original.ToJson)
	}

	// Simulate a fresh load from the DB: only the JSON columns are populated,
	// forcing From()/To() to lazily unmarshal instead of returning the
	// in-memory pointers set above.
	reloaded := &TransferData{
		CreatorUserID: "u1",
		FromJson:      original.FromJson,
		ToJson:        original.ToJson,
	}

	if got := reloaded.From().SpaceID; got != spaceA {
		t.Errorf("reloaded.From().SpaceID = %v, want %v", got, spaceA)
	}
	if got := reloaded.To().SpaceID; got != spaceB {
		t.Errorf("reloaded.To().SpaceID = %v, want %v", got, spaceB)
	}
	if reloaded.From().SpaceID == reloaded.To().SpaceID {
		t.Errorf("expected distinct SpaceIDs to survive the JSON round trip, both were %v", reloaded.From().SpaceID)
	}
}

// TestTransferData_Direction_IsSpaceAgnostic verifies that Direction(),
// Counterparty(), and CounterpartyInfoByUserID() resolve purely from
// CreatorUserID/UserID matching. None of them inspect SpaceID, so differing
// From/To SpaceIDs (the cross-space case) do not change how direction or
// counterparty resolution behaves compared to the same-space case. This is
// the part of the "cross-space" contract that already works today.
func TestTransferData_Direction_IsSpaceAgnostic(t *testing.T) {
	const spaceA coretypes.SpaceID = "spaceA"
	const spaceB coretypes.SpaceID = "spaceB"

	td := &TransferData{
		CreatorUserID: "u1",
		from:          &TransferCounterpartyInfo{UserID: "u1", SpaceID: spaceA},
		to:            &TransferCounterpartyInfo{ContactID: "c-borrower", SpaceID: spaceB},
	}

	if got := td.Direction(); got != TransferDirectionUser2Counterparty {
		t.Errorf("Direction() = %v, want %v", got, TransferDirectionUser2Counterparty)
	}
	if got := td.Counterparty(); got != td.to {
		t.Errorf("Counterparty() = %v, want the 'to' side (%v)", got, td.to)
	}
	if got := td.CounterpartyInfoByUserID("u1"); got != td.to {
		t.Errorf("CounterpartyInfoByUserID(u1) = %v, want the 'to' side", got)
	}
}

// TestNewDebtusContactKey_IsSpaceBounded verifies that the same contactID
// under two different spaceIDs resolves to two distinct storage keys. This is
// the concrete evidence that DebtusSpaceContactDbo (the counterparty side of
// a transfer, linked to a contactus space contact) is genuinely bounded per
// space, per the Sneat "/spaces/{spaceID}/ext/debtus/..." convention.
func TestNewDebtusContactKey_IsSpaceBounded(t *testing.T) {
	const spaceA coretypes.SpaceID = "spaceA"
	const spaceB coretypes.SpaceID = "spaceB"
	const contactID = "c-shared-id"

	keyInA := NewDebtusContactKey(spaceA, contactID)
	keyInB := NewDebtusContactKey(spaceB, contactID)

	if keyInA.String() == keyInB.String() {
		t.Fatalf("expected different keys for the same contactID in different spaces, got the same key: %v", keyInA)
	}
	if keyInA.ID != contactID || keyInB.ID != contactID {
		t.Errorf("expected both keys to keep contactID=%v as the leaf ID, got %v and %v", contactID, keyInA.ID, keyInB.ID)
	}
	// Both keys must be rooted under their respective space, i.e. the space
	// segment of the key path must actually differ.
	if keyInA.Parent() == nil || keyInB.Parent() == nil {
		t.Fatalf("expected keys to have a parent (space/module) segment, got keyInA.Parent()=%v keyInB.Parent()=%v", keyInA.Parent(), keyInB.Parent())
	}
}

// TestNewDebtusSpaceContactEntry_IsSpaceBounded is the entry-level analogue
// of TestNewDebtusContactKey_IsSpaceBounded: two entries built for the same
// contactID in different spaces must not collide.
func TestNewDebtusSpaceContactEntry_IsSpaceBounded(t *testing.T) {
	const spaceA coretypes.SpaceID = "spaceA"
	const spaceB coretypes.SpaceID = "spaceB"
	const contactID = "c-shared-id"

	entryInA := NewDebtusSpaceContactEntry(spaceA, contactID, nil)
	entryInB := NewDebtusSpaceContactEntry(spaceB, contactID, nil)

	if entryInA.ID != contactID || entryInB.ID != contactID {
		t.Errorf("expected ID=%v for both entries, got %v and %v", contactID, entryInA.ID, entryInB.ID)
	}
	if entryInA.Key.String() == entryInB.Key.String() {
		t.Fatalf("expected different keys for the same contactID in different spaces, got the same key: %v", entryInA.Key)
	}
}
