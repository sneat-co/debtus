package api4unsorted

import (
	"context"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-core-modules/userus/facade4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/splitusbot/facade4splitusbot"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/debtusdal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/facade4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
)

// seam vars for external-IO dependencies — overridden in tests

var saveUserBrowser = func(ctx context.Context, userID string, userAgent string) (models4auth.UserBrowser, error) {
	return facade4userus.SaveUserBrowser(ctx, userID, userAgent)
}

var saveGaClient = func(ctx context.Context, gaClientId, userAgent, ipAddress string) (models4auth.GaClient, error) {
	return facade4userus.SaveGaClient(ctx, gaClientId, userAgent, ipAddress)
}

var getUser = func(ctx context.Context, tx dal.ReadSession, user dbo4userus.UserEntry) error {
	return dal4userus.GetUser(ctx, tx, user)
}

var runReadwriteTransaction = func(ctx context.Context, f func(ctx context.Context, tx dal.ReadwriteTransaction) error, options ...dal.TransactionOption) error {
	return facade.RunReadwriteTransaction(ctx, f, options...)
}

var deleteContact = func(ctx context.Context, userCtx facade.UserContext, spaceID coretypes.SpaceID, contactID string) error {
	return facade4debtus.DeleteContact(ctx, userCtx, spaceID, contactID)
}

var changeContactStatus = func(ctx facade.ContextWithUser, spaceID coretypes.SpaceID, contactID string, newStatus models4debtus.DebtusContactStatus) (dal4contactus.ContactEntry, models4debtus.DebtusSpaceContactEntry, error) {
	return facade4debtus.ChangeContactStatus(ctx, spaceID, contactID, newStatus)
}

var updateContact = func(ctx context.Context, spaceID coretypes.SpaceID, counterpartyID string, values map[string]string) (models4debtus.DebtusSpaceContactEntry, error) {
	return facade4debtus.UpdateContact(ctx, spaceID, counterpartyID, values)
}

var createContact = func(ctx facade.ContextWithUser, tx dal.ReadwriteTransaction, spaceID coretypes.SpaceID, contactDetails dto4contactus.ContactDetails) (dal4contactus.ContactEntry, dal4contactus.ContactusSpaceEntry, models4debtus.DebtusSpaceContactEntry, error) {
	return facade4debtus.CreateContact(ctx, tx, spaceID, contactDetails)
}

var loadTransfersByContactID = func(ctx context.Context, contactID string, offset, limit int) ([]models4debtus.TransferEntry, bool, error) {
	return dal4debtus.Default.Transfer.LoadTransfersByContactID(ctx, contactID, offset, limit)
}

var getDebtusSpaceContactByID = func(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, contactID string) (models4debtus.DebtusSpaceContactEntry, error) {
	return facade4debtus.GetDebtusSpaceContactByID(ctx, tx, spaceID, contactID)
}

var newUserContext = func(userID string) facade.UserContext {
	return facade.NewUserContext(userID)
}

var getContactsByIDs = func(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, ids []string) ([]dal4contactus.ContactEntry, error) {
	return dal4contactus.GetContactsByIDs(ctx, tx, spaceID, ids)
}

var getContactusSpace = func(ctx context.Context, tx dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
	return dal4contactus.GetContactusSpace(ctx, tx, entry)
}

var createGroupFn = func(ctx context.Context, groupEntity *models4splitus.GroupDbo, tgBotCode string,
	beforeInsert func(context.Context, *models4splitus.GroupDbo) (models4splitus.GroupEntry, error),
	afterInsert func(context.Context, models4splitus.GroupEntry, dbo4userus.UserEntry) error,
) (models4splitus.GroupEntry, models4splitus.GroupMember, error) {
	return facade4splitusbot.CreateGroup(ctx, groupEntity, tgBotCode, beforeInsert, afterInsert)
}

var getMultiRecords = func(ctx context.Context, db dal.DB, records []dal.Record) error {
	return db.GetMulti(ctx, records)
}

var getTransferByID = func(ctx context.Context, tx dal.ReadSession, transferID string) (models4debtus.TransferEntry, error) {
	return facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
}

var saveTransfer = func(ctx context.Context, tx dal.ReadwriteTransaction, transfer models4debtus.TransferEntry) error {
	return facade4debtus.Transfers.SaveTransfer(ctx, tx, transfer)
}

var delayUpdateTransfersWithCreatorName = func(ctx context.Context, userID string) error {
	return debtusdal.DelayUpdateTransfersWithCreatorName(ctx, userID)
}

var executeQueryToRecordsReader = func(ctx context.Context, db dal.DB, q dal.Query) (dal.RecordsReader, error) {
	return db.ExecuteQueryToRecordsReader(ctx, q)
}

var setLastCurrency = func(ctx facade.ContextWithUser, currencyCode money.CurrencyCode) error {
	return facade4userus.SetLastCurrency(ctx, currencyCode)
}
