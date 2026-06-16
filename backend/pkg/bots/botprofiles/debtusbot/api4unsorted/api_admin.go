package api4unsorted

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/update"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/core/queues"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
	"github.com/strongo/validation"
)

func HandleAdminFindUser(ctx context.Context, w http.ResponseWriter, r *http.Request, _ token4auth.AuthInfo) {

	if userID := r.URL.Query().Get("userID"); userID != "" {
		appUser := dbo4userus.NewUserEntry(userID)
		if err := getUser(ctx, nil, appUser); err != nil {
			logus.Errorf(ctx, fmt.Errorf("failed to get user by userID=%s: %w", userID, err).Error())
		} else {
			common4all.JsonToResponse(ctx, w, []dto4debtus.ApiUserDto{{ID: userID, Name: appUser.Data.GetFullName()}})
		}
		return
	} else {
		// Search by Telegram username was removed: the legacy implementation
		// queried a nonexistent field and never returned results (R10).
		common4all.BadRequestMessage(ctx, w, "search by tgUser is not supported; use userID")
	}
}

func HandleAdminMergeUserContacts(ctx context.Context, w http.ResponseWriter, r *http.Request, _ token4auth.AuthInfo) {
	keepID := common4all.GetStrID(ctx, w, r, "keepID")
	if keepID == "" {
		return
	}
	deleteID := common4all.GetStrID(ctx, w, r, "deleteID")
	if deleteID == "" {
		return
	}
	spaceID := coretypes.SpaceID(common4all.GetStrID(ctx, w, r, "spaceID"))
	if spaceID == "" {
		common4all.BadRequestError(ctx, w, validation.NewErrRequestIsMissingRequiredField("spaceID"))
		return
	}

	logus.Infof(ctx, "keepID: %s, deleteID: %s", keepID, deleteID)

	if err := runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {

		contacts, err := getContactsByIDs(ctx, tx, spaceID, []string{keepID, deleteID})
		if err != nil {
			return err
		}
		if len(contacts) < 2 {
			return fmt.Errorf("len(contacts):%d < 2", len(contacts))
		}
		contactToKeep := contacts[0]
		contactToDelete := contacts[1]
		if contactToKeep.Data.UserID != contactToDelete.Data.UserID {
			return errors.New("contactToKeep.UserID != contactToDelete.UserID")
		}
		if contactToDelete.Data.UserID != "" && contactToKeep.Data.UserID == "" {
			return errors.New("contactToDelete.CounterpartyUserID != 0 && contactToKeep.CounterpartyUserID == 0")
		}
		contactusSpace := dal4contactus.NewContactusSpaceEntry(spaceID)

		if err = getContactusSpace(ctx, tx, contactusSpace); err != nil {
			return err
		}

		if contactusSpace.Data.HasContact(deleteID) {
			u := contactusSpace.Data.RemoveContact(deleteID)
			if err = tx.Update(ctx, contactusSpace.Key, []update.Update{u}); err != nil {
				return err
			}
		}
		if err := delayChangeTransfersCounterparty.EnqueueWork(ctx, delaying.With(queues.QueueSupport, "changeTransfersCounterparty", 0), deleteID, keepID, ""); err != nil {
			return err
		}
		if err := tx.Delete(ctx, models4debtus.NewDebtusContactKey(spaceID, deleteID)); err != nil {
			return err
		} else {
			logus.Warningf(ctx, "DebtusSpaceContactEntry %s has been deleted from DB (non revocable)", deleteID)
		}
		return nil
	}); err != nil {
		common4all.ErrorAsJson(ctx, w, http.StatusInternalServerError, err)
		return
	}
}

func DelayedChangeTransfersCounterparty(ctx context.Context, oldID, newID int64, cursor string) (err error) {
	logus.Debugf(ctx, "delayedChangeTransfersCounterparty(oldID=%d, newID=%d)", oldID, newID)

	var q = dal.From(dal.NewRootCollectionRef(models4debtus.TransfersCollection, "")).
		NewQuery().
		WhereArrayContains("BothCounterpartyIDs", oldID).
		Limit(100).
		SelectKeysOnly(reflect.Int)

	var db dal.DB

	if db, err = facade.GetSneatDB(ctx); err != nil {
		return err
	}

	var reader dal.RecordsReader
	if reader, err = executeQueryToRecordsReader(ctx, db, q); err != nil {
		return err
	}
	transferIDs, err := dal.SelectAllIDs[int](ctx, reader, dal.WithLimit(q.Limit()))
	if err != nil {
		return err
	}

	logus.Infof(ctx, "Loaded %d transferIDs", len(transferIDs))
	args := make([][]interface{}, len(transferIDs))
	for i, id := range transferIDs {
		args[i] = []interface{}{id, oldID, newID, ""}
	}
	return delayChangeTransferCounterparty.EnqueueWorkMulti(ctx, delaying.With(queues.QueueSupport, "changeTransferCounterparty", 0), args...)
}

func DelayedChangeTransferCounterparty(ctx context.Context, spaceID coretypes.SpaceID, transferID, oldID, newID string, cursor string) (err error) {
	logus.Debugf(ctx, "delayedChangeTransferCounterparty(spaceID=%s, oldID=%s, newID=%s, cursor=%s)", spaceID, oldID, newID, cursor)
	if _, err = getDebtusSpaceContactByID(ctx, nil, spaceID, newID); err != nil {
		return err
	}
	err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		transfer, err := getTransferByID(ctx, tx, transferID)
		if err != nil {
			return err
		}
		changed := false
		for i, contactID := range transfer.Data.BothCounterpartyIDs {
			if contactID == oldID {
				transfer.Data.BothCounterpartyIDs[i] = newID
				changed = true
				break
			}
		}
		if changed {
			if from := transfer.Data.From(); from.ContactID == oldID {
				from.ContactID = newID
			} else if to := transfer.Data.To(); to.ContactID == oldID {
				to.ContactID = newID
			}
			err = saveTransfer(ctx, tx, transfer)
		}
		return err
	})
	return err
}
