package dto4debtus

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
)

func TestTransferDto_String_MarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("forced marshal error")
	}
	dto := TransferDto{Id: "t1"}
	s := dto.String()
	if s != "forced marshal error" {
		t.Errorf("expected error message, got %q", s)
	}
}

func TestContactDto_MarshalJSON(t *testing.T) {
	dto := TransferDto{Id: "t1", Amount: money.Amount{Currency: money.CurrencyEUR, Value: 100}}
	s := dto.String()
	if !strings.Contains(s, "t1") {
		t.Errorf("unexpected String(): %v", s)
	}
}

func TestNewContactDto(t *testing.T) {
	dto := NewContactDto(models4debtus.TransferCounterpartyInfo{
		ContactID:   "c1",
		UserID:      "u1",
		ContactName: "Alice",
		Comment:     "comment",
	})
	if dto.ID != "c1" || dto.UserID != "u1" || dto.Name != "Alice" || dto.Comment != "comment" {
		t.Errorf("unexpected dto: %+v", dto)
	}

	noName := NewContactDto(models4debtus.TransferCounterpartyInfo{
		ContactName: dto4contactus.NoName,
	})
	if noName.Name != "" {
		t.Errorf("NoName should map to empty name, got %q", noName.Name)
	}
}

func newTransferEntry(id, creatorUserID string) models4debtus.TransferEntry {
	return models4debtus.NewTransfer(id, &models4debtus.TransferData{
		CreatorUserID: creatorUserID,
		DtCreated:     time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		AmountInCents: 100,
		Currency:      money.CurrencyEUR,
		FromJson:      `{"userID":"u1","contactID":"c1","contactName":"Alice"}`,
		ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
	})
}

func TestTransferToDto(t *testing.T) {
	transfer := newTransferEntry("t1", "u1")

	t.Run("for_from_user_only_to_is_set", func(t *testing.T) {
		dto := TransferToDto("u1", transfer)
		if dto.From != nil {
			t.Errorf("From should be nil for the from-user, got %+v", dto.From)
		}
		if dto.To == nil || dto.To.UserID != "u2" {
			t.Errorf("unexpected To: %+v", dto.To)
		}
	})

	t.Run("for_to_user_only_from_is_set", func(t *testing.T) {
		dto := TransferToDto("u2", transfer)
		if dto.To != nil {
			t.Errorf("To should be nil for the to-user, got %+v", dto.To)
		}
		if dto.From == nil || dto.From.UserID != "u1" {
			t.Errorf("unexpected From: %+v", dto.From)
		}
	})

	t.Run("for_other_users_both_are_set", func(t *testing.T) {
		for _, userID := range []string{"0", "u3"} {
			dto := TransferToDto(userID, transfer)
			if dto.From == nil || dto.To == nil {
				t.Errorf("both From and To should be set for userID=%q", userID)
			}
		}
	})

	t.Run("copies_transfer_fields", func(t *testing.T) {
		dto := TransferToDto("u1", transfer)
		if dto.Id != "t1" || dto.CreatorUserID != "u1" || dto.Amount.Value != 100 {
			t.Errorf("unexpected dto: %+v", dto)
		}
	})
}

func TestTransfersToDto(t *testing.T) {
	transfers := []models4debtus.TransferEntry{
		newTransferEntry("t1", "u1"),
		newTransferEntry("t2", "u2"),
	}
	dtos := TransfersToDto("u1", transfers)
	if len(dtos) != 2 || dtos[0].Id != "t1" || dtos[1].Id != "t2" {
		t.Errorf("unexpected dtos: %v", dtos)
	}
}
