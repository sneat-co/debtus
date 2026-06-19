package models4splitus

import (
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/debtus/backend/splitus/const4splitus"
)

type SplitusUserDbo struct {
	BillsHolder
}

type SplitusUserEntry = record.DataWithID[coretypes.ExtID, *SplitusUserDbo]

func NewSplitusUserEntry(userID string) SplitusUserEntry {
	return dal4userus.NewUserExtEntry(userID, const4splitus.ModuleID, new(SplitusUserDbo))
}
