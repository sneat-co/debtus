package api4transfers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/apicore"
	"github.com/sneat-co/sneat-go-core/apicore/verify"
	"github.com/sneat-co/sneat-go-core/facade"
)

// Seams for unit testing — override in tests to avoid real DB/network I/O.
var (
	createTransferFn   = facade4debtus.Transfers.CreateTransfer
	newTransferInputFn = func(env string, source dal4debtus.TransferSource, appUser dbo4userus.UserEntry, request facade4debtus.CreateTransferRequest, from, to *models4debtus.TransferCounterpartyInfo) facade4debtus.CreateTransferInput {
		return facade4debtus.NewTransferInput(env, source, appUser, request, from, to)
	}
)

// ErrTransferDirectionNotSupportedByAPI is returned when the HTTP
// create-transfer endpoint is asked to create a transfer whose direction it
// does not know how to resolve into From/To counterparties (currently only
// TransferDirectionUser2Counterparty and TransferDirectionCounterparty2User
// are supported — the "I lent" / "I borrowed" cases driven by the
// authenticated caller).
var ErrTransferDirectionNotSupportedByAPI = errors.New("transfer direction not supported by this endpoint")

// buildTransferCounterparties builds the From/To TransferCounterpartyInfo
// pair for a CreateTransferRequest, using selfUserID (the authenticated
// caller) for whichever side represents them.
//
// The counterparty side's ContactID is resolved against
// request.CounterpartySpaceIDOrDefault(): ordinarily the counterparty's
// contact record lives in the same space as the creator (SpaceID), but for
// cross-space lending the caller can supply CounterpartySpaceID explicitly.
func buildTransferCounterparties(selfUserID string, request facade4debtus.CreateTransferRequest) (from, to *models4debtus.TransferCounterpartyInfo, err error) {
	counterpartySpaceID := request.CounterpartySpaceIDOrDefault()
	self := &models4debtus.TransferCounterpartyInfo{
		UserID:  selfUserID,
		SpaceID: request.SpaceID,
		Note:    request.Note,
	}
	switch request.Direction {
	case models4debtus.TransferDirectionUser2Counterparty:
		from = self
		to = &models4debtus.TransferCounterpartyInfo{ContactID: request.ToContactID, SpaceID: counterpartySpaceID}
	case models4debtus.TransferDirectionCounterparty2User:
		to = self
		from = &models4debtus.TransferCounterpartyInfo{ContactID: request.FromContactID, SpaceID: counterpartySpaceID}
	default:
		err = fmt.Errorf("%w: %v", ErrTransferDirectionNotSupportedByAPI, request.Direction)
	}
	return
}

func HandleCreateTransfer(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	var request facade4debtus.CreateTransferRequest
	apicore.HandleAuthenticatedRequestWithBody(w, r, &request, verify.DefaultJsonWithAuthRequired, http.StatusCreated,
		func(ctx facade.ContextWithUser) (interface{}, error) {
			from, to, err := buildTransferCounterparties(authInfo.UserID, request)
			if err != nil {
				return nil, err
			}

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
