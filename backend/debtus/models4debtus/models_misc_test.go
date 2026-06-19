package models4debtus

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/strongo/gotwilio"
	"github.com/strongo/strongoapp/person"
)

func TestValidateString(t *testing.T) {
	if err := ValidateString("bad value", "a", []string{"a", "b"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateString("bad value", "c", []string{"a", "b"}); err == nil {
		t.Error("expected error for unknown value")
	}
}

func TestNewDebtusContactDbo(t *testing.T) {
	dbo := NewDebtusContactDbo(dto4contactus.ContactDetails{
		NameFields: person.NameFields{FirstName: "Jack", LastName: "Brown"},
	})
	if dbo.Status != "active" {
		t.Errorf("Status = %v, want active", dbo.Status)
	}
	if dbo.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if dbo.FirstName != "Jack" {
		t.Errorf("FirstName = %v, want Jack", dbo.FirstName)
	}
}

func TestNewDebtusContactKey(t *testing.T) {
	key := NewDebtusContactKey("space1", "c1")
	if key.ID != "c1" {
		t.Errorf("key.ID = %v, want c1", key.ID)
	}
}

func TestNewDebtusSpaceContacts(t *testing.T) {
	contacts := NewDebtusSpaceContacts("space1", "c1", "c2")
	if len(contacts) != 2 || contacts[0].ID != "c1" || contacts[1].ID != "c2" {
		t.Errorf("unexpected contacts: %v", contacts)
	}
	records := DebtusContactRecords(contacts)
	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}
	mustPanic(t, "empty contact ID", func() { NewDebtusSpaceContacts("space1", "") })
}

func TestNewDebtusContactRecord(t *testing.T) {
	r := NewDebtusContactRecord()
	if r.Key().Collection() != "contacts" {
		t.Errorf("collection = %v, want contacts", r.Key().Collection())
	}
}

func TestDebtusSpaceContactDbo_String(t *testing.T) {
	dbo := &DebtusSpaceContactDbo{Status: DebtusContactStatusActive}
	if s := dbo.String(); !strings.Contains(s, "active") {
		t.Errorf("unexpected String(): %v", s)
	}
}

func TestDebtusSpaceContactDbo_TransfersInfo(t *testing.T) {
	dbo := &DebtusSpaceContactDbo{}
	if dbo.GetTransfersInfo() != nil {
		t.Error("expected nil transfers info")
	}
	if err := dbo.SetTransfersInfo(UserContactTransfersInfo{Count: 2}); err != nil {
		t.Fatal(err)
	}
	if got := dbo.GetTransfersInfo(); got == nil || got.Count != 2 {
		t.Errorf("GetTransfersInfo() = %v", got)
	}
	if err := dbo.SetTransfersInfo(UserContactTransfersInfo{Count: -1}); err == nil {
		t.Error("expected validation error for negative count")
	}
}

func TestDebtusSpaceContactDbo_Info(t *testing.T) {
	dbo := &DebtusSpaceContactDbo{
		ContactDetails: dto4contactus.ContactDetails{
			NameFields: person.NameFields{FirstName: "Jack", LastName: "Brown"},
		},
	}
	info := dbo.Info("c1", "note1", "comment1")
	if info.ContactID != "c1" || info.Note != "note1" || info.Comment != "comment1" {
		t.Errorf("unexpected info: %+v", info)
	}
	if info.ContactName != "Jack Brown" {
		t.Errorf("ContactName = %q, want 'Jack Brown'", info.ContactName)
	}
}

func TestDebtusSpaceContactDbo_Validate(t *testing.T) {
	dbo := NewDebtusContactDbo(dto4contactus.ContactDetails{})
	dbo.CreatedBy = "u1"
	dbo.EmailAddressOriginal = "  Jack@Example.com "
	if err := dbo.Validate(); err != nil {
		t.Fatal(err)
	}
	if dbo.EmailAddressOriginal != "Jack@Example.com" {
		t.Errorf("EmailAddressOriginal = %q", dbo.EmailAddressOriginal)
	}
	if dbo.EmailAddress != "jack@example.com" {
		t.Errorf("EmailAddress = %q", dbo.EmailAddress)
	}
}

func TestDebtusSpaceContactDbo_MustMatchCounterparty(t *testing.T) {
	t.Run("matching_reversed_balances", func(t *testing.T) {
		dbo := &DebtusSpaceContactDbo{Balanced: money.Balanced{Balance: money.Balance{"EUR": 100}}}
		counterparty := NewDebtusSpaceContactEntry("space1", "c2", &DebtusSpaceContactDbo{
			Balanced: money.Balanced{Balance: money.Balance{"EUR": -100}},
		})
		dbo.MustMatchCounterparty(counterparty) // must not panic
	})

	t.Run("zero_balances_match", func(t *testing.T) {
		dbo := &DebtusSpaceContactDbo{}
		counterparty := NewDebtusSpaceContactEntry("space1", "c2", &DebtusSpaceContactDbo{})
		dbo.MustMatchCounterparty(counterparty) // must not panic
	})

	t.Run("mismatching_balances_panic", func(t *testing.T) {
		dbo := &DebtusSpaceContactDbo{Balanced: money.Balanced{Balance: money.Balance{"EUR": 100}}}
		counterparty := NewDebtusSpaceContactEntry("space1", "c2", &DebtusSpaceContactDbo{
			Balanced: money.Balanced{Balance: money.Balance{"EUR": 100}},
		})
		mustPanic(t, "MustMatchCounterparty", func() {
			dbo.MustMatchCounterparty(counterparty)
		})
	})
}

func TestDebtusSpaceContactDbo_BalanceWithInterest(t *testing.T) {
	dbo := &DebtusSpaceContactDbo{}
	if _, err := dbo.BalanceWithInterest(context.Background(), time.Now()); err != nil {
		t.Errorf("no transfers info should not error: %v", err)
	}
	dbo.Transfers = &UserContactTransfersInfo{
		OutstandingWithInterest: []TransferWithInterestJson{{
			TransferID: "t1",
			Currency:   money.CurrencyEUR,
			Amount:     100,
			Starts:     time.Now().Add(-24 * time.Hour),
		}},
	}
	if _, err := dbo.BalanceWithInterest(context.Background(), time.Now()); err == nil {
		t.Error("expected ErrBalanceIsZero as the balance has no EUR entry")
	}
}

func TestContactsByID(t *testing.T) {
	contacts := NewDebtusSpaceContacts("space1", "c1", "c2")
	byID := ContactsByID(contacts)
	if len(byID) != 2 || byID["c1"] == nil || byID["c2"] == nil {
		t.Errorf("unexpected map: %v", byID)
	}
}

func TestDebtusSpaceDbo_SetContacts(t *testing.T) {
	dbo := &DebtusSpaceDbo{}
	contacts := map[string]*DebtusContactBrief{
		"c1": {Balance: money.Balance{"EUR": 100}},
	}
	dbo.SetContacts(contacts)
	if len(dbo.Contacts) != 1 {
		t.Errorf("len(Contacts) = %d, want 1", len(dbo.Contacts))
	}
}

func TestDebtusSpaceDbo_BalanceWithInterest_NotImplemented(t *testing.T) {
	dbo := &DebtusSpaceDbo{}
	if _, err := dbo.BalanceWithInterest(context.Background(), time.Now()); err == nil {
		t.Error("expected 'not implemented' error")
	}
}

func TestGetDebtusSpace(t *testing.T) {
	ctx := context.Background()
	db := dalgo2memory.NewDB()
	originalGetSneatDB := facade.GetSneatDB
	facade.GetSneatDB = func(ctx context.Context) (dal.DB, error) { return db, nil }
	t.Cleanup(func() { facade.GetSneatDB = originalGetSneatDB })

	seeded := NewDebtusSpaceEntry("space1")
	seeded.Data.Contacts = map[string]*DebtusContactBrief{"c1": {Status: DebtusContactStatusActive}}
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, seeded.Record)
	}); err != nil {
		t.Fatal(err)
	}

	space := NewDebtusSpaceEntry("space1")
	if err := GetDebtusSpace(ctx, nil, space); err != nil {
		t.Fatalf("GetDebtusSpace() returned error: %v", err)
	}
	if len(space.Data.Contacts) != 1 {
		t.Errorf("len(Contacts) = %d, want 1", len(space.Data.Contacts))
	}
}

func TestNewDebtusUserEntry(t *testing.T) {
	u := NewDebtusUserEntry("u1")
	if u.Data == nil {
		t.Fatal("Data == nil")
	}
}

func TestWithTransferCounts(t *testing.T) {
	var v WithTransferCounts
	if err := v.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
	var d WithHasDueTransfers
	u := d.SetHasDueTransfers(true)
	if !d.HasDueTransfers {
		t.Error("HasDueTransfers should be true")
	}
	if u.FieldName() != "hasDueTransfers" {
		t.Errorf("update field = %v", u.FieldName())
	}
}

func TestWithGroups(t *testing.T) {
	var wg WithGroups
	if groups := wg.ActiveGroups(); len(groups) != 0 {
		t.Errorf("expected no groups, got %v", groups)
	}

	group := models4splitus.NewGroupEntry("g1", &models4splitus.GroupDbo{})
	group.Data.Name = "Group One"
	group.Data.Note = "note"

	if changed := wg.AddGroup(group, "bot1"); !changed {
		t.Error("expected changed=true when adding a new group")
	}
	groups := wg.ActiveGroups()
	if len(groups) != 1 || groups[0].ID != "g1" || groups[0].Name != "Group One" {
		t.Errorf("unexpected groups: %v", groups)
	}
	if len(groups[0].TgBots) != 1 || groups[0].TgBots[0] != "bot1" {
		t.Errorf("unexpected TgBots: %v", groups[0].TgBots)
	}

	// Same group with a changed name should report change
	group2 := models4splitus.NewGroupEntry("g1", &models4splitus.GroupDbo{})
	group2.Data.Name = "Renamed"
	if changed := wg.AddGroup(group2, ""); !changed {
		t.Error("expected changed=true when group renamed")
	}

	wg.SetActiveGroups(nil)
	if wg.GroupsCountActive != 0 || wg.GroupsJsonActive != "" {
		t.Errorf("expected groups cleared: count=%d json=%q", wg.GroupsCountActive, wg.GroupsJsonActive)
	}
}

func TestNewReceiptKeys(t *testing.T) {
	if key := NewReceiptKey("r1"); key.ID != "r1" {
		t.Errorf("key.ID = %v, want r1", key.ID)
	}
	if key := NewReceiptKey(""); key.ID != nil {
		t.Errorf("incomplete key should have nil ID, got %v", key.ID)
	}
	if key := NewReceiptIncompleteKey(); key.Collection() != ReceiptKind {
		t.Errorf("collection = %v, want %v", key.Collection(), ReceiptKind)
	}
}

func TestNewReceiptEntity(t *testing.T) {
	createdOn := general4debtus.CreatedOn{CreatedOnPlatform: "telegram", CreatedOnID: "bot1"}
	receipt := NewReceiptEntity("u1", "t1", "u2", "en", "telegram", "u2", createdOn)
	if receipt.Status != ReceiptStatusCreated {
		t.Errorf("Status = %v, want created", receipt.Status)
	}
	if receipt.CreatorUserID != "u1" || receipt.CounterpartyUserID != "u2" || receipt.TransferID != "t1" {
		t.Errorf("unexpected receipt: %+v", receipt)
	}

	mustPanic(t, "creator==counterparty", func() {
		NewReceiptEntity("u1", "t1", "u1", "en", "telegram", "u1", createdOn)
	})
	mustPanic(t, "empty transferID", func() {
		NewReceiptEntity("u1", "", "u2", "en", "telegram", "u2", createdOn)
	})
	mustPanic(t, "empty CreatedOnID", func() {
		NewReceiptEntity("u1", "t1", "u2", "en", "telegram", "u2", general4debtus.CreatedOn{CreatedOnPlatform: "telegram"})
	})
	mustPanic(t, "empty CreatedOnPlatform", func() {
		NewReceiptEntity("u1", "t1", "u2", "en", "telegram", "u2", general4debtus.CreatedOn{CreatedOnID: "bot1"})
	})
}

func TestReceiptDbo_Validate(t *testing.T) {
	newValid := func() *ReceiptDbo {
		return &ReceiptDbo{
			Status:             ReceiptStatusCreated,
			TransferID:         "t1",
			CreatorUserID:      "u1",
			CounterpartyUserID: "u2",
			Lang:               "en",
			CreatedOn:          general4debtus.CreatedOn{CreatedOnPlatform: "telegram", CreatedOnID: "bot1"},
		}
	}
	valid := newValid()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid receipt should not error: %v", err)
	}
	if valid.DtCreated.IsZero() {
		t.Error("Validate should default DtCreated")
	}

	for name, mutate := range map[string]func(r *ReceiptDbo){
		"no_transfer_id":        func(r *ReceiptDbo) { r.TransferID = "" },
		"unknown_status":        func(r *ReceiptDbo) { r.Status = "nonsense" },
		"no_creator":            func(r *ReceiptDbo) { r.CreatorUserID = "" },
		"counterparty==creator": func(r *ReceiptDbo) { r.CounterpartyUserID = r.CreatorUserID },
		"no_created_on_id":      func(r *ReceiptDbo) { r.CreatedOnID = "" },
		"no_platform":           func(r *ReceiptDbo) { r.CreatedOnPlatform = "" },
		"no_lang":               func(r *ReceiptDbo) { r.Lang = "" },
	} {
		t.Run(name, func(t *testing.T) {
			r := newValid()
			mutate(r)
			if err := r.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestNewInviteAndKey(t *testing.T) {
	if key := NewInviteKey("code1"); key.ID != "code1" {
		t.Errorf("key.ID = %v, want code1", key.ID)
	}
	invite := NewInvite("code1", &InviteData{Type: InviteTypePersonal})
	if invite.ID != "code1" || invite.Data.Type != InviteTypePersonal {
		t.Errorf("unexpected invite: %+v", invite)
	}
}

func TestNewInviteClaimHelpers(t *testing.T) {
	data := NewInviteClaimData("code1", "u1", "Telegram", "bot1")
	if data.InviteCode != "code1" || data.UserID != "u1" || data.ClaimedOn != "Telegram" || data.ClaimedVia != "bot1" {
		t.Errorf("unexpected claim data: %+v", data)
	}
	if data.DtClaimed.IsZero() {
		t.Error("DtClaimed should be set")
	}
	if key := NewInviteClaimKey(""); key.ID != nil {
		t.Errorf("empty claim ID should produce incomplete key, got ID=%v", key.ID)
	}
	if key := NewInviteClaimKey("7"); key.ID != "7" {
		t.Errorf("key.ID = %v, want \"7\"", key.ID)
	}
	if key := NewInviteClaimIncompleteKey(); key.Collection() != InviteClaimKind {
		t.Errorf("collection = %v", key.Collection())
	}
}

func TestNewFeedbackKey(t *testing.T) {
	if key := NewFeedbackKey("5"); key.ID != "5" {
		t.Errorf("key.ID = %v, want 5", key.ID)
	}
}

func TestTwilioSmsConstructors(t *testing.T) {
	if r := NewTwilioSmsRecord(); r.Key().Collection() != TwilioSmsKind {
		t.Errorf("collection = %v", r.Key().Collection())
	}

	data := &TwilioSmsDbo{}
	rec := dal.NewRecordWithData(dal.NewKeyWithID(TwilioSmsKind, "s1"), data)
	rec.SetError(nil)
	sms := NewTwilioSmsFromRecord(rec)
	if sms.ID != "s1" || sms.Data != data {
		t.Errorf("unexpected sms: %+v", sms)
	}
	smses := NewTwilioSmsFromRecords([]dal.Record{rec})
	if len(smses) != 1 || smses[0].ID != "s1" {
		t.Errorf("unexpected smses: %v", smses)
	}
}

func TestNewTwilioSmsFromSmsResponse(t *testing.T) {
	price := float32(0.05)
	resp := &gotwilio.SmsResponse{
		AccountSid: "AC1",
		To:         "+123",
		From:       "+456",
		Body:       "hello",
		Status:     "sent",
		Direction:  "outbound",
		Price:      &price,
	}
	data := NewTwilioSmsFromSmsResponse("u1", resp)
	if data.UserID != "u1" || data.To != "+123" || data.Price != price {
		t.Errorf("unexpected sms data: %+v", data)
	}
	resp.Price = nil
	data = NewTwilioSmsFromSmsResponse("u1", resp)
	if data.Price != 0 {
		t.Errorf("Price = %v, want 0", data.Price)
	}
}

func TestTransferInterestHelpers(t *testing.T) {
	ti := NewInterest("simple", 7, 7).WithMinimumPeriod(3).WithGracePeriod(2)
	if ti.InterestMinimumPeriod != 3 || ti.InterestGracePeriod != 2 {
		t.Errorf("unexpected interest: %+v", ti)
	}
	if !ti.HasInterest() {
		t.Error("HasInterest() should be true")
	}
	if NoInterest().HasInterest() {
		t.Error("NoInterest().HasInterest() should be false")
	}
	if ti.GetInterestData() != ti {
		t.Error("GetInterestData() should return itself")
	}

	mustPanic(t, "zero percent", func() { NewInterest("simple", 0, 7) })
	mustPanic(t, "negative period", func() { NewInterest("simple", 7, -1) })
	mustPanic(t, "unknown formula", func() { NewInterest("nonsense", 7, 7) })
}

func TestTransferInterest_ValidateTransferInterest(t *testing.T) {
	if err := (TransferInterest{}).ValidateTransferInterest(); err != nil {
		t.Errorf("zero interest should be valid: %v", err)
	}
	if err := (TransferInterest{InterestPeriod: -1, InterestPercent: 1}).ValidateTransferInterest(); err == nil {
		t.Error("expected error for negative period")
	}
	if err := (TransferInterest{InterestPeriod: 7}).ValidateTransferInterest(); err == nil {
		t.Error("expected error for zero percent")
	}
	if err := (TransferInterest{InterestPeriod: 7, InterestPercent: 5}).ValidateTransferInterest(); err == nil {
		t.Error("expected error for empty formula")
	}
	if err := (TransferInterest{InterestType: "nonsense", InterestPeriod: 7, InterestPercent: 5}).ValidateTransferInterest(); err == nil {
		t.Error("expected error for unknown formula")
	}
	if err := (TransferInterest{InterestType: "simple", InterestPeriod: 7, InterestPercent: 5}).ValidateTransferInterest(); err != nil {
		t.Errorf("valid interest should not error: %v", err)
	}
}

func TestTransferInterest_Validate(t *testing.T) {
	if err := (TransferInterest{InterestType: "simple"}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	for name, ti := range map[string]TransferInterest{
		"unknown_type":     {InterestType: "nonsense"},
		"negative_percent": {InterestType: "simple", InterestPercent: -1},
		"negative_grace":   {InterestType: "simple", InterestGracePeriod: -1},
		"negative_min":     {InterestType: "simple", InterestMinimumPeriod: -1},
		"negative_period":  {InterestType: "simple", InterestPeriod: -1},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ti.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}
