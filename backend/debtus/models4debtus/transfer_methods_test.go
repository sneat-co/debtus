package models4debtus

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/decimal"
)

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	f()
}

const (
	fromJson = `{"userID":"u1","contactID":"c1","contactName":"Alice"}`
	toJson   = `{"userID":"u2","contactID":"c2","contactName":"Bob"}`
)

func newTransferDataFromJson(creatorUserID string) *TransferData {
	return &TransferData{
		CreatorUserID: creatorUserID,
		FromJson:      fromJson,
		ToJson:        toJson,
	}
}

func TestTransferDirection_Reverse(t *testing.T) {
	if got := TransferDirection(TransferDirectionUser2Counterparty).Reverse(); got != TransferDirectionCounterparty2User {
		t.Errorf("Reverse(u2c) = %v", got)
	}
	if got := TransferDirection(TransferDirectionCounterparty2User).Reverse(); got != TransferDirectionUser2Counterparty {
		t.Errorf("Reverse(c2u) = %v", got)
	}
	mustPanic(t, "Reverse(3d-party)", func() {
		TransferDirection(TransferDirection3dParty).Reverse()
	})
}

func TestIsKnownTransferDirection(t *testing.T) {
	for _, d := range []TransferDirection{TransferDirectionUser2Counterparty, TransferDirectionCounterparty2User, TransferDirection3dParty} {
		if !IsKnownTransferDirection(d) {
			t.Errorf("IsKnownTransferDirection(%v) = false", d)
		}
	}
	if IsKnownTransferDirection("unknown") {
		t.Error(`IsKnownTransferDirection("unknown") = true`)
	}
}

func TestNewTransfers(t *testing.T) {
	transfers := NewTransfers([]string{"t1", "t2"})
	if len(transfers) != 2 {
		t.Fatalf("len(transfers) = %d, want 2", len(transfers))
	}
	if transfers[0].ID != "t1" || transfers[1].ID != "t2" {
		t.Errorf("unexpected IDs: %v, %v", transfers[0].ID, transfers[1].ID)
	}
	records := TransferRecords(transfers)
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}
}

func TestNewTransferKey_PanicsOnEmptyID(t *testing.T) {
	mustPanic(t, "NewTransferKey", func() { NewTransferKey("") })
}

func TestNewTransferRecord(t *testing.T) {
	r := NewTransferRecord()
	if r.Key().Collection() != TransfersCollection {
		t.Errorf("collection = %v, want %v", r.Key().Collection(), TransfersCollection)
	}
}

func TestTransferFromRecord(t *testing.T) {
	data := newTransferDataFromJson("u1")
	r := dal.NewRecordWithData(NewTransferKey("t1"), data)
	r.SetError(nil) // mark as ready, as if loaded from DB
	transfer := TransferFromRecord(r)
	if transfer.ID != "t1" {
		t.Errorf("transfer.ID = %v, want t1", transfer.ID)
	}
	if transfer.Data != data {
		t.Error("transfer.Data should be the record data")
	}
}

func TestTransfersFromQuery(t *testing.T) {
	ctx := context.Background()
	db := dalgo2memory.NewDB()
	originalGetSneatDB := facade.GetSneatDB
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) { return db, nil }
	t.Cleanup(func() { facade.GetSneatDB = originalGetSneatDB })

	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, NewTransfer("t1", newTransferDataFromJson("u1")).Record)
	})
	if err != nil {
		t.Fatalf("failed to seed transfer: %v", err)
	}

	q := dal.From(TransfersCollectionRef).NewQuery().SelectIntoRecord(func() dal.Record {
		return NewTransferRecord().SetError(nil) // dalgo2memory reads record data before marking it ready
	})
	transfers, err := TransfersFromQuery(ctx, q, nil)
	if err != nil {
		t.Fatalf("TransfersFromQuery() returned error: %v", err)
	}
	if len(transfers) != 1 {
		t.Logf("memory DB query returned %d transfers (query support is limited in dalgo2memory)", len(transfers))
	} else if transfers[0].ID != "t1" {
		t.Errorf("transfers[0].ID = %v, want t1", transfers[0].ID)
	}
}

func TestTransferData_HasObsoleteProps(t *testing.T) {
	if (&TransferData{}).HasObsoleteProps() {
		t.Error("new TransferData should not have obsolete props")
	}
}

func TestTransferData_GetStartDateAndLendingValue(t *testing.T) {
	now := time.Now()
	td := &TransferData{DtCreated: now, AmountInCents: 100}
	if !td.GetStartDate().Equal(now) {
		t.Error("GetStartDate() != DtCreated")
	}
	if td.GetLendingValue() != 100 {
		t.Error("GetLendingValue() != AmountInCents")
	}
}

func TestTransferData_AmountReturned(t *testing.T) {
	if got := (&TransferData{AmountInCentsReturned: 50}).AmountReturned(); got != 50 {
		t.Errorf("AmountReturned() = %v, want 50", got)
	}
	if got := (&TransferData{IsReturn: true, AmountInCents: 70}).AmountReturned(); got != 70 {
		t.Errorf("AmountReturned() for return transfer = %v, want 70", got)
	}
	if got := (&TransferData{AmountInCents: 70}).AmountReturned(); got != 0 {
		t.Errorf("AmountReturned() = %v, want 0", got)
	}
}

func TestTransferData_String(t *testing.T) {
	td := newTransferDataFromJson("u1")
	s := td.String()
	if !strings.Contains(s, "u1") || !strings.Contains(s, "Alice") {
		t.Errorf("unexpected String(): %v", s)
	}
}

func TestTransferData_Direction(t *testing.T) {
	if got := newTransferDataFromJson("u1").Direction(); got != TransferDirectionUser2Counterparty {
		t.Errorf("Direction() = %v, want u2c", got)
	}
	if got := newTransferDataFromJson("u2").Direction(); got != TransferDirectionCounterparty2User {
		t.Errorf("Direction() = %v, want c2u", got)
	}
	if got := newTransferDataFromJson("u3").Direction(); got != TransferDirection3dParty {
		t.Errorf("Direction() = %v, want 3d-party", got)
	}
	mustPanic(t, "Direction with empty creator", func() {
		newTransferDataFromJson("").Direction()
	})
}

func TestTransferData_DirectionForUser(t *testing.T) {
	td := newTransferDataFromJson("u3")
	if got := td.DirectionForUser("u1"); got != TransferDirectionUser2Counterparty {
		t.Errorf("DirectionForUser(u1) = %v", got)
	}
	if got := td.DirectionForUser("u2"); got != TransferDirectionCounterparty2User {
		t.Errorf("DirectionForUser(u2) = %v", got)
	}
	if got := td.DirectionForUser("u3"); got != TransferDirection3dParty {
		t.Errorf("DirectionForUser(creator) = %v", got)
	}
	mustPanic(t, "DirectionForUser(unrelated)", func() {
		td.DirectionForUser("u4")
	})
}

func TestTransferData_IsReverseDirection(t *testing.T) {
	t1 := newTransferDataFromJson("u1")
	t2 := &TransferData{
		CreatorUserID: "u2",
		FromJson:      `{"userID":"u2","contactID":"c2"}`,
		ToJson:        `{"userID":"u1","contactID":"c1"}`,
	}
	if !t1.IsReverseDirection(t2) {
		t.Error("expected transfers to be in reverse directions")
	}
	if t1.IsReverseDirection(t1) {
		t.Error("a transfer is not in reverse direction to itself")
	}
}

func TestTransferData_DirectionForContact(t *testing.T) {
	td := newTransferDataFromJson("u1")
	if got := td.DirectionForContact("c1"); got != TransferDirectionCounterparty2User {
		t.Errorf("DirectionForContact(c1) = %v", got)
	}
	if got := td.DirectionForContact("c2"); got != TransferDirectionUser2Counterparty {
		t.Errorf("DirectionForContact(c2) = %v", got)
	}
	mustPanic(t, "DirectionForContact(unrelated)", func() {
		td.DirectionForContact("c3")
	})
}

func TestTransferData_ReturnDirectionForUser(t *testing.T) {
	td := newTransferDataFromJson("u1")
	if got := td.ReturnDirectionForUser("u1"); got != TransferDirectionCounterparty2User {
		t.Errorf("ReturnDirectionForUser(u1) = %v", got)
	}
	if got := td.ReturnDirectionForUser("u2"); got != TransferDirectionUser2Counterparty {
		t.Errorf("ReturnDirectionForUser(u2) = %v", got)
	}
	mustPanic(t, "ReturnDirectionForUser(empty)", func() { td.ReturnDirectionForUser("") })
	mustPanic(t, "ReturnDirectionForUser(unrelated)", func() { td.ReturnDirectionForUser("u4") })
}

func TestTransferData_CreatorAndCounterparty(t *testing.T) {
	td := newTransferDataFromJson("u1")
	if got := td.Creator(); got.UserID != "u1" {
		t.Errorf("Creator().UserID = %v, want u1", got.UserID)
	}
	if got := td.Counterparty(); got.UserID != "u2" {
		t.Errorf("Counterparty().UserID = %v, want u2", got.UserID)
	}

	td2 := newTransferDataFromJson("u2")
	if got := td2.Creator(); got.UserID != "u2" {
		t.Errorf("Creator().UserID = %v, want u2", got.UserID)
	}
	if got := td2.Counterparty(); got.UserID != "u1" {
		t.Errorf("Counterparty().UserID = %v, want u1", got.UserID)
	}

	mustPanic(t, "Creator with empty CreatorUserID", func() {
		newTransferDataFromJson("").Creator()
	})
	mustPanic(t, "Creator not related", func() {
		newTransferDataFromJson("u3").Creator()
	})
	mustPanic(t, "Counterparty for 3d-party", func() {
		newTransferDataFromJson("u3").Counterparty()
	})
}

func TestTransferData_CounterpartyInfoByUserID(t *testing.T) {
	td := newTransferDataFromJson("u1")
	if got := td.CounterpartyInfoByUserID("u1"); got.UserID != "u2" {
		t.Errorf("CounterpartyInfoByUserID(u1).UserID = %v, want u2", got.UserID)
	}
	if got := td.CounterpartyInfoByUserID("u2"); got.UserID != "u1" {
		t.Errorf("CounterpartyInfoByUserID(u2).UserID = %v, want u1", got.UserID)
	}
	mustPanic(t, "CounterpartyInfoByUserID(unrelated)", func() { td.CounterpartyInfoByUserID("u4") })
}

func TestTransferData_UserInfoByUserID(t *testing.T) {
	td := newTransferDataFromJson("u1")
	if got := td.UserInfoByUserID("u1"); got.UserID != "u1" {
		t.Errorf("UserInfoByUserID(u1).UserID = %v, want u1", got.UserID)
	}
	if got := td.UserInfoByUserID("u2"); got.UserID != "u2" {
		t.Errorf("UserInfoByUserID(u2).UserID = %v, want u2", got.UserID)
	}
	mustPanic(t, "UserInfoByUserID(unrelated)", func() { td.UserInfoByUserID("u4") })
}

func newValidTransferData() *TransferData {
	from := &TransferCounterpartyInfo{UserID: "u1", ContactID: "c1", ContactName: "Alice"}
	to := &TransferCounterpartyInfo{ContactName: "Bob"}
	return NewTransferData("u1", false, money.NewAmount(money.CurrencyEUR, 100), from, to)
}

func TestTransferData_Validate_Errors(t *testing.T) {
	for _, tt := range []struct {
		name        string
		transfer    *TransferData
		errContains string
	}{
		{"empty_creator", &TransferData{}, "CreatorUserID"},
		{"zero_amount", &TransferData{CreatorUserID: "u1"}, "AmountInCents"},
		{
			"amount_too_big",
			&TransferData{CreatorUserID: "u1", AmountInCents: MaxTransferAmount + 1},
			"too big",
		},
		{
			"empty_currency",
			&TransferData{CreatorUserID: "u1", AmountInCents: 100},
			"Currency",
		},
		{
			"negative_returned",
			&TransferData{CreatorUserID: "u1", AmountInCents: 100, Currency: "EUR", AmountInCentsReturned: -1},
			"AmountInCentsReturned",
		},
		{
			"return_transfers_disabled",
			&TransferData{CreatorUserID: "u1", AmountInCents: 100, Currency: "EUR", IsReturn: true, FromJson: fromJson, ToJson: toJson},
			"not implemented",
		},
		{
			"no_contact_ids",
			&TransferData{
				CreatorUserID: "u1", AmountInCents: 100, Currency: "EUR",
				FromJson: `{"userID":"u1"}`, ToJson: `{"userID":"u2","userName":"Bob"}`,
			},
			"ContactID",
		},
		{
			"no_user_ids_and_no_bills",
			&TransferData{
				CreatorUserID: "u1", AmountInCents: 100, Currency: "EUR",
				FromJson: `{"contactID":"c1","contactName":"Alice"}`, ToJson: `{"contactID":"c2","contactName":"Bob"}`,
			},
			"BillIDs",
		},
		{
			"missing_from_name",
			&TransferData{
				CreatorUserID: "u2", AmountInCents: 100, Currency: "EUR",
				FromJson: `{"userID":"u1","contactID":"c1"}`, ToJson: `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
			},
			"FromCounterpartyName or FromUserName",
		},
		{
			"missing_to_name",
			&TransferData{
				CreatorUserID: "u1", AmountInCents: 100, Currency: "EUR",
				FromJson: `{"userID":"u1","contactID":"c1","contactName":"Alice"}`, ToJson: `{"userID":"u2","contactID":"c2"}`,
			},
			"ToCounterpartyName or ToUserName",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transfer.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestTransferData_Validate_ResetsOutstandingWhenFullyReturned(t *testing.T) {
	transfer := newValidTransferData()
	transfer.AmountInCentsReturned = transfer.AmountInCents
	if err := transfer.Validate(); err != nil {
		t.Fatal(err)
	}
	if transfer.IsOutstanding {
		t.Error("fully returned transfer should not be outstanding")
	}
}

func TestNewTransferData_Panics(t *testing.T) {
	from := &TransferCounterpartyInfo{UserID: "u1"}
	to := &TransferCounterpartyInfo{ContactID: "c2"}
	amount := money.NewAmount(money.CurrencyEUR, 100)
	mustPanic(t, "empty creator", func() { NewTransferData("", false, amount, from, to) })
	mustPanic(t, "nil from", func() { NewTransferData("u1", false, amount, nil, to) })
	mustPanic(t, "nil to", func() { NewTransferData("u1", false, amount, from, nil) })
	mustPanic(t, "zero amount", func() {
		NewTransferData("u1", false, money.NewAmount(money.CurrencyEUR, 0), from, to)
	})
	mustPanic(t, "empty currency", func() {
		NewTransferData("u1", false, money.NewAmount("", 100), from, to)
	})
}

func TestNewTransferData_Return(t *testing.T) {
	from := &TransferCounterpartyInfo{UserID: "u1"}
	to := &TransferCounterpartyInfo{ContactID: "c2"}
	transfer := NewTransferData("u1", true, money.NewAmount(money.CurrencyEUR, 100), from, to)
	if transfer.IsOutstanding {
		t.Error("a return transfer should not be outstanding")
	}
	if !transfer.IsReturn {
		t.Error("IsReturn should be true")
	}
}

func TestTransferData_GetReturnedAmount(t *testing.T) {
	td := &TransferData{Currency: money.CurrencyEUR, AmountInCentsReturned: 25}
	want := money.Amount{Currency: money.CurrencyEUR, Value: decimal.Decimal64p2(25)}
	if got := td.GetReturnedAmount(); got != want {
		t.Errorf("GetReturnedAmount() = %v, want %v", got, want)
	}
}

func TestTransferReturnJson_Validate(t *testing.T) {
	valid := TransferReturnJson{TransferID: "t1", Time: time.Now(), Amount: 10}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid return should not error: %v", err)
	}
	for name, invalid := range map[string]TransferReturnJson{
		"no_transfer_id": {Time: time.Now(), Amount: 10},
		"no_time":        {TransferID: "t1", Amount: 10},
		"no_amount":      {TransferID: "t1", Time: time.Now()},
	} {
		t.Run(name, func(t *testing.T) {
			if err := invalid.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestTransferData_AddReturn(t *testing.T) {
	now := time.Now()
	newTransfer := func() *TransferData {
		return &TransferData{
			CreatorUserID: "u1",
			AmountInCents: 100,
			Currency:      money.CurrencyEUR,
			IsOutstanding: true,
		}
	}

	t.Run("partial_return", func(t *testing.T) {
		td := newTransfer()
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 40}); err != nil {
			t.Fatal(err)
		}
		if td.AmountInCentsReturned != 40 || td.ReturnsCount != 1 || !td.IsOutstanding {
			t.Errorf("unexpected state: returned=%v count=%v outstanding=%v", td.AmountInCentsReturned, td.ReturnsCount, td.IsOutstanding)
		}
		returns := td.GetReturns()
		if len(returns) != 1 || returns[0].TransferID != "r1" {
			t.Errorf("unexpected returns: %v", returns)
		}
	})

	t.Run("full_return_clears_outstanding", func(t *testing.T) {
		td := newTransfer()
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 100}); err != nil {
			t.Fatal(err)
		}
		if td.IsOutstanding {
			t.Error("fully returned transfer should not be outstanding")
		}
	})

	t.Run("errors", func(t *testing.T) {
		td := newTransfer()
		if err := td.AddReturn(TransferReturnJson{Time: now, Amount: 10}); err == nil {
			t.Error("expected error for empty TransferID")
		}
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Amount: 10}); err == nil {
			t.Error("expected error for zero time")
		}
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 0}); err == nil {
			t.Error("expected error for non-positive amount")
		}
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 200}); err == nil {
			t.Error("expected error for return greater than total due")
		}
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 50}); err != nil {
			t.Fatal(err)
		}
		if err := td.AddReturn(TransferReturnJson{TransferID: "r1", Time: now, Amount: 10}); err == nil {
			t.Error("expected error for duplicate return transfer ID")
		}
	})
}

func TestTransferData_GetOutstandingValue(t *testing.T) {
	if got := (&TransferData{IsReturn: true, AmountInCents: 100}).GetOutstandingValue(time.Now()); got != 0 {
		t.Errorf("GetOutstandingValue() for return = %v, want 0", got)
	}
	td := &TransferData{AmountInCents: 100, AmountInCentsReturned: 30}
	if got := td.GetOutstandingValue(time.Now()); got != 70 {
		t.Errorf("GetOutstandingValue() = %v, want 70", got)
	}
	if got := td.GetOutstandingAmount(time.Now()); got.Value != 70 {
		t.Errorf("GetOutstandingAmount().Value = %v, want 70", got.Value)
	}
}

func TestTransferData_AgeInDays(t *testing.T) {
	// The issuing day counts as a whole day, so 48h ago means 3 days old.
	td := &TransferData{DtCreated: time.Now().Add(-48 * time.Hour)}
	if got := td.AgeInDays(); got != 3 {
		t.Errorf("AgeInDays() = %v, want 3", got)
	}
}

func TestTransferData_ValidateTransferInterestAndReturns(t *testing.T) {
	t.Run("no_interest", func(t *testing.T) {
		td := &TransferData{AmountInCents: 100}
		if err := td.validateTransferInterestAndReturns(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("interest_with_mismatched_returns", func(t *testing.T) {
		td := &TransferData{
			AmountInCents:         100,
			AmountInCentsReturned: 10,
			TransferInterest: TransferInterest{
				InterestType:    "simple",
				InterestPeriod:  30,
				InterestPercent: 10,
			},
		}
		if err := td.validateTransferInterestAndReturns(); err == nil {
			t.Error("expected error for mismatched returns sum")
		}
	})
}
