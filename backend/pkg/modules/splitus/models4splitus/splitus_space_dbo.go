package models4splitus

import (
	"fmt"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/dal-go/dalgo/update"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/const4splitus"
	"github.com/strongo/random"
)

type SplitusSpaceDbo struct {
	BillsHolder
	Members []briefs4splitus.SpaceSplitMember `firestore:"members,omitempty"`

	BillsCountActive int    `firestore:",omitempty"`
	BillsJsonActive  string `firestore:",omitempty"`
	//
	BillSchedulesCountActive int    `firestore:",omitempty"`
	BillSchedulesJsonActive  string `firestore:",omitempty"`
}

type SplitusSpaceEntry = record.DataWithID[string, *SplitusSpaceDbo]

func NewSplitusSpaceEntry(spaceID coretypes.SpaceID) (space SplitusSpaceEntry) {
	key := dbo4spaceus.NewSpaceModuleKey(spaceID, const4splitus.ModuleID)
	return record.NewDataWithID(const4splitus.ModuleID, key, new(SplitusSpaceDbo))
}
func (v *SplitusSpaceDbo) AddOrGetMember(userID, contactID, name string) (isNew, changed bool, index int, member briefs4splitus.SpaceSplitMember, groupMembers []briefs4splitus.SpaceSplitMember) {
	if userID == "" {
		panic("userID is empty string")
	}
	if name == "" {
		panic("name is empty string")
	}
	members := v.GetMembers()
	groupMembers = v.GetGroupMembers()
	var m briefs4splitus.MemberBrief
	if index, m, isNew, changed = AddOrGetMember(members, "", userID, contactID, name); isNew {
		member = briefs4splitus.SpaceSplitMember{
			MemberBrief: m,
		}
		groupMembers = append(groupMembers, member)
		if index != len(groupMembers)-1 {
			panic("index != len(groupMembers) - 1")
		}
		changed = true
	} else /* existing member */ if member = groupMembers[index]; member.ID != m.ID {
		panic("member.ContactID != m.ContactID")
	}
	if member.ID == "" {
		panic("member.ContactID is empty string")
	}
	return
}

func AddOrGetMember(members []briefs4splitus.MemberBrief, memberID, userID, contactID, name string) (index int, member briefs4splitus.MemberBrief, isNew, changed bool) {
	if userID != "" || contactID != "" {
		for i, m := range members {
			if m.ID == memberID || m.UserID == userID {
				member = m
				if contactID != "" {
					for _, cID := range m.ContactIDs {
						if cID == contactID {
							goto contactFound
						}
					}
					m.ContactIDs = append(m.ContactIDs, contactID)
					changed = true
				contactFound:
				}
				member = m
				index = i
				return
			} else if contactID != "" {
				for _, cID := range m.ContactIDs {
					if cID == contactID {
						member = m
						index = i
						return
					}
				}
			}
		}
	}
	member = briefs4splitus.MemberBrief{
		ID:     memberID,
		Name:   name,
		UserID: userID,
	}
	if member.ID == "" {
	randomID:
		for j := 0; j < 100; j++ {
			member.ID = random.ID(const4debtus.MemberIdLen)
			for _, m := range members {
				if m.ID == member.ID {
					continue randomID
				}
			}
			break
		}
		if member.ID == "" {
			panic("Failed to generate random member ContactID")
		}
	}
	return len(members), member, true, true
}

// AddBill adds the bill to outstanding bills and, if the bill is new to this
// space, applies the bill members' balances to the matching space members.
func (v *SplitusSpaceDbo) AddBill(bill BillEntry) (changed bool, err error) {
	var isNew bool
	if isNew, changed, err = v.BillsHolder.AddBill(bill); err != nil || !isNew {
		return
	}
	groupMembers := v.GetGroupMembers()
	billMembers := bill.Data.GetBillMembers()
	for j, groupMember := range groupMembers {
		for _, billMember := range billMembers {
			if billMember.ID == groupMember.ID {
				if balance := billMember.Balance(); balance != 0 {
					if groupMember.Balance == nil {
						groupMember.Balance = make(money.Balance, 1)
					}
					groupMember.Balance[bill.Data.Currency] += balance
					groupMembers[j] = groupMember
				}
				break
			}
		}
	}
	v.SetGroupMembers(groupMembers)
	return
}

// RemoveBill removes the bill from outstanding bills and reverses the bill
// members' balances on the matching space members.
func (v *SplitusSpaceDbo) RemoveBill(bill BillEntry) (changed bool, err error) {
	outstandingBills := v.GetOutstandingBills()
	if _, ok := outstandingBills[bill.ID]; !ok {
		return
	}
	delete(outstandingBills, bill.ID)
	if err = v.SetOutstandingBills(outstandingBills); err != nil {
		return
	}
	changed = true
	groupMembers := v.GetGroupMembers()
	billMembers := bill.Data.GetBillMembers()
	for j, groupMember := range groupMembers {
		for _, billMember := range billMembers {
			if billMember.ID == groupMember.ID {
				if balance := billMember.Balance(); balance != 0 {
					if groupMember.Balance == nil {
						groupMember.Balance = make(money.Balance, 1)
					}
					groupMember.Balance[bill.Data.Currency] -= balance
					if groupMember.Balance[bill.Data.Currency] == 0 {
						delete(groupMember.Balance, bill.Data.Currency)
					}
					groupMembers[j] = groupMember
				}
				break
			}
		}
	}
	v.SetGroupMembers(groupMembers)
	return
}

func (v *SplitusSpaceDbo) GetGroupMembers() []briefs4splitus.SpaceSplitMember {
	return v.Members
}

func (v *SplitusSpaceDbo) GetGroupMemberByID(id string) (briefs4splitus.SpaceSplitMember, error) {
	if id == "" {
		return briefs4splitus.SpaceSplitMember{}, fmt.Errorf("%w: empty id", dal.ErrRecordNotFound)
	}
	for _, m := range v.GetGroupMembers() {
		if m.ID == id {
			return m, nil
		}
	}
	return briefs4splitus.SpaceSplitMember{}, fmt.Errorf("%w: unknown id="+id, dal.ErrRecordNotFound)
}

func (v *SplitusSpaceDbo) GetGroupMemberByUserID(userID string) (briefs4splitus.SpaceSplitMember, error) {
	if userID == "" {
		return briefs4splitus.SpaceSplitMember{}, fmt.Errorf("%w: empty id", dal.ErrRecordNotFound)
	}
	for _, m := range v.GetGroupMembers() {
		if m.UserID == userID {
			return m, nil
		}
	}
	return briefs4splitus.SpaceSplitMember{}, fmt.Errorf("%w: unknown userID=%s", dal.ErrRecordNotFound, userID)
}

func (v *SplitusSpaceDbo) GetMembers() (members []briefs4splitus.MemberBrief) {
	groupMembers := v.GetGroupMembers()
	members = make([]briefs4splitus.MemberBrief, len(groupMembers))
	for i, gm := range groupMembers {
		members[i] = gm.MemberBrief
	}
	return
}

func (v *SplitusSpaceDbo) GetSplitMode() SplitMode {
	if len(v.Members) == 0 {
		return SplitModeEqually
	}
	var minShares, maxShares int
	for _, m := range v.GetGroupMembers() {
		if m.Shares < minShares || minShares == 0 {
			minShares = m.Shares
		}
		if m.Shares > maxShares {
			maxShares = m.Shares
		}
	}
	if minShares == maxShares {
		return SplitModeEqually
	}
	return SplitModeShare
}

func (v *SplitusSpaceDbo) TotalShares() (n int) {
	for _, m := range v.Members {
		n += m.Shares
	}
	return
}

func (v *SplitusSpaceDbo) UserIsMember(userID string) bool {
	for _, m := range v.Members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

func (v *SplitusSpaceDbo) SetGroupMembers(members []briefs4splitus.SpaceSplitMember) (updates []update.Update) {
	v.Members = members
	return []update.Update{update.ByFieldName("members", members)}
}

//func (v *SplitusSpaceDbo) validateMembers(members []briefs4splitus.SpaceSplitMember) error {
//
//	type Empty struct {
//	}
//
//	EMPTY := Empty{}
//
//	totalBalance := make(money.Balance)
//
//	userIDs := make(map[string]Empty, len(v.Members))
//	contactIDs := make(map[string]Empty, len(v.Members))
//
//	memberIDs := make(map[string]Empty, len(v.Members))
//
//	for i, m := range members {
//		if m.ContactID == "" {
//			return fmt.Errorf("members[%d].ContactID is empty string", i)
//		}
//		if strings.TrimSpace(m.UserTitle) == "" {
//			return fmt.Errorf("members[%d].UserTitle is empty string", i)
//		}
//		if _, ok := memberIDs[m.ContactID]; ok {
//			return fmt.Errorf("members[%d]: Duplicate ContactID: %v", i, m.ContactID)
//		}
//		memberIDs[m.ContactID] = EMPTY
//		if m.UserID == "" && len(m.ContactIDs) == 0 {
//			return fmt.Errorf("members[%d]: m.UserID == 0 && len(m.ContactIDs) == 0", i)
//		}
//		if m.UserID != "" {
//			if _, ok := userIDs[m.UserID]; ok {
//				return fmt.Errorf("members[%d]: Duplicate UserID: %v", i, m.UserID)
//			}
//			userIDs[m.UserID] = EMPTY
//		} else if len(m.ContactIDs) > 0 {
//			for _, contactID := range m.ContactIDs {
//				if _, ok := contactIDs[contactID]; ok {
//					return fmt.Errorf("members[%d]: Duplicate ContactID: %v", i, contactID)
//				}
//				contactIDs[contactID] = EMPTY
//			}
//		}
//		for currency, amount := range m.Balance {
//			totalBalance[currency] += amount
//		}
//	}
//
//	// Validate total balance is 0
//	for currency, amount := range totalBalance {
//		if amount != 0 {
//			return fmt.Errorf("%w: %v=%v", ErrGroupTotalBalanceHasNonZeroValue, currency, amount)
//		}
//	}
//	return nil
//}
