package delayed4debtus

import (
	"context"
	"errors"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/const4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/delayer4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/strongo/delaying"
	"github.com/strongo/logus"
)

func delayOnReceiptSendFail(ctx context.Context, receiptID string, tgChatID int64, tgMsgID int, failedAt time.Time, locale, details string) error {
	if receiptID == "" {
		return errors.New("receiptID == 0")
	}
	if failedAt.IsZero() {
		return errors.New("failedAt.IsZero()")
	}
	if err := delayer4debtus.OnReceiptSendFail.EnqueueWork(ctx, delaying.With(const4debtus.QueueReceipts, "on-receipt-send-fail", 0), receiptID, tgChatID, tgMsgID, failedAt, locale, details); err != nil {
		logus.Errorf(ctx, err.Error())
		return DelayedOnReceiptSendFail(ctx, receiptID, tgChatID, tgMsgID, failedAt, locale, details)
	}
	return nil
}

func DelayedOnReceiptSendFail(ctx context.Context, receiptID string, tgChatID int64, tgMsgID int, failedAt time.Time, locale, details string) (err error) {
	logus.Debugf(ctx, "DelayedOnReceiptSendFail(receiptID=%v, failedAt=%v)", receiptID, failedAt)
	_ = locale
	if receiptID == "" {
		return errors.New("receiptID == 0")
	}
	var receipt models4debtus.ReceiptEntry
	if err = facade.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		if receipt, err = dal4debtus.Default.Receipt.GetReceiptByID(ctx, tx, receiptID); err != nil {
			return err
		} else if receipt.Data.DtFailed.IsZero() {
			receipt.Data.DtFailed = failedAt
			receipt.Data.Error = details
			if ndsErr := dal4debtus.Default.Receipt.UpdateReceipt(ctx, tx, receipt); ndsErr != nil {
				logus.Errorf(ctx, "Failed to update ReceiptEntry with error information: %v", ndsErr) // Discard error
			}
			return err
		}
		return nil
	}, nil); err != nil {
		return
	}

	if err = editTgMessageTextFn(ctx, receipt.Data.CreatedOnID, tgChatID, tgMsgID, emoji.ERROR_ICON+" Failed to send receipt: "+details); err != nil {
		logus.Errorf(ctx, err.Error())
		err = nil
	}
	return
}
