package delayed4debtus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func delayOnReceiptSentSuccess(ctx context.Context, sentAt time.Time, receiptID, transferID string, tgChatID int64, tgMsgID int, tgBotID, locale string) error {
	if receiptID == "" {
		return errors.New("receiptID == 0")
	}
	if transferID == "" {
		return errors.New("transferID == 0")
	}
	if err := delayer4debtus.OnReceiptSentSuccess.EnqueueWork(ctx, delaying.With(const4debtus.QueueReceipts, "on-receipt-sent-success", 0), sentAt, receiptID, transferID, tgChatID, tgMsgID, tgBotID, locale); err != nil {
		logus.Errorf(ctx, err.Error())
		return DelayedOnReceiptSentSuccess(ctx, sentAt, receiptID, transferID, tgChatID, tgMsgID, tgBotID, locale)
	}
	return nil
}

func DelayedOnReceiptSentSuccess(ctx context.Context, sentAt time.Time, receiptID, transferID string, tgChatID int64, tgMsgID int, tgBotID, locale string) (err error) {
	logus.Debugf(ctx, "DelayedOnReceiptSentSuccess(sentAt=%v, receiptID=%v, transferID=%v, tgChatID=%v, tgMsgID=%v tgBotID=%v, locale=%v)", sentAt, receiptID, transferID, tgChatID, tgMsgID, tgBotID, locale)
	if receiptID == "" {
		logus.Errorf(ctx, "receiptID == 0")
		return

	}
	if transferID == "" {
		logus.Errorf(ctx, "transferID == 0")
		return
	}
	if tgChatID == 0 {
		logus.Errorf(ctx, "tgChatID == 0")
		return
	}
	if tgMsgID == 0 {
		logus.Errorf(ctx, "tgMsgID == 0")
		return
	}
	var mt string
	var receipt models4debtus.ReceiptDbo
	if err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		receipt := models4debtus.NewReceipt(receiptID, nil)
		transfer := models4debtus.NewTransfer(transferID, nil)
		var (
			transferEntity models4debtus.TransferData
		)
		// TODO: Replace with DAL call?
		if err := tx.GetMulti(ctx, []dal.Record{receipt.Record, transfer.Record}); err != nil {
			return err
		}
		if receipt.Data.TransferID != transferID {
			return errors.New("receipt.TransferID != transferID")
		}
		if receipt.Data.Status == models4debtus.ReceiptStatusSent {
			return nil
		}

		transferEntity.Counterparty().TgBotID = tgBotID
		transferEntity.Counterparty().TgChatID = tgChatID
		receipt.Data.DtSent = sentAt
		receipt.Data.Status = models4debtus.ReceiptStatusSent
		if err = tx.SetMulti(ctx, []dal.Record{transfer.Record, receipt.Record}); err != nil {
			return fmt.Errorf("failed to save transfer & receipt to datastore: %w", err)
		}

		if transferEntity.DtDueOn.After(time.Now()) {
			if err := dal4debtus.Default.Reminder.DelayCreateReminderForTransferUser(ctx, transferID, transferEntity.Counterparty().UserID); err != nil {
				return fmt.Errorf("failed to delay creation of reminder for transfer counterparty: %w", err)
			}
		}
		return nil
	}); err != nil {
		mt = err.Error()
	} else {
		var translator i18n.SingleLocaleTranslator
		if translator, err = getTranslator(ctx, locale); err != nil {
			return
		}
		mt = translator.Translate(trans.MESSAGE_TEXT_RECEIPT_SENT_THROW_TELEGRAM)
	}

	if err = editTgMessageTextFn(ctx, tgBotID, tgChatID, tgMsgID, mt); err != nil {
		errMessage := err.Error()
		err = fmt.Errorf("failed to update Telegram message (botID=%v, chatID=%v, msgID=%v): %w", tgBotID, tgChatID, tgMsgID, err)
		if strings.Contains(errMessage, "Bad Request") && strings.Contains(errMessage, " not found") {
			logMessage := logus.Errorf
			switch {
			case receipt.DtCreated.Before(time.Now().Add(-time.Hour * 24)):
				logMessage = logus.Debugf
			case receipt.DtCreated.Before(time.Now().Add(-time.Hour)):
				logMessage = logus.Infof
			case receipt.DtCreated.Before(time.Now().Add(-time.Minute)):
				logMessage = logus.Warningf
			}
			logMessage(ctx, err.Error())
			err = nil
		}
		return
	}
	return
}
