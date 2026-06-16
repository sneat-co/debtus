package facade4debtus

import (
	"context"
	"reflect"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/spaceus/dal4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/const4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/strongo/logus"
)

func delayedUpdateSpaceHasDueTransfers(ctx context.Context, userID string, spaceID coretypes.SpaceID) (err error) {
	logus.Infof(ctx, "delayedUpdateSpaceHasDueTransfers(userID=%v)", userID)
	ctxWithUser := facade.NewContextWithUserID(ctx, userID)
	err = dal4spaceus.RunModuleSpaceWorkerWithUserCtx(ctxWithUser, spaceID, const4debtus.ModuleID, new(models4debtus.DebtusSpaceDbo),
		func(ctx facade.ContextWithUser, tx dal.ReadwriteTransaction, params *dal4spaceus.ModuleSpaceWorkerParams[*models4debtus.DebtusSpaceDbo]) error {
			if !params.SpaceModuleEntry.Data.HasDueTransfers {
				params.SpaceModuleUpdates = append(params.SpaceModuleUpdates, params.SpaceModuleEntry.Data.SetHasDueTransfers(true))
				params.SpaceModuleEntry.Record.MarkAsChanged()
			}
			return nil
		})
	return
}

func checkHasDueTransfers(ctx context.Context, db dal.ReadSession, userID, spaceID string) (hasDueTransfer bool, err error) {
	q := dal.From(dal.NewRootCollectionRef(models4debtus.TransfersCollection, "")).
		NewQuery().
		WhereArrayContains("BothUserIDs", userID).
		WhereField("isOutstanding", dal.Equal, true).
		WhereField("dtDueOn", dal.GreaterThen, time.Time{}).
		Limit(1).
		SelectKeysOnly(reflect.Int)

	var reader dal.RecordsReader
	if reader, err = db.ExecuteQueryToRecordsReader(ctx, q); err != nil {
		return
	}

	var transferIDs []int
	if transferIDs, err = dal.SelectAllIDs[int](ctx, reader, dal.WithLimit(q.Limit())); err != nil {
		return
	}

	return len(transferIDs) > 0, nil
}

func delayedUpdateUserHasDueTransfers(ctx context.Context, userID, spaceID string) (err error) {
	logus.Infof(ctx, "delayerUpdateUserHasDueTransfers(userID=%v)", userID)
	if userID == "" {
		logus.Errorf(ctx, "userID == 0")
		return nil
	}

	var db dal.DB
	if db, err = facade.GetSneatDB(ctx); err != nil {
		return err
	}

	debtusUser := models4debtus.NewDebtusUserEntry(userID)
	if err = db.Get(ctx, debtusUser.Record); err != nil && !dal.IsNotFound(err) {
		return err
	}

	if debtusUser.Data.HasDueTransfers {
		logus.Infof(ctx, "Already debtusUser.HasDueTransfers == %v", true)
		return nil
	}

	var hasDueTransfers bool
	if hasDueTransfers, err = checkHasDueTransfers(ctx, db, userID, spaceID); err != nil {
		return err
	}

	if !hasDueTransfers {
		logus.Infof(ctx, "No due transfers found")
		return nil
	}

	err = dal4userus.RunUserExtWorker[models4debtus.DebtusUserDbo](ctx, userID, const4debtus.ModuleID, new(models4debtus.DebtusUserDbo),
		func(ctx context.Context, tx dal.ReadwriteTransaction, params *dal4userus.UserExtWorkerParams[models4debtus.DebtusUserDbo]) error {
			if !params.UserExt.Data.HasDueTransfers {
				params.UserExtUpdates = append(params.UserExtUpdates, params.UserExt.Data.SetHasDueTransfers(true))
				logus.Infof(ctx, "User updated & saved to datastore")
			}
			return nil
		})
	return err
}
