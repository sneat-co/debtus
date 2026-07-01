package api4transfers

import (
	"context"
	"net/http"
	"slices"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/api/api4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-go-core/facade"
)

// Seams for unit testing — override in tests to avoid real DB/network I/O.
var (
	getTransferByIDFn4transfers          = facade4debtus.Transfers.GetTransferByID
	checkTransferCreatorNameFn4transfers = facade4debtus.CheckTransferCreatorNameAndFixIfNeeded
)

// HandleGetTransfer requires an authenticated caller and only returns the
// transfer if that caller is one of its two parties (or an admin).
// Fable refactoring (SEC-1): route was previously registered without any
// auth wrapper, allowing anyone to read any transfer by ID (IDOR).
func HandleGetTransfer(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	if transferID := common4all.GetStrID(ctx, w, r, "id"); transferID == "" {
		return
	} else {
		transfer, err := getTransferByIDFn4transfers(ctx, nil, transferID)
		if common4all.HasError(ctx, w, err, models4debtus.TransfersCollection, transferID, http.StatusBadRequest) {
			return
		}

		if !authInfo.IsAdmin && !slices.Contains([]string{transfer.Data.From().UserID, transfer.Data.To().UserID}, authInfo.UserID) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			if err = checkTransferCreatorNameFn4transfers(ctx, tx, transfer); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			return err
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		record := api4debtus.NewReceiptTransferDto(ctx, transfer)

		common4all.JsonToResponse(ctx, w, &record)
	}
}

type transferSourceSetToAPI struct {
	appPlatform string
	createdOnID string
}

func (s transferSourceSetToAPI) PopulateTransfer(t *models4debtus.TransferData) {
	t.CreatedOnPlatform = s.appPlatform
	t.CreatedOnID = s.createdOnID
}
