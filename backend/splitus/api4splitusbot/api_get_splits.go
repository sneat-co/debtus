package api4splitusbot

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
)

// Seams for testing (same pattern as api_bills.go / api_create_split.go).
var listBillsBySpace = facade4splitus.ListBillsBySpace

// getBillTransfers loads the Debtus transfers backing a split. A split's
// who-owes-what lives ONLY in Debtus: handleCreateSplit posts one u2c transfer
// per non-payer participant with CreateTransferRequest.BillID = bill.ID, which
// CreateTransfer persists on TransferData.BillIDs. Reading transfers back by
// that linkage — instead of storing transfer IDs or any settled flag on the
// Bill — is what keeps Debtus the single source of truth for settled state.
var getBillTransfers = func(ctx context.Context, spaceID coretypes.SpaceID, billID string) ([]models4debtus.TransferEntry, error) {
	query := dal.From(models4debtus.TransfersCollectionRef).
		NewQuery().
		WhereArrayContains("BillIDs", billID).
		Limit(100).
		SelectIntoRecord(models4debtus.NewTransferRecord)
	return models4debtus.TransfersFromQuery(ctx, query, nil)
}

const (
	splitShareStatusSettled     = "settled"
	splitShareStatusOutstanding = "outstanding"
)

type splitParticipantDto struct {
	ContactID string              `json:"contactID,omitempty"`
	UserID    string              `json:"userID,omitempty"`
	Name      string              `json:"name"`
	Share     decimal.Decimal64p2 `json:"share"`
	IsPayer   bool                `json:"isPayer,omitempty"`
	// Status is "settled" or "outstanding", DERIVED per request by reading the
	// linked Debtus transfers — Splitus stores no settled/unsettled state.
	Status string `json:"status"`
}

type getSplitResponse struct {
	ID       string              `json:"id"`
	Title    string              `json:"title,omitempty"`
	Currency money.CurrencyCode  `json:"currency"`
	Amount   decimal.Decimal64p2 `json:"amount"`
	// Status is derived: "settled" once every participant's share is settled
	// in Debtus, otherwise the bill's stored lifecycle status.
	Status       string                `json:"status"`
	Participants []splitParticipantDto `json:"participants"`
}

// handleGetSplit returns a single split with per-participant settled state
// derived by reading the linked Debtus transfers (see getBillTransfers).
func handleGetSplit(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	if authInfo.UserID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	query := r.URL.Query()
	spaceID := coretypes.SpaceID(query.Get("spaceID"))
	if spaceID == "" {
		common4all.BadRequestError(ctx, w, errors.New("missing required parameter: spaceID"))
		return
	}
	billID := query.Get("id")
	if billID == "" {
		common4all.BadRequestError(ctx, w, errors.New("missing required parameter: id"))
		return
	}

	// SEC-6: verify membership of the REQUESTED space before touching the bill,
	// so non-members can't probe bill IDs at all.
	if isMember, err := userBelongsToSpace(ctx, authInfo.UserID, spaceID); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else if !authInfo.IsAdmin && !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	bill, err := getBillByID(ctx, nil, billID)
	if err != nil {
		if dal.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		common4all.InternalError(ctx, w, err)
		return
	}
	// A bill that exists but belongs to another space is "not found" for this
	// space — do not leak cross-space existence to members of other spaces.
	if bill.Data.SpaceID != spaceID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	transfers, err := getBillTransfers(ctx, spaceID, billID)
	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}

	// Sum each contact's outstanding value across the bill's debt transfers.
	// A share is settled when its transfer(s) are fully returned in Debtus —
	// GetOutstandingValue() == 0 (returns are tracked on the original transfer
	// via AmountInCentsReturned; return-transfers themselves are skipped).
	now := time.Now()
	outstandingByContactID := make(map[string]decimal.Decimal64p2, len(transfers))
	contactHasTransfer := make(map[string]bool, len(transfers))
	for _, transfer := range transfers {
		if transfer.Data == nil || transfer.Data.IsReturn {
			continue
		}
		contactID := transfer.Data.To().ContactID
		if contactID == "" {
			continue
		}
		contactHasTransfer[contactID] = true
		outstandingByContactID[contactID] += transfer.Data.GetOutstandingValue(now)
	}

	creatorUserID := bill.Data.CreatorUserID
	members := bill.Data.GetBillMembers()
	participants := make([]splitParticipantDto, len(members))
	allSettled := true
	for i, member := range members {
		participant := splitParticipantDto{
			UserID:    member.UserID,
			Name:      member.Name,
			Share:     member.Owes,
			ContactID: member.ContactByUser[creatorUserID].ContactID,
		}
		if member.Paid >= member.Owes {
			// The payer (or anyone who already covered their share within the
			// bill itself) owes nothing to anybody.
			participant.IsPayer = member.Paid > 0
			participant.Status = splitShareStatusSettled
		} else if contactHasTransfer[participant.ContactID] && outstandingByContactID[participant.ContactID] == 0 {
			participant.Status = splitShareStatusSettled
		} else {
			// No Debtus record of settlement (including "no transfer found at
			// all") means the share is still outstanding.
			participant.Status = splitShareStatusOutstanding
			allSettled = false
		}
		participants[i] = participant
	}

	status := bill.Data.Status
	if status == models4splitus.BillStatusOutstanding && allSettled {
		status = models4splitus.BillStatusSettled // derived, never written back
	}

	common4all.JsonToResponse(ctx, w, getSplitResponse{
		ID:           bill.ID,
		Title:        bill.Data.Name,
		Currency:     bill.Data.Currency,
		Amount:       bill.Data.AmountTotal,
		Status:       status,
		Participants: participants,
	})
}

type splitListItemDto struct {
	ID           string              `json:"id"`
	Title        string              `json:"title,omitempty"`
	Amount       decimal.Decimal64p2 `json:"amount"`
	Currency     money.CurrencyCode  `json:"currency"`
	Status       string              `json:"status"`
	MembersCount int                 `json:"membersCount"`
}

type getSplitsResponse struct {
	Splits []splitListItemDto `json:"splits"`
}

// handleGetSplits lists a space's splits (summaries only; per-participant
// settled state is served by handleGetSplit).
func handleGetSplits(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	if authInfo.UserID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))
	if spaceID == "" {
		common4all.BadRequestError(ctx, w, errors.New("missing required parameter: spaceID"))
		return
	}

	// SEC-6: non-members get an authorization error on read as well as create.
	if isMember, err := userBelongsToSpace(ctx, authInfo.UserID, spaceID); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else if !authInfo.IsAdmin && !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	bills, err := listBillsBySpace(ctx, spaceID)
	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}

	response := getSplitsResponse{Splits: make([]splitListItemDto, len(bills))}
	for i, bill := range bills {
		response.Splits[i] = splitListItemDto{
			ID:           bill.ID,
			Title:        bill.Data.Name,
			Amount:       bill.Data.AmountTotal,
			Currency:     bill.Data.Currency,
			Status:       bill.Data.Status,
			MembersCount: len(bill.Data.Members),
		}
	}
	common4all.JsonToResponse(ctx, w, response)
}
