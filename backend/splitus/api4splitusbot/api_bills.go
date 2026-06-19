package api4splitusbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/facade4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/strongo/decimal"
)

// Seams for testing — allows tests to replace with stubs without a real database.
var getBillByID = facade4splitus.GetBillByID
var getContactsByIDs = dal4contactus.GetContactsByIDs
var getUsersByIDs = dal4userus.GetUsersByIDs
var createBill = facade4splitus.CreateBill
var runReadwriteTransaction = facade.RunReadwriteTransaction

func handleGetBill(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	billID := r.URL.Query().Get("id")
	if billID == "" {
		common4all.BadRequestError(ctx, w, errors.New("missing id parameter"))
		return
	}
	bill, err := getBillByID(ctx, nil, billID)
	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}
	billToResponse(ctx, w, authInfo.UserID, bill)
}

func handleCreateBill(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	splitMode := models4splitus.SplitMode(r.PostFormValue("split"))
	if !models4splitus.IsValidBillSplit(splitMode) {
		common4all.BadRequestMessage(ctx, w, fmt.Sprintf("Split parameter has unkown value: %v", splitMode))
		return
	}
	spaceID := coretypes.SpaceID(r.PostFormValue("spaceID"))
	if spaceID == "" {
		common4all.BadRequestMessage(ctx, w, "Missing required parameter: spaceID")
		return
	}
	amountStr := r.PostFormValue("amount")
	if amountStr == "" {
		common4all.BadRequestMessage(ctx, w, fmt.Sprintf("Missing required parameter: amount. %v", r.PostForm))
		return
	}
	amount, err := decimal.ParseDecimal64p2(amountStr)
	if err != nil {
		common4all.BadRequestError(ctx, w, err)
		return
	}
	var members []dto4debtus.BillMemberDto
	{
		membersJSON := r.PostFormValue("members")
		if err = json.Unmarshal([]byte(membersJSON), &members); err != nil {
			common4all.BadRequestError(ctx, w, err)
			return
		}

	}
	if len(members) == 0 {
		common4all.BadRequestMessage(ctx, w, "No members has been provided")
		return
	}
	billEntity := models4splitus.NewBillEntity(models4splitus.BillCommon{
		Status:        models4splitus.BillStatusDraft,
		SplitMode:     splitMode,
		CreatorUserID: authInfo.UserID,
		Name:          r.PostFormValue("name"),
		Currency:      money.CurrencyCode(r.PostFormValue("currency")),
		AmountTotal:   amount,
	})

	var (
		totalByMembers decimal.Decimal64p2
	)

	contactIDs := make([]string, 0, len(members))
	memberUserIDs := make([]string, 0, len(members))

	for i, member := range members {
		if member.ContactID == "" && member.UserID == "" {
			common4all.BadRequestMessage(ctx, w, fmt.Sprintf("members[%d]: ContactID == 0 && UserID == 0", i))
			return
		}
		if member.ContactID != "" {
			contactIDs = append(contactIDs, member.ContactID)
		}
		if member.UserID != "" {
			memberUserIDs = append(memberUserIDs, member.UserID)
		}
	}

	var contacts []dal4contactus.ContactEntry
	if len(contactIDs) > 0 {
		if contacts, err = getContactsByIDs(ctx, nil, spaceID, contactIDs); err != nil {
			common4all.InternalError(ctx, w, err)
			return
		}
	}

	var memberUsers []dbo4userus.UserEntry
	if len(memberUserIDs) > 0 {
		if memberUsers, err = getUsersByIDs(ctx, memberUserIDs); err != nil {
			common4all.InternalError(ctx, w, err)
			return
		}
	}

	billMembers := make([]*briefs4splitus.BillMemberBrief, len(members))
	for i, member := range members {
		if member.UserID != "" && member.ContactID != "" {
			common4all.BadRequestMessage(ctx, w, fmt.Sprintf("Member has both UserID and ContactID: %v, %v", member.UserID, member.ContactID))
			return
		}
		totalByMembers += member.Amount
		billMembers[i] = &briefs4splitus.BillMemberBrief{
			MemberBrief: briefs4splitus.MemberBrief{
				UserID: member.UserID,
				Shares: member.Share,
			},
			Percent:    member.Percent,
			Owes:       member.Amount,
			Adjustment: member.Adjustment,
		}
		if member.ContactID != "" {
			for _, contact := range contacts {
				if contact.ID == member.ContactID {
					contactName := contact.Data.GetTitle()
					billMembers[i].ContactByUser = briefs4splitus.MemberContactBriefsByUserID{
						contact.Data.UserID: briefs4splitus.MemberContactBrief{
							ContactID:   member.ContactID,
							ContactName: contactName,
						},
					}
					if billMembers[i].Name == "" {
						billMembers[i].Name = contactName
					}
					goto contactFound
				}
			}
			common4all.BadRequestError(ctx, w, fmt.Errorf("contact not found by member.ContactID=%s", member.ContactID))
			return
		contactFound:
		}
		if member.UserID != "" {
			for _, u := range memberUsers {
				if u.ID == member.UserID {
					billMembers[i].Name = u.Data.GetFullName()
					break
				}
			}
		}
	}
	if totalByMembers != amount {
		common4all.BadRequestMessage(ctx, w, fmt.Sprintf("Total amount is not equal to sum of member's amounts: %v != %v", amount, totalByMembers))
		return
	}

	billEntity.SplitMode = models4splitus.SplitModePercentage

	if err = billEntity.SetBillMembers(billMembers); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}

	var bill models4splitus.BillEntry
	err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		bill, err = createBill(ctx, tx, spaceID, billEntity)
		return
	})

	if err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}
	billToResponse(ctx, w, authInfo.UserID, bill)
}

func billToResponse(ctx context.Context, w http.ResponseWriter, userID string, bill models4splitus.BillEntry) {
	if userID == "" {
		common4all.InternalError(ctx, w, errors.New("required parameter userID == 0"))
		return
	}
	if bill.ID == "" {
		common4all.InternalError(ctx, w, errors.New("required parameter bill.ContactID is empty string"))
		return
	}
	if bill.Data == nil {
		common4all.InternalError(ctx, w, errors.New("required parameter bill.BillDbo is nil"))
		return
	}
	billDto := dto4debtus.BillDto{
		ID:   bill.ID,
		Name: bill.Data.Name,
		Amount: money.Amount{
			Currency: money.CurrencyCode(bill.Data.Currency),
			Value:    decimal.Decimal64p2(bill.Data.AmountTotal),
		},
	}
	billMembers := bill.Data.GetBillMembers()
	members := make([]dto4debtus.BillMemberDto, len(billMembers))
	for i, billMember := range billMembers {
		members[i] = dto4debtus.BillMemberDto{
			UserID:     billMember.UserID,
			ContactID:  billMember.ContactByUser[userID].ContactID,
			Amount:     billMember.Owes,
			Adjustment: billMember.Adjustment,
			Share:      billMember.Shares,
		}
	}
	billDto.Members = members
	common4all.JsonToResponse(ctx, w, map[string]dto4debtus.BillDto{"BillEntry": billDto}) // TODO: Define DTO as need to clean BillMember.ContactByUser
}
