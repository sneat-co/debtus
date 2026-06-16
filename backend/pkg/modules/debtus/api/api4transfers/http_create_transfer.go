package api4transfers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/apicore"
	"github.com/sneat-co/sneat-go-core/apicore/verify"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

// Seams for unit testing — override in tests to avoid real DB/network I/O.
var (
	createTransferFn   = facade4debtus.Transfers.CreateTransfer
	newTransferInputFn = func(env string, source dal4debtus.TransferSource, appUser dbo4userus.UserEntry, request facade4debtus.CreateTransferRequest, from, to *models4debtus.TransferCounterpartyInfo) facade4debtus.CreateTransferInput {
		return facade4debtus.NewTransferInput(env, source, appUser, request, from, to)
	}
)

func HandleCreateTransfer(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	var request facade4debtus.CreateTransferRequest
	apicore.HandleAuthenticatedRequestWithBody(w, r, &request, verify.DefaultJsonWithAuthRequired, http.StatusCreated,
		func(ctx facade.ContextWithUser) (interface{}, error) {
			var from, to *models4debtus.TransferCounterpartyInfo

			appUser, err := dal4userus.GetUserByID(ctx, nil, authInfo.UserID)
			if err != nil {
				return nil, err
			}

			newTransfer := newTransferInputFn(common4all.GetEnvironment(r),
				transferSourceSetToAPI{appPlatform: "api4debtus", createdOnID: r.Host},
				appUser,
				request,
				from, to,
			)

			output, err := createTransferFn(ctx, newTransfer)
			if err != nil {
				return nil, err
			}

			response := dto4debtus.CreateTransferResponse{
				Transfer: dto4debtus.TransferToDto(authInfo.UserID, output.Transfer),
			}

			var counterparty models4debtus.DebtusSpaceContactEntry
			switch output.Transfer.Data.CreatorUserID {
			case output.Transfer.Data.From().UserID:
				counterparty = output.To.DebtusContact
			case output.Transfer.Data.To().UserID:
				counterparty = output.From.DebtusContact
			default:
				return nil, fmt.Errorf("transfer creator userID=%v is neither from.UserID=%v nor to.UserID=%v",
					output.Transfer.Data.CreatorUserID, output.Transfer.Data.From().UserID, output.Transfer.Data.To().UserID)
			}
			if len(counterparty.Data.Balance) > 0 {
				response.CounterpartyBalance = counterparty.Data.Balance
			}
			return response, err
		})
}
