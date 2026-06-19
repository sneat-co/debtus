package dal4debtus

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/decimal"
	"github.com/strongo/gotwilio"
	"github.com/strongo/strongoapp"
)

type TransferSource interface {
	PopulateTransfer(t *models4debtus.TransferData)
}

const (
	AckAccept  = "accept"
	AckDecline = "decline"
)

type TransferReturnUpdate struct {
	TransferID     string
	ReturnedAmount decimal.Decimal64p2
}

type RewardDal interface {
	//GetRewardByID(ctx context.Context, rewardID int64) (reward models.Reward, err error)
	InsertReward(ctx context.Context, tx dal.ReadwriteTransaction, rewardEntity *models4debtus.RewardDbo) (reward models4debtus.Reward, err error)
}

type TransferDal interface {
	GetTransfersByID(ctx context.Context, tx dal.ReadSession, transferIDs []string) ([]models4debtus.TransferEntry, error)
	LoadTransfersByUserID(ctx context.Context, userID string, offset, limit int) (transfers []models4debtus.TransferEntry, hasMore bool, err error)
	LoadTransfersByContactID(ctx context.Context, contactID string, offset, limit int) (transfers []models4debtus.TransferEntry, hasMore bool, err error)
	LoadTransferIDsByContactID(ctx context.Context, contactID string, limit int, startCursor string) (transferIDs []string, endCursor string, err error)
	LoadOverdueTransfers(ctx context.Context, tx dal.ReadSession, userID string, limit int) (transfers []models4debtus.TransferEntry, err error)
	LoadOutstandingTransfers(ctx context.Context, tx dal.ReadSession, periodEnds time.Time, userID, contactID string, currency money.CurrencyCode, direction models4debtus.TransferDirection) (transfers []models4debtus.TransferEntry, err error)
	LoadDueTransfers(ctx context.Context, tx dal.ReadSession, userID string, limit int) (transfers []models4debtus.TransferEntry, err error)
	LoadLatestTransfers(ctx context.Context, offset, limit int) ([]models4debtus.TransferEntry, error)
	DelayUpdateTransferWithCreatorReceiptTgMessageID(ctx context.Context, botCode string, transferID string, creatorTgChatID, creatorTgReceiptMessageID int64) error
	DelayUpdateTransfersWithCounterparty(ctx context.Context, spaceID coretypes.SpaceID, creatorCounterpartyID, counterpartyCounterpartyID string) error
	DelayUpdateTransfersOnReturn(ctx context.Context, returnTransferID string, transferReturnUpdates []TransferReturnUpdate) (err error)
}

type ReceiptDal interface {
	UpdateReceipt(ctx context.Context, tx dal.ReadwriteTransaction, receipt models4debtus.ReceiptEntry) error
	GetReceiptByID(ctx context.Context, tx dal.ReadSession, id string) (models4debtus.ReceiptEntry, error)
	MarkReceiptAsSent(ctx context.Context, receiptID, transferID string, sentTime time.Time) error
	CreateReceipt(ctx context.Context, data *models4debtus.ReceiptDbo) (receipt models4debtus.ReceiptEntry, err error)
	DelayedMarkReceiptAsSent(ctx context.Context, receiptID, transferID string, sentTime time.Time) error
	DelayCreateAndSendReceiptToCounterpartyByTelegram(ctx context.Context, env string, transferID string, userID string) error
}

var ErrReminderAlreadyRescheduled = errors.New("reminder already rescheduled")

type ReminderDal interface {
	DelayDiscardRemindersForTransfers(ctx context.Context, transferIDs []string, returnTransferID string) error
	DelayCreateReminderForTransferUser(ctx context.Context, transferID string, userID string) error
	GetActiveReminderIDsByTransferID(ctx context.Context, tx dal.ReadSession, transferID string) ([]string, error)
	GetSentReminderIDsByTransferID(ctx context.Context, tx dal.ReadSession, transferID string) ([]string, error)
}

type FeedbackDal interface {
	GetFeedbackByID(ctx context.Context, tx dal.ReadSession, feedbackID string) (feedback models4debtus.Feedback, err error)
}

type ContactDal interface {
	GetLatestContacts(ctx context.Context, appUserID string, tx dal.ReadSession, spaceID coretypes.SpaceID, limit, totalCount int) (contacts []models4debtus.DebtusSpaceContactEntry, err error)
	InsertContact(ctx context.Context, tx dal.ReadwriteTransaction, contactEntity *models4debtus.DebtusSpaceContactDbo) (contact models4debtus.DebtusSpaceContactEntry, err error)
	GetContactIDsByTitle(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, userID string, title string, caseSensitive bool) (contactIDs []string, err error)
	GetContactsWithDebts(ctx context.Context, tx dal.ReadSession, spaceID coretypes.SpaceID, userID string) (contacts []models4debtus.DebtusSpaceContactEntry, err error)
}

type TwilioDal interface {
	GetLastTwilioSmsesForUser(ctx context.Context, tx dal.ReadSession, userID string, to string, limit int) (result []models4debtus.TwilioSms, err error)
	SaveTwilioSms(
		ctx context.Context,
		smsResponse *gotwilio.SmsResponse,
		transfer models4debtus.TransferEntry,
		phoneContact dto4contactus.PhoneContact,
		userID string,
		tgChatID int64,
		smsStatusMessageID int,
	) (twiliosSms models4debtus.TwilioSms, err error)
}

const LetterBytes = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed 1, I and 0, O as can be messed with l/1 and 0.
var InviteCodeRegex = regexp.MustCompile(fmt.Sprintf("[%v]+", LetterBytes))

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomCode(n uint8) string {
	b := make([]byte, n)
	lettersCount := len(LetterBytes)
	for i := range b {
		b[i] = LetterBytes[random.Intn(lettersCount)]
	}
	return string(b)
}

type InviteDal interface {
	GetInvite(ctx context.Context, tx dal.ReadSession, inviteCode string) (models4debtus.Invite, error)
	ClaimInvite(ctx context.Context, userID string, inviteCode, claimedOn, claimedVia string) (err error)
	ClaimInvite2(ctx context.Context, inviteCode string, invite models4debtus.Invite, claimedByUserID string, claimedOn, claimedVia string) (err error)
	CreatePersonalInvite(ec strongoapp.ExecutionContext, userID string, inviteBy models4debtus.InviteBy, inviteToAddress, createdOnPlatform, createdOnID, related string) (models4debtus.Invite, error)
	CreateMassInvite(ec strongoapp.ExecutionContext, userID string, inviteCode string, maxClaimsCount int32, createdOnPlatform string) (invite models4debtus.Invite, err error)
}

type AdminDal interface {
	DeleteAll(ctx context.Context, botCode, botChatID string) error
	LatestUsers(ctx context.Context) (users []dbo4userus.UserEntry, err error)
}

// HttpAppHost is a legacy GAE-era global that is referenced by a few HTTP
// handlers but never assigned (latent nil-panic). It is intentionally NOT
// part of dal4debtus.DAL; its remaining call sites are tracked for removal or a
// Firebase-Auth-based replacement (see DEBTUS-IMPROVEMENT-PLAN.md).
var HttpAppHost strongoapp.HttpAppHost

// InsertWithRandomStringID inserts a record, generating a random string ID if
// the record's key is incomplete. Both dalgo2memory and dalgo2firestore v0.8+
// honor dal.WithAdapterGeneratedID(); the adapter assigns the ID to the key
// before returning, so callers can read record.Key().ID after the call.
func InsertWithRandomStringID(ctx context.Context, tx dal.ReadwriteTransaction, record dal.Record) error {
	key := record.Key()
	if id, _ := key.ID.(string); id != "" {
		// Preset ID — insert as-is.
		return tx.Insert(ctx, record)
	}
	return tx.Insert(ctx, record, dal.WithAdapterGeneratedID())
}
