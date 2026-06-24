package debtusdal

import (
	"context"
	"fmt"
	"strings"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus-ext/backend/contactusmodels/const4contactus"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
)

type ContactDal struct {
}

func NewContactDal() ContactDal {
	return ContactDal{}
}

var _ dal4debtus.ContactDal = (*ContactDal)(nil)

func (contactDal ContactDal) DeleteContact(ctx context.Context, tx dal.ReadwriteTransaction, spaceID coretypes.SpaceID, contactID string) (err error) {
	logus.Debugf(ctx, "ContactDal.DeleteContact(spaceID=%s, contactID=%s)", spaceID, contactID)
	if err = tx.Delete(ctx, models4debtus.NewDebtusContactKey(spaceID, contactID)); err != nil {
		return
	}
	if err = delayDeleteContactTransfers(ctx, contactID, ""); err != nil { // TODO: Move to facade4debtus!
		return
	}
	return
}

const DeleteContactTransfersFuncKey = "DeleteContactTransfers"

func delayDeleteContactTransfers(ctx context.Context, contactID string, cursor string) error {
	if err := delayer4debtus.DeleteContactTransfersDelayFunc.EnqueueWork(ctx, delaying.With(const4debtus.QueueTransfers, DeleteContactTransfersFuncKey, 0), contactID, cursor); err != nil {
		return err
	}
	return nil
}

func delayedDeleteContactTransfers(ctx context.Context, contactID string, cursor string) (err error) {
	logus.Debugf(ctx, "delayedDeleteContactTransfers(contactID=%s, cursor=%v", contactID, cursor)
	const limit = 100
	var transferIDs []string
	transferIDs, cursor, err = dal4debtus.Default.Transfer.LoadTransferIDsByContactID(ctx, contactID, limit, cursor)
	if err != nil {
		return
	}
	keys := make([]*dal.Key, len(transferIDs))
	for i, transferID := range transferIDs {
		keys[i] = models4debtus.NewTransferKey(transferID)
	}
	if err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if err = tx.DeleteMulti(ctx, keys); err != nil {
			return err
		}
		if len(transferIDs) == limit {
			if err = delayDeleteContactTransfers(ctx, contactID, cursor); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return
	}
	return
}

func (ContactDal) SaveContact(ctx context.Context, tx dal.ReadwriteTransaction, contact models4debtus.DebtusSpaceContactEntry) error {
	if err := tx.Set(ctx, contact.Record); err != nil {
		return fmt.Errorf("failed to SaveContact(): %w", err)
	}
	return nil
}

func newUserActiveContactsQuery(userID string) dal.IQueryBuilder {
	return newUserContactsQuery(userID).WhereField("Status", dal.Equal, const4debtus.StatusActive)
}

func newUserContactsQuery(userID string) dal.IQueryBuilder {
	return dal.From(dal.NewRootCollectionRef(const4contactus.ContactsCollection, "")).NewQuery().WhereField("UserID", dal.Equal, userID)
}

func (ContactDal) GetContactsWithDebts(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, userID string) (counterparties []models4debtus.DebtusSpaceContactEntry, err error) {
	query := newUserContactsQuery(userID).
		WhereField("BalanceCount", dal.GreaterThen, 0).
		SelectIntoRecord(models4debtus.NewDebtusContactRecord)
	//var (
	//	counterpartyEntities []*models.DebtusSpaceContactDbo
	//)
	records, err := dal.ExecuteQueryAndReadAllToRecords(ctx, query, tx)
	counterparties = make([]models4debtus.DebtusSpaceContactEntry, len(records))
	for i, record := range records {
		counterparties[i] = models4debtus.NewDebtusSpaceContactEntry(spaceID, record.Key().ID.(string), record.Data().(*models4debtus.DebtusSpaceContactDbo))
	}
	return
}

func (ContactDal) GetLatestContacts(ctx context.Context, appUserID string, tx dal.ReadSession, spaceID coretypes.SpaceID, limit, totalCount int) (contacts []models4debtus.DebtusSpaceContactEntry, err error) {
	query := newUserActiveContactsQuery(appUserID).
		OrderBy(dal.DescendingField("LastTransferAt")).
		Limit(limit).
		SelectIntoRecord(models4debtus.NewDebtusContactRecord)
	if tx == nil {
		if tx, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
	}
	var records []dal.Record
	records, err = dal.ExecuteQueryAndReadAllToRecords(ctx, query, tx)
	var contactsCount = len(records)
	logus.Debugf(ctx, "GetLatestContacts(limit=%v, totalCount=%v): %v", limit, totalCount, contactsCount)
	if (limit == 0 && contactsCount < totalCount) || (limit > 0 && totalCount > 0 && contactsCount < limit && contactsCount < totalCount) {
		logus.Debugf(ctx, "Querying contacts without index -LastTransferAt")
		query = newUserActiveContactsQuery(appUserID).
			Limit(limit).
			SelectIntoRecord(models4debtus.NewTransferRecord)
		if records, err = dal.ExecuteQueryAndReadAllToRecords(ctx, query, tx); err != nil {
			return
		}
	}
	contacts = make([]models4debtus.DebtusSpaceContactEntry, len(records))
	for i, record := range records {
		contactID := record.Key().ID.(string)
		dbo := record.Data().(*models4debtus.DebtusSpaceContactDbo)
		contacts[i] = models4debtus.NewDebtusSpaceContactEntry(spaceID, contactID, dbo)
	}
	return
}

func (contactDal ContactDal) GetContactIDsByTitle(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, userID string, title string, caseSensitive bool) (contactIDs []string, err error) {
	contactusSpace := dal4contactus.NewContactusSpaceEntry(spaceID)
	if err = dal4contactus.GetContactusSpace(ctx, tx, contactusSpace); err != nil {
		return
	}
	if caseSensitive {
		for id, contact := range contactusSpace.Data.Contacts {
			if contact.Names.GetFullName() == title {
				contactIDs = append(contactIDs, id)
			}
		}
	} else {
		title = strings.ToLower(title)
		for id, contact := range contactusSpace.Data.Contacts {
			if strings.ToLower(contact.Names.GetFullName()) == title {
				contactIDs = append(contactIDs, id)
			}
		}
	}
	return
}

//func zipCounterparty(keys []*datastore.Key, entities []*models.DebtusSpaceContactDbo) (contacts []models.DebtusSpaceContactEntry) {
//	if len(keys) != len(entities) {
//		panic(fmt.Sprintf("len(keys):%d != len(entities):%d", len(keys), len(entities)))
//	}
//	contacts = make([]models.DebtusSpaceContactEntry, len(entities))
//	for i, entity := range entities {
//		contacts[i] = models.NewDebtusSpaceContactEntry(keys[i].IntID(), entity)
//	}
//	return
//}

func (contactDal ContactDal) InsertContact(ctx context.Context, tx dal.ReadwriteTransaction, contactEntity *models4debtus.DebtusSpaceContactDbo) (
	contact models4debtus.DebtusSpaceContactEntry, err error,
) {
	contact.Data = contactEntity
	if err = tx.Insert(ctx, contact.Record); err != nil {
		return
	}
	contact.ID = contact.Key.ID.(string)
	return
}
