// Package botscompat holds bot helpers that used to live in the sneat-go
// pkg/bots package and are still needed by the debtus/splitus bots after the
// extraction. They depend only on published modules (no sneat-go imports).
//
// TODO: These belong in a more appropriate shared module (sneat-bots). They are
// kept here to avoid a cross-repo import cycle back into sneat-go.
package botscompat

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
)

// NewLinkerFromWhc builds a common4all.Linker from a webhook context.
func NewLinkerFromWhc(whc botsfw.WebhookContext) common4all.Linker {
	botCode := whc.GetBotCode()
	botPlatformID := whc.BotPlatform().ID()
	userID := whc.AppUserID()
	return common4all.NewLinker(whc.Environment(), userID, whc.Locale().SiteCode(), token4auth.GetBotIssuer(botPlatformID, botCode))
}

// TgChatDal is a seam for storing telegram chat data within a transaction.
type TgChatDal interface {
	DoSomething( // TODO: WTF name?
		ctx context.Context,
		userTask *sync.WaitGroup,
		currency string,
		tgChatID int64,
		authInfo token4auth.AuthInfo,
		user dbo4userus.UserEntry,
		sendToTelegram func(tgChat botsfwtgmodels.TgChatData) error,
	) (err error)
}

// TgChat is the active TgChatDal implementation (overridable in tests).
var TgChat TgChatDal = tgChatDal{}

// runTgChatTransaction is a seam for facade.RunReadwriteTransaction to allow testing tx.Set error paths.
var runTgChatTransaction = facade.RunReadwriteTransaction

type tgChatDal struct{}

func (tgChatDal) DoSomething(
	ctx context.Context,
	userTask *sync.WaitGroup, currency string, tgChatID int64, authInfo token4auth.AuthInfo, user dbo4userus.UserEntry,
	sendToTelegram func(tgChat botsfwtgmodels.TgChatData) error,
) (err error) {
	var isSentToTelegram bool // Needed in case of failed to save to DB and is auto-retry
	tgChatData := new(anybot.SneatAppTgChatDbo)

	id := strconv.FormatInt(tgChatID, 10)
	debtusTgChat := anybot.SneatAppTgChatEntry{
		RecordWithID: record.NewWithID(id, dal.NewKeyWithID(botsfwtgmodels.TgChatCollection, id), tgChatData),
		Data:         tgChatData,
	}

	if err = runTgChatTransaction(ctx, func(tctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		if err = tx.Get(tctx, debtusTgChat.Record); err != nil {
			return err
		}
		debtusTgChat.Data.AddWizardParam("currency", string(currency))

		if !isSentToTelegram {
			if err = sendToTelegram(debtusTgChat.Data); err != nil {
				return err
			}
			isSentToTelegram = true
		}
		if err = tx.Set(tctx, debtusTgChat.Record); err != nil {
			return fmt.Errorf("failed to save Telegram chat record to db: %w", err)
		}
		return err
	}, nil); err != nil {
		err = fmt.Errorf("method TgChatDal.DoSomething() transaction failed: %w", err)
	}
	return
}
