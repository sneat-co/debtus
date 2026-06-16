package facade4splitus

import (
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/update"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
)

// Test seams: package-level vars wrapping hard-wired dependencies so that
// unit tests can substitute behaviors (mostly failure modes) that are
// otherwise unreachable. Production behavior is unchanged.
var (
	applyBillBalanceDifference = func(dbo *models4splitus.SplitusSpaceDbo, currency money.CurrencyCode, diff briefs4splitus.BillBalanceDifference) (bool, error) {
		return dbo.ApplyBillBalanceDifference(currency, diff)
	}
	billAddOrGetMember = func(dbo *models4splitus.BillDbo, groupMemberID, userID, contactID, name string) (isNew, changed bool, index int, member *briefs4splitus.BillMemberBrief, billMembers []*briefs4splitus.BillMemberBrief) {
		return dbo.AddOrGetMember(groupMemberID, userID, contactID, name)
	}
	splitusSpaceAddBill = func(dbo *models4splitus.SplitusSpaceDbo, bill models4splitus.BillEntry) (bool, error) {
		return dbo.AddBill(bill)
	}
	splitusSpaceRemoveBill = func(dbo *models4splitus.SplitusSpaceDbo, bill models4splitus.BillEntry) (bool, error) {
		return dbo.RemoveBill(bill)
	}
	setUserOutstandingBills = func(dbo *models4splitus.SplitusUserDbo, bills map[string]briefs4splitus.BillBrief) error {
		return dbo.SetOutstandingBills(bills)
	}
	setSpaceGroupMembers = func(dbo *models4splitus.SplitusSpaceDbo, members []briefs4splitus.SpaceSplitMember) []update.Update {
		return dbo.SetGroupMembers(members)
	}
	newBillKey = func() (*dal.Key, error) {
		return dal.NewKeyWithOptions(models4splitus.BillKind, dal.WithRandomStringID(dal.RandomLength(const4debtus.BillIdLen)))
	}
	getBillEntryByID = GetBillByID
	isValidBillSplit = models4splitus.IsValidBillSplit
)
