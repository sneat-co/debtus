package models4debtus

import (
	"testing"

	"github.com/crediterra/money"
)

func TestAddOrUpdateDebtusContact(t *testing.T) {
	t.Run("panics_on_nil_contact_data", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on nil contact data")
			}
		}()
		debtusSpace := NewDebtusSpaceEntry("space1")
		contact := DebtusSpaceContactEntry{}
		_, _ = AddOrUpdateDebtusContact(debtusSpace, contact)
	})

	t.Run("adds_contact_to_fresh_space_with_nil_contacts_map", func(t *testing.T) {
		debtusSpace := NewDebtusSpaceEntry("space1") // Contacts map is nil
		contact := NewDebtusSpaceContactEntry("space1", "c1", &DebtusSpaceContactDbo{
			Status: DebtusContactStatusActive,
		})
		brief, changed := AddOrUpdateDebtusContact(debtusSpace, contact)
		if !changed {
			t.Error("expected changed=true when adding a new contact")
		}
		if brief == nil {
			t.Fatal("expected non-nil brief")
		}
		if got := debtusSpace.Data.Contacts["c1"]; got != brief {
			t.Errorf("debtusSpace.Data.Contacts[c1] = %v, want %v", got, brief)
		}
	})

	t.Run("updates_existing_contact_when_changed", func(t *testing.T) {
		debtusSpace := NewDebtusSpaceEntry("space1")
		debtusSpace.Data.Contacts = map[string]*DebtusContactBrief{
			"c1": {Status: DebtusContactStatusActive},
		}
		contact := NewDebtusSpaceContactEntry("space1", "c1", &DebtusSpaceContactDbo{
			Status: DebtusContactStatusArchived,
		})
		if _, changed := AddOrUpdateDebtusContact(debtusSpace, contact); !changed {
			t.Error("expected changed=true when contact status changed")
		}
		if got := debtusSpace.Data.Contacts["c1"].Status; got != DebtusContactStatusArchived {
			t.Errorf("status = %v, want %v", got, DebtusContactStatusArchived)
		}
	})

	t.Run("unchanged_when_equal", func(t *testing.T) {
		debtusSpace := NewDebtusSpaceEntry("space1")
		balance := money.Balance{"EUR": 100}
		debtusSpace.Data.Contacts = map[string]*DebtusContactBrief{
			"c1": {Status: DebtusContactStatusActive, Balance: balance},
		}
		contact := NewDebtusSpaceContactEntry("space1", "c1", &DebtusSpaceContactDbo{
			Status:   DebtusContactStatusActive,
			Balanced: money.Balanced{Balance: balance},
		})
		if _, changed := AddOrUpdateDebtusContact(debtusSpace, contact); changed {
			t.Error("expected changed=false when contact brief is equal")
		}
	})
}
