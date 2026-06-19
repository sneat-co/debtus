package delayed4debtus

import (
	"context"
	"fmt"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/logus"
)

func DelayedUpdateTransferWithCreatorReceiptTgMessageID(ctx context.Context, botCode string, transferID string, creatorTgChatID, creatorTgReceiptMessageID int64) (err error) {
	logus.Infof(ctx, "DelayedUpdateTransferWithCreatorReceiptTgMessageID(botCode=%v, transferID=%v, creatorTgChatID=%v, creatorReceiptTgMessageID=%v)", botCode, transferID, creatorTgChatID, creatorTgReceiptMessageID)
	return facade.RunReadwriteTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) error {
		transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
		if err != nil {
			logus.Errorf(ctx, "Failed to get transfer by ContactID: %v", err)
			if dal.IsNotFound(err) {
				return nil
			} else {
				return err
			}
		}
		logus.Debugf(ctx, "Loaded transfer: %v", transfer.Data)
		if transfer.Data.Creator().TgBotID != botCode || transfer.Data.Creator().TgChatID != creatorTgChatID || transfer.Data.CreatorTgReceiptByTgMsgID != creatorTgReceiptMessageID {
			transfer.Data.Creator().TgBotID = botCode
			transfer.Data.Creator().TgChatID = creatorTgChatID
			transfer.Data.CreatorTgReceiptByTgMsgID = creatorTgReceiptMessageID
			if err = facade4debtus.Transfers.SaveTransfer(ctx, tx, transfer); err != nil {
				err = fmt.Errorf("failed to save transfer to db: %w", err)
			}
		}
		return err
	}, nil)
}
