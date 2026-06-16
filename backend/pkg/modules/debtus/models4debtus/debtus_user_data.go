package models4debtus

import (
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/core/coremodels"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
)

type DebtusUserDbo struct { // TODO: Move back into debtus module
	money.Balanced
	WithTransferCounts
	WithHasDueTransfers
	coremodels.SmsStats
	models4splitus.BillsHolder
}

type DebtusUserEntry = record.DataWithID[coretypes.ExtID, *DebtusUserDbo]

func NewDebtusUserEntry(userID string) DebtusUserEntry {
	return dal4userus.NewUserExtEntry(userID, "debtus", new(DebtusUserDbo))
}
