package models4debtus

import (
	"testing"

	"github.com/sneat-co/contactus/backend/dto4contactus"
	"github.com/strongo/strongoapp/person"
)

func TestContactDetailsFullname(t *testing.T) {
	lastName := "Smith"
	contactDetails := &dto4contactus.ContactDetails{
		NameFields: person.NameFields{
			LastName: lastName,
		},
	}
	if fullName := contactDetails.FullName(); fullName != lastName {
		t.Errorf("Expected %v, got %v", lastName, fullName)
	}
}
