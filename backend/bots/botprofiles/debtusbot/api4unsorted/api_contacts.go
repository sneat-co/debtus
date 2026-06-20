package api4unsorted

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/contactus/backend/dto4contactus"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus/dto4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/logus"
	"github.com/strongo/strongoapp/person"
)

type UserCounterpartiesResponse struct {
	UserID         int64
	Counterparties []dto4debtus.ContactListDto
}

func HandleCreateCounterparty(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	name := strings.TrimSpace(r.PostForm.Get("name"))
	email := strings.TrimSpace(r.PostForm.Get("email"))
	tel := strings.TrimSpace(r.PostForm.Get("tel"))
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))

	contactDetails := dto4contactus.ContactDetails{
		NameFields: person.NameFields{
			UserName: name,
		},
	}
	if len(email) > 0 {
		contactDetails.EmailAddressOriginal = email
	}
	if len(tel) > 0 {
		telNumber, err := strconv.ParseInt(tel, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		contactDetails.PhoneNumber = telNumber
	}
	var err error
	var debtusContact models4debtus.DebtusSpaceContactEntry
	err = runReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		ctxWithUser := facade.NewContextWithUserID(ctx, authInfo.UserID)
		_, _, debtusContact, err = createContact(ctxWithUser, tx, spaceID, contactDetails)
		return err
	})

	if err != nil {
		common4all.ErrorAsJson(ctx, w, http.StatusInternalServerError, err)
		return
	}
	_, _ = w.Write([]byte(debtusContact.ID))
}

func getContactID(w http.ResponseWriter, query url.Values) string {
	counterpartyID := query.Get("id")
	if counterpartyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Required parameter 'id' is missing."))
	}
	return counterpartyID
}

func HandleGetContact(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	query := r.URL.Query()
	contactID := getContactID(w, query)
	spaceID := coretypes.SpaceID(query.Get("spaceID"))
	if contactID == "" {
		return
	}

	var db dal.DB
	var err error
	if db, err = facade.GetSneatDB(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	contact := dal4contactus.NewContactEntry(spaceID, contactID)
	debtusContact := models4debtus.NewDebtusSpaceContactEntry(spaceID, contactID, nil)

	if err = getMultiRecords(ctx, db, []dal.Record{contact.Record, debtusContact.Record}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	contactToResponse(ctx, w, authInfo, contact, debtusContact)
}

func contactToResponse(
	ctx context.Context,
	w http.ResponseWriter,
	authInfo token4auth.AuthInfo,
	contact dal4contactus.ContactEntry,
	debtusContact models4debtus.DebtusSpaceContactEntry,
) {
	if !authInfo.IsAdmin && contact.Data.UserID != authInfo.UserID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	transfers, hasMoreTransfers, err := loadTransfersByContactID(ctx, contact.ID, 0, 100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	counterpartyJson := dto4debtus.ContactDetailsDto{
		ContactListDto: dto4debtus.ContactListDto{
			Status: contact.Data.Status,
			ContactDto: dto4debtus.ContactDto{
				ID:     contact.ID,
				Name:   contact.Data.Names.GetFullName(),
				UserID: contact.Data.UserID,
			},
		},
		TransfersResultDto: dto4debtus.TransfersResultDto{
			HasMoreTransfers: hasMoreTransfers,
			Transfers:        dto4debtus.TransfersToDto(authInfo.UserID, transfers),
		},
	}
	if len(debtusContact.Data.Balance) > 0 {
		counterpartyJson.Balance = debtusContact.Data.Balance
	}

	//if contact.Data.EmailAddressOriginal != "" {
	//	counterpartyJson.Email = &dto4debtus.EmailInfo{
	//		Address:     contact.Data.EmailAddressOriginal,
	//		IsConfirmed: contact.Data.EmailConfirmed,
	//	}
	//}
	//if contact.Data.PhoneNumber != 0 {
	//	counterpartyJson.Phone = &dto4debtus.PhoneInfo{
	//		Number:      contact.Data.PhoneNumber,
	//		IsConfirmed: contact.Data.PhoneNumberConfirmed,
	//	}
	//}

	//if len(contact.Data.SpaceIDs) > 0 {
	//	err = errors.New("not implemented")
	//	api4debtus.ErrorAsJson(ctx, w, http.StatusInternalServerError, err)
	//	return
	//	for _, spaceID := range contact.Data.SpaceIDs {
	//		var group models4splitus.GroupEntry
	//		if group, err = dal4debtus.Group.GetGroupByID(ctx, nil, spaceID); err != nil {
	//			api4debtus.ErrorAsJson(ctx, w, http.StatusInternalServerError, err)
	//			return
	//		}
	//		for _, member := range group.Data.GetGroupMembers() {
	//			for _, memberContactID := range member.ContactIDs {
	//				if memberContactID == contact.ContactID {
	//					counterpartyJson.Groups = append(counterpartyJson.Groups, dto4debtus.ContactGroupDto{
	//						ContactID:           group.ContactID,
	//						UserTitle:         group.Data.UserTitle,
	//						MemberID:     memberContactID,
	//						MembersCount: group.Data.MembersCount,
	//					})
	//				}
	//			}
	//		}
	//	}
	//}

	common4all.JsonToResponse(ctx, w, counterpartyJson)
}

//type CounterpartyTransfer struct {
//
//}

func HandleDeleteContact(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	logus.Debugf(ctx, "HandleDeleteContact()")
	//err := r.ParseForm()
	//if err != nil {
	//	BadRequestError(ctx, hashedWriter, err)
	//	return
	//}
	contactID := getContactID(w, r.URL.Query())
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))
	if contactID == "" {
		return
	}
	logus.Debugf(ctx, "contactID: %v", contactID)
	userCtx := newUserContext("")
	if err := deleteContact(ctx, userCtx, spaceID, contactID); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	}
	logus.Infof(ctx, "DebtusSpaceContactEntry deleted: %v", contactID)
}

func HandleArchiveCounterparty(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	//err := r.ParseForm()
	//if err != nil {
	//	BadRequestError(ctx, hashedWriter, err)
	//	return
	//}
	contactID := getContactID(w, r.URL.Query())
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))
	if contactID == "" {
		return
	}
	ctxWithUser := facade.NewContextWithUserID(ctx, authInfo.UserID)
	if contact, debtusContact, err := changeContactStatus(ctxWithUser, spaceID, contactID, const4debtus.StatusArchived); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else {
		contactToResponse(ctx, w, authInfo, contact, debtusContact)
	}
}

func HandleActivateCounterparty(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	//err := r.ParseForm()
	//if err != nil {
	//	BadRequestError(ctx, hashedWriter, err)
	//	return
	//}

	contactID := getContactID(w, r.URL.Query())
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))
	ctxWithUser := facade.NewContextWithUserID(ctx, authInfo.UserID)
	if contactID == "" {
		return
	}
	if contact, debtusContact, err := changeContactStatus(ctxWithUser, spaceID, contactID, const4debtus.StatusActive); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else {
		contactToResponse(ctx, w, authInfo, contact, debtusContact)
	}
}

func HandleUpdateCounterparty(ctx context.Context, w http.ResponseWriter, r *http.Request, authInfo token4auth.AuthInfo) {
	counterpartyID := getContactID(w, r.URL.Query())
	if counterpartyID == "" {
		return
	}
	spaceID := coretypes.SpaceID(r.URL.Query().Get("spaceID"))
	values := make(map[string]string, len(r.PostForm))
	for k, vals := range r.PostForm {
		switch len(vals) {
		case 1:
			values[k] = vals[0]
		case 0:
			values[k] = vals[0]
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "Too many values for '%v'", k)
			return
		}
	}

	if debtusContact, err := updateContact(ctx, spaceID, counterpartyID, values); err != nil {
		common4all.InternalError(ctx, w, err)
		return
	} else {
		contact := dal4contactus.NewContactEntry(spaceID, debtusContact.ID)
		contactToResponse(ctx, w, authInfo, contact, debtusContact)
	}
}
