package models4debtus

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/contactus-ext/backend/contactusmodels/const4contactus"
	"github.com/sneat-co/contactus/backend/dto4contactus"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/sneat-core-modules/core/coremodels"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/strongoapp/with"
)

func NewDebtusContactDbo(details dto4contactus.ContactDetails) *DebtusSpaceContactDbo {
	return &DebtusSpaceContactDbo{
		Status: const4debtus.StatusActive,
		CreatedFields: with.CreatedFields{
			CreatedAtField: with.CreatedAtField{
				CreatedAt: time.Now(),
			},
		},
		ContactDetails: details,
		Balanced:       money.Balanced{Balance: make(money.Balance)},
	}
}

type DebtusSpaceContactEntry = record.DataWithID[string, *DebtusSpaceContactDbo]

func NewDebtusSpaceContactEntry(spaceID coretypes.SpaceID, contactID string, dbo *DebtusSpaceContactDbo) DebtusSpaceContactEntry {
	key := dbo4spaceus.NewSpaceModuleItemKey(spaceID, const4debtus.ModuleID, const4contactus.ContactsCollection, contactID)
	if dbo == nil {
		dbo = new(DebtusSpaceContactDbo)
	}
	if dbo.Balance == nil {
		// Balance must be non-nil so a first-ever transfer can call
		// money.Balanced.AddToBalance() without panicking on a nil-map write.
		dbo.Balance = make(money.Balance)
	}
	return record.NewDataWithID(contactID, key, dbo)
}

func NewDebtusContactKey(spaceID coretypes.SpaceID, contactID string) *dal.Key {
	return dbo4spaceus.NewSpaceModuleItemKey(spaceID, const4debtus.ModuleID, const4contactus.ContactsCollection, contactID)
}

func DebtusContactRecords(contacts []DebtusSpaceContactEntry) (records []dal.Record) {
	records = make([]dal.Record, len(contacts))
	for i, contact := range contacts {
		records[i] = contact.Record
	}
	return
}

func NewDebtusSpaceContacts(spaceID coretypes.SpaceID, contactIDs ...string) (contacts []DebtusSpaceContactEntry) {
	contacts = make([]DebtusSpaceContactEntry, len(contactIDs))
	for i, id := range contactIDs {
		if id == "" {
			panic(fmt.Sprintf("contactIDs[%d] == 0", i))
		}
		contacts[i] = NewDebtusSpaceContactEntry(spaceID, id, nil)
	}
	return
}

func NewDebtusContactRecord() dal.Record {
	return dal.NewRecordWithIncompleteKey(const4contactus.ContactsCollection, reflect.String, new(DebtusSpaceContactDbo))
}

// MustMatchCounterparty panics if the contact balance is not the reverse of
// the counterparty contact balance. Re-enabled on the dalgo models: the old
// GAE version also compared BalanceCount, which no longer exists.
func (dbo *DebtusSpaceContactDbo) MustMatchCounterparty(counterparty DebtusSpaceContactEntry) {
	if !dbo.Balance.Equal(counterparty.Data.Balance.Reversed()) {
		panic(fmt.Sprintf("contact.Balance != counterpartyContact[%s].Balance.Reversed(): %v != %v",
			counterparty.ID, dbo.Balance, counterparty.Data.Balance))
	}
}

type WithCounterpartyFields struct {
	CounterpartyContactID string `json:"counterpartyContactID,omitempty" firestore:"counterpartyContactID,omitempty"` // The counterparty user ContactID if registered
}

func (v *WithCounterpartyFields) Validate() error {
	return nil
}

// DebtusSpaceContactDbo is stored in a collection at path "/teams/{teamID}/modules/debtusbot/contacts/{contactID}".
type DebtusSpaceContactDbo struct {
	with.CreatedFields
	money.Balanced
	WithCounterpartyFields
	LinkedBy string `firestore:",omitempty"`
	//
	Status DebtusContactStatus
	dto4contactus.ContactDetails
	Transfers *UserContactTransfersInfo `firestore:"transfers,omitempty"`
	coremodels.SmsStats
	//
	//TelegramChatID int

	// Decided against as we do not need it really and would require either 2 Put() instead of 1 PutMulti()
	//LastTransferID int  `firestore:",omitempty"`

	SearchName          []string `firestore:"searchName,omitempty"` // Deprecated
	NoTransferUpdatesBy []string `firestore:"noTransferUpdatesBy,omitempty"`
	SpaceIDs            []string `firestore:"spaceIDs,omitempty"`
}

func (dbo *DebtusSpaceContactDbo) String() string {
	return fmt.Sprintf("DebtusSpaceContactEntry{CounterpartyContactID: %s, Status: %s, ContactDetails: %v, LastTransferAt: %v}", dbo.CounterpartyContactID, dbo.Status, dbo.ContactDetails, dbo.LastTransferAt)
}

func (dbo *DebtusSpaceContactDbo) GetTransfersInfo() (transfersInfo *UserContactTransfersInfo) {
	return dbo.Transfers
}

func (dbo *DebtusSpaceContactDbo) SetTransfersInfo(transfersInfo UserContactTransfersInfo) error {
	if err := transfersInfo.Validate(); err != nil {
		return err
	}
	dbo.Transfers = &transfersInfo
	return nil
}

func (dbo *DebtusSpaceContactDbo) Info(counterpartyID string, note, comment string) TransferCounterpartyInfo {
	return TransferCounterpartyInfo{
		ContactID: counterpartyID,
		//UserID:      dbo.UserID,
		ContactName: dbo.FullName(),
		Note:        note,
		Comment:     comment,
	}
}

//func (entity *DebtusSpaceContactDbo) UpdateSearchName() {
//	fullName := entity.GetFullName()
//	entity.SearchName = []string{strings.ToLower(fullName)}
//	if entity.Username != "" {
//		username := strings.ToLower(fullName)
//		found := false
//		for _, searchName := range entity.SearchName {
//			if searchName == username {
//				found = true
//			}
//		}
//		if !found {
//			entity.SearchName = append(entity.SearchName, username)
//		}
//	}
//}

// Validate returns error if not valid. TODO: Validate DebtusSpaceContactDbo.Balanced
func (dbo *DebtusSpaceContactDbo) Validate() (err error) {
	//dbo.UpdateSearchName()
	dbo.EmailAddressOriginal = strings.TrimSpace(dbo.EmailAddressOriginal)
	dbo.EmailAddress = strings.ToLower(dbo.EmailAddressOriginal)
	if err = dbo.CreatedFields.Validate(); err != nil {
		return
	}
	if err = dbo.WithCounterpartyFields.Validate(); err != nil {
		return err
	}
	return nil
}

func (dbo *DebtusSpaceContactDbo) BalanceWithInterest(_ context.Context, periodEnds time.Time) (balance money.Balance, err error) {
	if transferInfo := dbo.GetTransfersInfo(); transferInfo != nil {
		err = updateBalanceWithInterest(true, dbo.Balance, transferInfo.OutstandingWithInterest, periodEnds)
	}
	return
}

func ContactsByID(contacts []DebtusSpaceContactEntry) (contactsByID map[string]*DebtusSpaceContactDbo) {
	contactsByID = make(map[string]*DebtusSpaceContactDbo, len(contacts))
	for _, contact := range contacts {
		contactsByID[contact.ID] = contact.Data
	}
	return
}
