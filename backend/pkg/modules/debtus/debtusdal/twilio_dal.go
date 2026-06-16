package debtusdal

import (
	"context"
	"fmt"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"github.com/strongo/gotwilio"
	"github.com/strongo/logus"
)

type TwilioDal struct {
}

func NewTwilioDal() TwilioDal {
	return TwilioDal{}
}

func (TwilioDal) GetLastTwilioSmsesForUser(ctx context.Context, tx dal.ReadSession, userID string, to string, limit int) (result []models4debtus.TwilioSms, err error) {
	q := dal.From(dal.NewRootCollectionRef(models4debtus.TwilioSmsKind, "")).
		NewQuery().
		WhereField("UserID", dal.Equal, userID).
		OrderBy(dal.DescendingField("DtCreated"))

	if to != "" {
		q = q.WhereField("To", dal.Equal, to)
	}
	query := q.Limit(limit).SelectIntoRecord(models4debtus.NewTwilioSmsRecord)
	var records []dal.Record
	if records, err = dal.ExecuteQueryAndReadAllToRecords(ctx, query, tx); err != nil {
		return
	}
	result = models4debtus.NewTwilioSmsFromRecords(records)
	return
}

func (TwilioDal) SaveTwilioSms(
	ctx context.Context,
	smsResponse *gotwilio.SmsResponse,
	transfer models4debtus.TransferEntry,
	phoneContact dto4contactus.PhoneContact,
	userID string,
	tgChatID int64,
	smsStatusMessageID int,
) (twilioSms models4debtus.TwilioSms, err error) {
	_ = phoneContact // TODO: restore appending an unverified phone to the counterparty contact - the contactus model no longer stores a flat Phones list
	if err = facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) error {
		debtusUser := models4debtus.NewDebtusUserEntry(userID)
		twilioSms = models4debtus.NewTwilioSms(smsResponse.Sid, nil)
		if err = tx.GetMulti(tctx, []dal.Record{debtusUser.Record, twilioSms.Record, transfer.Record}); err != nil {
			return err
		}
		if twilioSms.Record.Exists() {
			logus.Warningf(ctx, "Twilio SMS already saved to DB")
			return nil
		}
		if !debtusUser.Record.Exists() {
			return fmt.Errorf("debtus user not found: %w: id=%s", dal.ErrRecordNotFound, userID)
		}
		if !transfer.Record.Exists() {
			return fmt.Errorf("transfer not found: %w: id=%s", dal.ErrRecordNotFound, transfer.ID)
		}
		twilioSms.Data.TwilioSmsData = models4debtus.NewTwilioSmsFromSmsResponse(userID, smsResponse)
		twilioSms.Data.CreatorTgChatID = tgChatID
		twilioSms.Data.CreatorTgSmsStatusMessageID = smsStatusMessageID

		debtusUser.Data.SmsCount += 1
		debtusUser.Data.SmsCost += float64(twilioSms.Data.Price)
		transfer.Data.SmsCount += 1
		transfer.Data.SmsCost += float64(twilioSms.Data.Price)

		return tx.SetMulti(tctx, []dal.Record{debtusUser.Record, twilioSms.Record, transfer.Record})
	}); err != nil {
		err = fmt.Errorf("failed to save Twilio response to DB: %w", err)
		return
	}
	return
}
