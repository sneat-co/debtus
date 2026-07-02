package api4splitusbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/spaceus/dto4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
)

// createDebtusTransfer is a seam for testing — the split endpoint posts
// who-owes-what records to Debtus through it, one transfer per non-payer
// participant. Splitus keeps no balance of its own.
var createDebtusTransfer = facade4debtus.Transfers.CreateTransfer

// splitusTransferSource marks Debtus transfers as created by the Splitus API.
type splitusTransferSource struct {
	createdOnID string
}

func (s splitusTransferSource) PopulateTransfer(t *models4debtus.TransferData) {
	t.CreatedOnPlatform = "api4splitus"
	t.CreatedOnID = s.createdOnID
}

type createSplitRequest struct {
	SpaceID  coretypes.SpaceID  `json:"spaceID"`
	Title    string             `json:"title,omitempty"`
	Currency money.CurrencyCode `json:"currency"`
	// Amount is the total expense as a decimal string, e.g. "90.00".
	Amount string `json:"amount"`
	// SplitMode defaults to "equally" — the only mode supported so far.
	SplitMode models4splitus.SplitMode `json:"splitMode,omitempty"`
	// ParticipantContactIDs are contactus contact IDs of the space; the payer
	// (the authenticated user) is always a participant and is not listed here.
	ParticipantContactIDs []string `json:"participantContactIDs"`
}

// validate checks field presence/shape and normalizes SplitMode; it returns
// the parsed total amount.
func (v *createSplitRequest) validate() (amount decimal.Decimal64p2, err error) {
	if v.SpaceID == "" {
		return 0, errors.New("missing required field: spaceID")
	}
	if v.Amount == "" {
		return 0, errors.New("missing required field: amount")
	}
	if amount, err = decimal.ParseDecimal64p2(v.Amount); err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}
	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	if v.Currency == "" {
		return 0, errors.New("missing required field: currency")
	}
	if len(v.ParticipantContactIDs) == 0 {
		return 0, errors.New("at least one participant contact ID is required")
	}
	seen := make(map[string]bool, len(v.ParticipantContactIDs))
	for i, contactID := range v.ParticipantContactIDs {
		if contactID == "" {
			return 0, fmt.Errorf("participantContactIDs[%d] is empty", i)
		}
		if seen[contactID] {
			return 0, fmt.Errorf("duplicate participant contact ID: %s", contactID)
		}
		seen[contactID] = true
	}
	if v.SplitMode == "" {
		v.SplitMode = models4splitus.SplitModeEqually
	}
	// Seam for other split modes (percentage, shares, exact-amount): extend
	// this check and the share computation in handleCreateSplit.
	if v.SplitMode != models4splitus.SplitModeEqually {
		return 0, fmt.Errorf("splitMode %q is not supported yet, only %q", v.SplitMode, models4splitus.SplitModeEqually)
	}
	return amount, nil
}

type createSplitTransferDto struct {
	ID        string              `json:"id"`
	ContactID string              `json:"contactID"`
	Amount    decimal.Decimal64p2 `json:"amount"`
}

type createSplitResponse struct {
	ID string `json:"id"`
	// Transfers references the Debtus transfers holding the balances — the
	// only who-owes-what records for this split.
	Transfers []createSplitTransferDto `json:"transfers"`
}

// handleCreateSplit records an expense paid by the authenticated user, splits
// it equally among the payer and the given space contacts, persists a Bill and
// posts one Debtus transfer (payer lends to contact, u2c) per non-payer
// participant for their share.
func handleCreateSplit(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	if authInfo.UserID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var request createSplitRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		common4all.BadRequestError(ctx, w, fmt.Errorf("invalid JSON body: %w", err))
		return
	}
	amount, err := request.validate()
	if err != nil {
		common4all.BadRequestError(ctx, w, err)
		return
	}

	if isMember, err := userBelongsToSpace(ctx, authInfo.UserID, request.SpaceID); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else if !authInfo.IsAdmin && !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	contacts, err := getContactsByIDs(ctx, nil, request.SpaceID, request.ParticipantContactIDs)
	if err != nil {
		if dal.IsNotFound(err) {
			common4all.BadRequestError(ctx, w, fmt.Errorf("participant is not a contact of space %s: %w", request.SpaceID, err))
		} else {
			common4all.InternalError(ctx, w, err)
		}
		return
	}
	// Participants must be existing contacts of the space: a contact record
	// that was not found comes back with zero-valued data and hence no title.
	contactNamesByID := make(map[string]string, len(contacts))
	for _, contact := range contacts {
		if contact.Data.Title == "" && contact.Data.Names == nil {
			continue // Contact record does not exist in the space.
		}
		if title := contact.Data.GetTitle(); title != "" {
			contactNamesByID[contact.ID] = title
		}
	}
	for _, contactID := range request.ParticipantContactIDs {
		if contactNamesByID[contactID] == "" {
			common4all.BadRequestError(ctx, w, fmt.Errorf("participant %q is not a contact of space %s", contactID, request.SpaceID))
			return
		}
	}

	payers, err := getUsersByIDs(ctx, []string{authInfo.UserID})
	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}
	var payer dbo4userus.UserEntry
	for _, u := range payers {
		if u.ID == authInfo.UserID {
			payer = u
		}
	}
	if payer.Data == nil {
		common4all.InternalError(ctx, w, fmt.Errorf("payer user not found by ID=%s", authInfo.UserID))
		return
	}

	// Payer first, then contacts in request order — SetBillMembers computes
	// equal shares and puts any remainder cent on the creator (the payer).
	billMembers := make([]*briefs4splitus.BillMemberBrief, 0, len(request.ParticipantContactIDs)+1)
	billMembers = append(billMembers, &briefs4splitus.BillMemberBrief{
		MemberBrief: briefs4splitus.MemberBrief{
			UserID: authInfo.UserID,
			Name:   payer.Data.GetFullName(),
		},
		Paid: amount,
	})
	for _, contactID := range request.ParticipantContactIDs {
		contactName := contactNamesByID[contactID]
		billMembers = append(billMembers, &briefs4splitus.BillMemberBrief{
			MemberBrief: briefs4splitus.MemberBrief{
				Name: contactName,
				ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
					authInfo.UserID: briefs4splitus.MemberContactBrief{
						ContactID:   contactID,
						ContactName: contactName,
					},
				},
			},
		})
	}

	billEntity := models4splitus.NewBillEntity(models4splitus.BillCommon{
		SpaceID:       request.SpaceID,
		Status:        models4splitus.BillStatusOutstanding,
		SplitMode:     request.SplitMode,
		CreatorUserID: authInfo.UserID,
		Name:          request.Title,
		Currency:      request.Currency,
		AmountTotal:   amount,
	})
	if err = billEntity.SetBillMembers(billMembers); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}

	env := common4all.GetEnvironment(r)
	source := splitusTransferSource{createdOnID: r.Host}

	var response createSplitResponse
	err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		var bill models4splitus.BillEntry
		if bill, err = createBill(ctx, tx, request.SpaceID, billEntity); err != nil {
			return err
		}
		response.ID = bill.ID
		response.Transfers = make([]createSplitTransferDto, 0, len(request.ParticipantContactIDs))
		for _, member := range billEntity.GetBillMembers() {
			if member.UserID == authInfo.UserID {
				continue // The payer owes their own share to nobody.
			}
			contactID := member.ContactByUser[authInfo.UserID].ContactID
			input := facade4debtus.CreateTransferInput{
				Env:         env,
				Source:      source,
				CreatorUser: payer,
				Request: facade4debtus.CreateTransferRequest{
					SpaceRequest: dto4spaceus.SpaceRequest{SpaceID: request.SpaceID},
					Direction:    models4debtus.TransferDirectionUser2Counterparty,
					Amount:       money.Amount{Currency: request.Currency, Value: member.Owes},
					ToContactID:  contactID,
					BillID:       bill.ID,
					Note:         request.Title,
				},
				From: &models4debtus.TransferCounterpartyInfo{UserID: authInfo.UserID, SpaceID: request.SpaceID},
				To:   &models4debtus.TransferCounterpartyInfo{ContactID: contactID, SpaceID: request.SpaceID},
			}
			if err = input.Validate(); err != nil {
				return fmt.Errorf("invalid transfer input for contact %s: %w", contactID, err)
			}
			var output facade4debtus.CreateTransferOutput
			if output, err = createDebtusTransfer(ctx, input); err != nil {
				return fmt.Errorf("failed to create debtus transfer for contact %s: %w", contactID, err)
			}
			response.Transfers = append(response.Transfers, createSplitTransferDto{
				ID:        output.Transfer.ID,
				ContactID: contactID,
				Amount:    member.Owes,
			})
		}
		return nil
	})
	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}
	common4all.JsonToResponse(ctx, w, response)
}
