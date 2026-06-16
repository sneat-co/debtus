package models4debtus

import (
	"context"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/const4debtus"
)

type DebtusSpaceEntry = record.DataWithID[string, *DebtusSpaceDbo]

func NewDebtusSpaceEntry(spaceID coretypes.SpaceID) DebtusSpaceEntry {
	key := dbo4spaceus.NewSpaceModuleKey(spaceID, const4debtus.ModuleID)
	return record.NewDataWithID(const4debtus.ModuleID, key, new(DebtusSpaceDbo))
}

func GetDebtusSpace(ctx context.Context, tx dal.ReadSession, space DebtusSpaceEntry) (err error) {
	if tx == nil {
		if tx, err = facade.GetSneatDB(ctx); err != nil {
			return err
		}
	}
	return tx.Get(ctx, space.Record)
}
