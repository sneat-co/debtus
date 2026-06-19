package models4debtus

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
)

type DebtusSpaceDbo struct {
	money.Balanced

	dbmodels.WithLastCurrencies
	WithTransferCounts
	WithHasDueTransfers

	Contacts map[string]*DebtusContactBrief `firestore:"contacts,omitempty"` // key is contact ID
}

func (v *DebtusSpaceDbo) TotalBalanceFromContacts() (balance money.Balance) {
	balance = make(money.Balance)

	add := func(contacts map[string]*DebtusContactBrief) {
		for _, contact := range contacts {
			for currency, cv := range contact.Balance {
				if v := balance[currency] + cv; v == 0 {
					delete(balance, currency)
				} else {
					balance[currency] = v
				}
			}
		}
	}
	add(v.Contacts)

	return
}

func (v *DebtusSpaceDbo) SetContacts(contacts map[string]*DebtusContactBrief) {
	{ // store to internal properties
		v.Contacts = contacts
	}

	{ // update balance
		balance := make(money.Balance)
		for _, contact := range contacts {
			for c, v := range contact.Balance {
				if newVal := balance[c] + v; newVal == 0 {
					delete(balance, c)
				} else {
					balance[c] = newVal
				}
			}
		}
		//if err := v.SetBalance(balance); err != nil {
		//	panic(err)
		//}
	}
}

// LatestCounterparties returns up to limit contacts ordered by the time of
// their last transfer, most recent first. Contacts without transfers go last.
func (v *DebtusSpaceDbo) LatestCounterparties(limit int) (contacts []*DebtusContactBriefWithContactID) {
	contacts = make([]*DebtusContactBriefWithContactID, 0, len(v.Contacts))
	for id, debtusContactBrief := range v.Contacts {
		contacts = append(contacts, &DebtusContactBriefWithContactID{DebtusContactBrief: *debtusContactBrief, ContactID: id})
	}
	lastTransferAt := func(c *DebtusContactBriefWithContactID) time.Time {
		if c.Transfers == nil {
			return time.Time{}
		}
		return c.Transfers.Last.At
	}
	sort.Slice(contacts, func(i, j int) bool {
		t1, t2 := lastTransferAt(contacts[i]), lastTransferAt(contacts[j])
		if t1.Equal(t2) {
			return contacts[i].ContactID < contacts[j].ContactID
		}
		return t1.After(t2)
	})
	if len(contacts) > limit {
		contacts = contacts[:limit]
	}
	return
}

func (v *DebtusSpaceDbo) BalanceWithInterest(_ context.Context, periodEnds time.Time) (balance money.Balance, err error) {
	err = errors.New("method BalanceWithInterest() is not implemented")
	//if v.TransfersWithInterestCount == 0 {
	//	balance = v.Balance()
	//} else if v.TransfersWithInterestCount > 0 {
	//	//var (
	//	//	userBalance Balance
	//	//)
	//	//userBalance = v.Balance()
	//	balance = make(money.Balance, v.BalanceCount)
	//	for _, contact := range v.Contacts() {
	//		var contactBalance money.Balance
	//		if contactBalance, err = contact.BalanceWithInterest(ctx, periodEnds); err != nil {
	//			err = fmt.Errorf("failed to get balance with interest for user's contact JSON %v: %w", contact.ContactID, err)
	//			return
	//		}
	//		for currency, value := range contactBalance {
	//			balance[currency] += value
	//		}
	//	}
	//	//if len(balance) != v.BalanceCount { // Theoretically can be eliminated by interest
	//	//	panic(fmt.Sprintf("len(balance) != v.BalanceCount: %v != %v", len(balance), v.BalanceCount))
	//	//}
	//	//for ctx, v := range balance { // It can be less if we have different interest condition for 2 contacts different direction!!!
	//	//	if tv := userBalance[ctx]; v < tv {
	//	//		panic(fmt.Sprintf("For currency %v balance with interest is less than total balance: %v < %v", ctx, v, tv))
	//	//	}
	//	//}
	//} else {
	//	panic(fmt.Sprintf("TransfersWithInterestCount > 0: %v", v.TransfersWithInterestCount))
	//}
	return
}
