package facade4debtus

import (
	"strings"
	"testing"

	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-core-modules/spaceus/dto4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
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

type testTransferSource struct{}

func (testTransferSource) PopulateTransfer(*models4debtus.TransferData) {}

func validCreateTransferRequest() CreateTransferRequest {
	return CreateTransferRequest{
		SpaceRequest: dto4spaceus.SpaceRequest{SpaceID: testSpaceID},
		Direction:    models4debtus.TransferDirectionUser2Counterparty,
		Amount:       money.NewAmount(money.CurrencyEUR, 100),
		ToContactID:  "c2",
	}
}

func TestCreateTransferRequest_Validate(t *testing.T) {
	validRequest := validCreateTransferRequest()
	if err := validRequest.Validate(); err != nil {
		t.Fatalf("valid request should not error: %v", err)
	}
	for name, mutate := range map[string]func(r *CreateTransferRequest){
		"missing_space_id":         func(r *CreateTransferRequest) { r.SpaceID = "" },
		"c2u_missing_from_contact": func(r *CreateTransferRequest) { r.Direction = models4debtus.TransferDirectionCounterparty2User },
		"u2c_missing_to_contact":   func(r *CreateTransferRequest) { r.ToContactID = "" },
		"3d_party_missing_from":    func(r *CreateTransferRequest) { r.Direction = models4debtus.TransferDirection3dParty },
		"3d_party_missing_to": func(r *CreateTransferRequest) {
			r.Direction = models4debtus.TransferDirection3dParty
			r.FromContactID = "c1"
			r.ToContactID = ""
		},
		"unknown_direction": func(r *CreateTransferRequest) { r.Direction = "nonsense" },
		"negative_amount":   func(r *CreateTransferRequest) { r.Amount.Value = -1 },
		"missing_currency":  func(r *CreateTransferRequest) { r.Amount.Currency = "" },
	} {
		t.Run(name, func(t *testing.T) {
			r := validCreateTransferRequest()
			mutate(&r)
			if err := r.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func newCreateTransferInput(creatorUserID string, from, to *models4debtus.TransferCounterpartyInfo) CreateTransferInput {
	creator := dbo4userus.NewUserEntry(creatorUserID)
	return CreateTransferInput{
		Source:      testTransferSource{},
		CreatorUser: creator,
		Request:     validCreateTransferRequest(),
		From:        from,
		To:          to,
	}
}

func TestCreateTransferInput_Direction(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}

	if got := newCreateTransferInput("u1", from, to).Direction(); got != models4debtus.TransferDirectionUser2Counterparty {
		t.Errorf("Direction() = %v, want u2c", got)
	}
	if got := newCreateTransferInput("u2", from, to).Direction(); got != models4debtus.TransferDirectionCounterparty2User {
		t.Errorf("Direction() = %v, want c2u", got)
	}

	input3d := newCreateTransferInput("u3", from, to)
	input3d.Request.BillID = "bill1"
	if got := input3d.Direction(); got != models4debtus.TransferDirection3dParty {
		t.Errorf("Direction() = %v, want 3d-party", got)
	}

	mustPanic(t, "no bill and unrelated creator", func() {
		newCreateTransferInput("u3", from, to).Direction()
	})
	mustPanic(t, "empty creator", func() {
		input := newCreateTransferInput("", from, to)
		input.Direction()
	})
}

func TestCreateTransferInput_CreatorContactID(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
	if got := newCreateTransferInput("u1", from, to).CreatorContactID(); got != "c2" {
		t.Errorf("CreatorContactID() = %v, want c2", got)
	}
	if got := newCreateTransferInput("u2", from, to).CreatorContactID(); got != "c1" {
		t.Errorf("CreatorContactID() = %v, want c1", got)
	}
	mustPanic(t, "3d-party", func() {
		newCreateTransferInput("u3", from, to).CreatorContactID()
	})
}

func TestCreateTransferOutput_Validate(t *testing.T) {
	mustPanic(t, "empty transfer ID", func() {
		CreateTransferOutput{}.Validate()
	})
	mustPanic(t, "nil transfer data", func() {
		output := CreateTransferOutput{Transfer: models4debtus.TransferEntry{}}
		output.Transfer.ID = "t1"
		output.Transfer.Data = nil
		output.Validate()
	})
	output := CreateTransferOutput{Transfer: models4debtus.NewTransfer("t1", &models4debtus.TransferData{})}
	output.Validate() // must not panic
}

func TestCreateTransferInput_Validate(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}

	valid := newCreateTransferInput("u1", from, to)
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input should not error: %v", err)
	}

	for name, mutate := range map[string]func(input *CreateTransferInput){
		"nil_source":      func(i *CreateTransferInput) { i.Source = nil },
		"empty_creator":   func(i *CreateTransferInput) { i.CreatorUser = dbo4userus.UserEntry{} },
		"invalid_request": func(i *CreateTransferInput) { i.Request.Amount.Currency = "" },
		"zero_amount":     func(i *CreateTransferInput) { i.Request.Amount.Value = 0 },
		"nil_from":        func(i *CreateTransferInput) { i.From = nil },
		"nil_to":          func(i *CreateTransferInput) { i.To = nil },
		"no_contact_or_user_ids": func(i *CreateTransferInput) {
			i.From = &models4debtus.TransferCounterpartyInfo{}
			i.To = &models4debtus.TransferCounterpartyInfo{}
		},
		"from_user_without_to": func(i *CreateTransferInput) {
			i.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
			i.To = &models4debtus.TransferCounterpartyInfo{}
		},
		"to_user_without_from": func(i *CreateTransferInput) {
			i.From = &models4debtus.TransferCounterpartyInfo{}
			i.To = &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
		},
		"same_user_ids": func(i *CreateTransferInput) {
			i.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
			i.To = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c2"}
		},
		"creator_from_without_to_contact": func(i *CreateTransferInput) {
			i.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
			i.To = &models4debtus.TransferCounterpartyInfo{UserID: "u2"}
		},
		"creator_to_without_from_contact": func(i *CreateTransferInput) {
			i.CreatorUser = dbo4userus.NewUserEntry("u2")
			i.From = &models4debtus.TransferCounterpartyInfo{UserID: "u1"}
			i.To = &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := newCreateTransferInput("u1", from, to)
			mutate(&input)
			if err := input.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestCreateTransferInput_String(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
	input := newCreateTransferInput("u1", from, to)
	if s := input.String(); !strings.Contains(s, "CreatorUserID=u1") {
		t.Errorf("unexpected String(): %v", s)
	}
}

func TestNewTransferInput(t *testing.T) {
	from := &models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1"}
	to := &models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2"}
	creator := dbo4userus.NewUserEntry("u1")

	input := NewTransferInput("test", testTransferSource{}, creator, validCreateTransferRequest(), from, to)
	if input.From != from || input.To != to {
		t.Error("unexpected input")
	}

	mustPanic(t, "invalid input", func() {
		NewTransferInput("test", nil, creator, validCreateTransferRequest(), from, to)
	})
}
