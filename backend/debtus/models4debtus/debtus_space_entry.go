package models4debtus

import (
	"context"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
)

type DebtusSpaceEntry = record.DataWithID[string, *DebtusSpaceDbo]

func NewDebtusSpaceEntry(spaceID coretypes.SpaceID) DebtusSpaceEntry {
	key := dbo4spaceus.NewSpaceModuleKey(spaceID, const4debtus.ModuleID)
	// Balance must be a non-nil map so a first-ever transfer can call
	// money.Balanced.AddToBalance() without panicking on a nil-map write
	// (money.Balance.Add() writes b[currency]=value on the underlying map).
	dbo := &DebtusSpaceDbo{Balanced: money.Balanced{Balance: make(money.Balance)}}
	return record.NewDataWithID(const4debtus.ModuleID, key, dbo)
}

func GetDebtusSpace(ctx context.Context, tx dal.ReadSession, space DebtusSpaceEntry) (err error) {
	if tx == nil {
		if tx, err = facade.GetSneatDB(ctx); err != nil {
			return err
		}
	}
	return tx.Get(ctx, space.Record)
}
