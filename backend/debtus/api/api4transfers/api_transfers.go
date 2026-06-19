package api4transfers

import (
	"net/http"

	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"

	"context"
)

func HandleAdminLatestTransfers(ctx context.Context, w http.ResponseWriter, r *http.Request, _ token4auth.AuthInfo) {
	transfers, err := dal4debtus.Default.Transfer.LoadLatestTransfers(ctx, 0, 20)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(([]byte)(err.Error()))
	}
	transfersToResponse(ctx, w, "", transfers, true)
}

func HandleUserTransfers(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo, user dbo4userus.UserEntry) {
	transfers, hasMore, err := dal4debtus.Default.Transfer.LoadTransfersByUserID(ctx, user.ID, 0, 100)
	if common4all.HasError(ctx, w, err, "", "", http.StatusInternalServerError) {
		return
	}
	transfersToResponse(ctx, w, user.ID, transfers, hasMore)
}

func transfersToResponse(ctx context.Context, w http.ResponseWriter, userID string, transfers []models4debtus.TransferEntry, hasMore bool) {
	common4all.JsonToResponse(ctx, w, dto4debtus.TransfersResultDto{
		Transfers:        dto4debtus.TransfersToDto(userID, transfers),
		HasMoreTransfers: hasMore,
	})
}
