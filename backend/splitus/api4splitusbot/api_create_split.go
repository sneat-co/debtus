package api4splitusbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

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

// createSplitShareDto carries one participant's custom share for the
// exact-amount or percentage split modes. ContactID identifies a non-payer
// participant and must be one of ParticipantContactIDs; an empty ContactID
// denotes the payer's own share. Exactly one payer entry plus one entry per
// participant is required when SplitMode is exact-amount or percentage.
type createSplitShareDto struct {
	ContactID string `json:"contactID,omitempty"`
	// Amount is this share's exact decimal amount, required for
	// SplitModeExactAmount, e.g. "35.00".
	Amount string `json:"amount,omitempty"`
	// Percent is this share's percentage of the total, required for
	// SplitModePercentage, e.g. "33.34".
	Percent string `json:"percent,omitempty"`
}

type createSplitRequest struct {
	SpaceID  coretypes.SpaceID  `json:"spaceID"`
	Title    string             `json:"title,omitempty"`
	Currency money.CurrencyCode `json:"currency"`
	// Amount is the total expense as a decimal string, e.g. "90.00".
	Amount string `json:"amount"`
	// SplitMode defaults to "equally". SplitModeExactAmount and
	// SplitModePercentage are also supported, driven by Shares.
	SplitMode models4splitus.SplitMode `json:"splitMode,omitempty"`
	// ParticipantContactIDs are contactus contact IDs of the space; the payer
	// (the authenticated user) is always a participant and is not listed here.
	ParticipantContactIDs []string `json:"participantContactIDs"`
	// Shares carries per-person custom shares for SplitModeExactAmount
	// (Amount) or SplitModePercentage (Percent); required for those modes —
	// one entry per participant plus exactly one payer entry (ContactID ==
	// ""). Ignored for SplitModeEqually.
	Shares []createSplitShareDto `json:"shares,omitempty"`

	// payerOwes and owesByContactID hold the reconciled per-member shares
	// computed by validate() for non-equal split modes.
	payerOwes       decimal.Decimal64p2
	owesByContactID map[string]decimal.Decimal64p2
}

// validate checks field presence/shape, normalizes SplitMode and — for
// SplitModeExactAmount/SplitModePercentage — reconciles the given shares
// against the total, populating payerOwes/owesByContactID. It returns the
// parsed total amount.
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
	switch v.SplitMode {
	case models4splitus.SplitModeEqually:
		// No custom shares to reconcile.
	case models4splitus.SplitModeExactAmount:
		if v.payerOwes, v.owesByContactID, err = computeExactAmountShares(v.Shares, v.ParticipantContactIDs, amount); err != nil {
			return 0, err
		}
	case models4splitus.SplitModePercentage:
		if v.payerOwes, v.owesByContactID, err = computePercentageShares(v.Shares, v.ParticipantContactIDs, amount); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("splitMode %q is not supported yet, only %q, %q, %q", v.SplitMode, models4splitus.SplitModeEqually, models4splitus.SplitModeExactAmount, models4splitus.SplitModePercentage)
	}
	return amount, nil
}

// matchShares pairs the given custom shares with the expected participants —
// exactly one payer entry (ContactID == "") plus one entry per
// participantContactIDs — and errors out naming the discrepancy otherwise.
func matchShares(shares []createSplitShareDto, participantContactIDs []string) (payer createSplitShareDto, byContactID map[string]createSplitShareDto, err error) {
	if len(shares) == 0 {
		return payer, nil, errors.New("shares are required for this splitMode: one entry per participant plus the payer (contactID \"\")")
	}
	expected := make(map[string]bool, len(participantContactIDs)+1)
	expected[""] = true
	for _, id := range participantContactIDs {
		expected[id] = true
	}
	seen := make(map[string]bool, len(shares))
	byContactID = make(map[string]createSplitShareDto, len(participantContactIDs))
	havePayer := false
	for _, s := range shares {
		if !expected[s.ContactID] {
			return payer, nil, fmt.Errorf("share for unknown contact ID %q — not in participantContactIDs", s.ContactID)
		}
		if seen[s.ContactID] {
			if s.ContactID == "" {
				return payer, nil, errors.New("duplicate payer share entry")
			}
			return payer, nil, fmt.Errorf("duplicate share entry for contact %q", s.ContactID)
		}
		seen[s.ContactID] = true
		if s.ContactID == "" {
			payer = s
			havePayer = true
		} else {
			byContactID[s.ContactID] = s
		}
	}
	if !havePayer {
		return payer, nil, errors.New("missing payer share entry (contactID \"\")")
	}
	for _, id := range participantContactIDs {
		if !seen[id] {
			return payer, nil, fmt.Errorf("missing share entry for participant contact %q", id)
		}
	}
	return payer, byContactID, nil
}

// computeExactAmountShares reconciles per-member exact amounts against the
// expense total, rejecting with an explanatory error when they do not sum
// exactly to it.
func computeExactAmountShares(shares []createSplitShareDto, participantContactIDs []string, total decimal.Decimal64p2) (payerOwes decimal.Decimal64p2, owesByContactID map[string]decimal.Decimal64p2, err error) {
	payerShare, byContactID, err := matchShares(shares, participantContactIDs)
	if err != nil {
		return 0, nil, err
	}
	parseAmount := func(label, s string) (decimal.Decimal64p2, error) {
		if s == "" {
			return 0, fmt.Errorf("missing amount for %s", label)
		}
		amt, perr := decimal.ParseDecimal64p2(s)
		if perr != nil {
			return 0, fmt.Errorf("invalid amount %q for %s: %w", s, label, perr)
		}
		if amt < 0 {
			return 0, fmt.Errorf("amount for %s must not be negative", label)
		}
		return amt, nil
	}
	if payerOwes, err = parseAmount("payer", payerShare.Amount); err != nil {
		return 0, nil, err
	}
	sum := payerOwes
	owesByContactID = make(map[string]decimal.Decimal64p2, len(participantContactIDs))
	for _, contactID := range participantContactIDs {
		var amt decimal.Decimal64p2
		if amt, err = parseAmount(fmt.Sprintf("contact %q", contactID), byContactID[contactID].Amount); err != nil {
			return 0, nil, err
		}
		owesByContactID[contactID] = amt
		sum += amt
	}
	if sum != total {
		return 0, nil, fmt.Errorf("shares sum to %s, expense total is %s", formatCents(sum), formatCents(total))
	}
	return payerOwes, owesByContactID, nil
}

// computePercentageShares reconciles per-member percentages against 100%,
// rejecting with an explanatory error when they do not sum exactly to it,
// then converts them to per-member Decimal64p2 amounts using a deterministic
// largest-remainder rule so the computed amounts always sum exactly to the
// total, even when the percentages themselves don't divide it evenly.
func computePercentageShares(shares []createSplitShareDto, participantContactIDs []string, total decimal.Decimal64p2) (payerOwes decimal.Decimal64p2, owesByContactID map[string]decimal.Decimal64p2, err error) {
	payerShare, byContactID, err := matchShares(shares, participantContactIDs)
	if err != nil {
		return 0, nil, err
	}
	parsePercent := func(label, s string) (decimal.Decimal64p2, error) {
		if s == "" {
			return 0, fmt.Errorf("missing percent for %s", label)
		}
		pct, perr := decimal.ParseDecimal64p2(s)
		if perr != nil {
			return 0, fmt.Errorf("invalid percent %q for %s: %w", s, label, perr)
		}
		if pct < 0 {
			return 0, fmt.Errorf("percent for %s must not be negative", label)
		}
		return pct, nil
	}

	type keyedPercent struct {
		key     string // "" denotes the payer.
		percent decimal.Decimal64p2
	}
	entries := make([]keyedPercent, 0, len(participantContactIDs)+1)

	payerPercent, err := parsePercent("payer", payerShare.Percent)
	if err != nil {
		return 0, nil, err
	}
	entries = append(entries, keyedPercent{key: "", percent: payerPercent})
	sumPercent := payerPercent

	for _, contactID := range participantContactIDs {
		pct, perr := parsePercent(fmt.Sprintf("contact %q", contactID), byContactID[contactID].Percent)
		if perr != nil {
			return 0, nil, perr
		}
		entries = append(entries, keyedPercent{key: contactID, percent: pct})
		sumPercent += pct
	}

	const fullPercent = decimal.Decimal64p2(10000) // 100.00%
	if sumPercent != fullPercent {
		return 0, nil, fmt.Errorf("percentages sum to %s%%, expected 100%%", formatCents(sumPercent))
	}

	// Largest-remainder rounding: floor each member's exact share, then hand
	// out the leftover cents (always < len(entries)) to the members with the
	// largest fractional remainder, breaking ties by entry order (payer
	// first, then participants in request order) for determinism.
	totalCents := int64(total)
	type calc struct {
		key       string
		floor     int64
		remainder int64
	}
	calcs := make([]calc, len(entries))
	var sumFloor int64
	for i, e := range entries {
		numerator := totalCents * int64(e.percent)
		calcs[i] = calc{key: e.key, floor: numerator / int64(fullPercent), remainder: numerator % int64(fullPercent)}
		sumFloor += calcs[i].floor
	}
	extraCents := totalCents - sumFloor

	order := make([]int, len(calcs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return calcs[order[a]].remainder > calcs[order[b]].remainder
	})
	for i := int64(0); i < extraCents; i++ {
		calcs[order[i]].floor++
	}

	owesByContactID = make(map[string]decimal.Decimal64p2, len(participantContactIDs))
	for _, c := range calcs {
		if c.key == "" {
			payerOwes = decimal.Decimal64p2(c.floor)
		} else {
			owesByContactID[c.key] = decimal.Decimal64p2(c.floor)
		}
	}
	return payerOwes, owesByContactID, nil
}

// formatCents renders a Decimal64p2 as a fixed 2-decimal string (e.g.
// "100.00"), unlike Decimal64p2.String() which drops trailing zeroes — used
// for explanatory reconciliation error messages.
func formatCents(d decimal.Decimal64p2) string {
	v := int64(d)
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%d.%02d", v/100, v%100)
	if neg {
		s = "-" + s
	}
	return s
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

	// Payer first, then contacts in request order. For SplitModeEqually,
	// SetBillMembers computes equal shares and puts any remainder cent on the
	// creator (the payer). For exact-amount/percentage, Owes was already
	// reconciled to the total in validate() and is set here directly.
	isCustomSplit := request.SplitMode != models4splitus.SplitModeEqually
	payerMember := &briefs4splitus.BillMemberBrief{
		MemberBrief: briefs4splitus.MemberBrief{
			UserID: authInfo.UserID,
			Name:   payer.Data.GetFullName(),
		},
		Paid: amount,
	}
	if isCustomSplit {
		payerMember.Owes = request.payerOwes
	}
	billMembers := make([]*briefs4splitus.BillMemberBrief, 0, len(request.ParticipantContactIDs)+1)
	billMembers = append(billMembers, payerMember)
	for _, contactID := range request.ParticipantContactIDs {
		contactName := contactNamesByID[contactID]
		member := &briefs4splitus.BillMemberBrief{
			MemberBrief: briefs4splitus.MemberBrief{
				Name: contactName,
				ContactByUser: briefs4splitus.MemberContactBriefsByUserID{
					authInfo.UserID: briefs4splitus.MemberContactBrief{
						ContactID:   contactID,
						ContactName: contactName,
					},
				},
			},
		}
		if isCustomSplit {
			member.Owes = request.owesByContactID[contactID]
		}
		billMembers = append(billMembers, member)
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
	if isCustomSplit {
		err = billEntity.SetBillMembersWithOwes(billMembers)
	} else {
		err = billEntity.SetBillMembers(billMembers)
	}
	if err != nil {
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
