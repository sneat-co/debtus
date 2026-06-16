package models4debtus

import (
	"testing"
	"time"

	"github.com/crediterra/money"
)

func contactBriefWithLastTransferAt(at time.Time) *DebtusContactBrief {
	brief := &DebtusContactBrief{Status: DebtusContactStatusActive}
	if !at.IsZero() {
		brief.Transfers = &UserContactTransfersInfo{
			Count: 1,
			Last:  LastTransfer{ID: "t1", At: at},
		}
	}
	return brief
}

func TestDebtusSpaceDbo_LatestCounterparties(t *testing.T) {
	day := func(d int) time.Time {
		return time.Date(2026, 6, d, 0, 0, 0, 0, time.UTC)
	}
	for _, tt := range []struct {
		name     string
		contacts map[string]*DebtusContactBrief
		limit    int
		wantIDs  []string
	}{
		{
			name:     "nil_contacts",
			contacts: nil,
			limit:    5,
			wantIDs:  []string{},
		},
		{
			name: "sorted_by_last_transfer_desc",
			contacts: map[string]*DebtusContactBrief{
				"c1": contactBriefWithLastTransferAt(day(1)),
				"c2": contactBriefWithLastTransferAt(day(3)),
				"c3": contactBriefWithLastTransferAt(day(2)),
			},
			limit:   5,
			wantIDs: []string{"c2", "c3", "c1"},
		},
		{
			name: "truncated_to_limit_keeping_latest",
			contacts: map[string]*DebtusContactBrief{
				"c1": contactBriefWithLastTransferAt(day(1)),
				"c2": contactBriefWithLastTransferAt(day(3)),
				"c3": contactBriefWithLastTransferAt(day(2)),
				"c4": contactBriefWithLastTransferAt(day(4)),
			},
			limit:   2,
			wantIDs: []string{"c4", "c2"},
		},
		{
			name: "contacts_without_transfers_go_last",
			contacts: map[string]*DebtusContactBrief{
				"c1": contactBriefWithLastTransferAt(time.Time{}),
				"c2": contactBriefWithLastTransferAt(day(2)),
			},
			limit:   5,
			wantIDs: []string{"c2", "c1"},
		},
		{
			name: "equal_times_tie_break_by_contact_id",
			contacts: map[string]*DebtusContactBrief{
				"c2": contactBriefWithLastTransferAt(day(1)),
				"c1": contactBriefWithLastTransferAt(day(1)),
				"c3": contactBriefWithLastTransferAt(day(1)),
			},
			limit:   5,
			wantIDs: []string{"c1", "c2", "c3"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dbo := &DebtusSpaceDbo{Contacts: tt.contacts}
			got := dbo.LatestCounterparties(tt.limit)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d contacts, want %d", len(got), len(tt.wantIDs))
			}
			for i, want := range tt.wantIDs {
				if got[i] == nil {
					t.Fatalf("got[%d] == nil, want ContactID=%s", i, want)
				}
				if got[i].ContactID != want {
					t.Errorf("got[%d].ContactID = %s, want %s", i, got[i].ContactID, want)
				}
			}
		})
	}
}

func TestDebtusSpaceDbo_TotalBalanceFromContacts(t *testing.T) {
	dbo := &DebtusSpaceDbo{Contacts: map[string]*DebtusContactBrief{
		"c1": {Balance: money.Balance{"EUR": 1025, "USD": 500}},
		"c2": {Balance: money.Balance{"EUR": 75, "USD": -500}},
	}}
	balance := dbo.TotalBalanceFromContacts()
	if got := balance["EUR"]; got != 1100 {
		t.Errorf(`balance["EUR"] = %v, want 1100`, got)
	}
	if _, hasUSD := balance["USD"]; hasUSD {
		t.Error(`balance["USD"] should be removed as it sums to zero`)
	}
}
