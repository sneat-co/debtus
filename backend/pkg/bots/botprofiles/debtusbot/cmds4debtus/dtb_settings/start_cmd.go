package dtb_settings

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/userus/delays4userus"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_transfer"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/strongo/logus"
)

/*
Examples:

	receipt-{ContactID}-view_{LANG_CODE5}_[GA_CLIENT_ID]
*/
var reInviteOrReceiptCodeFromStart = regexp.MustCompile(`^(invite|receipt)-(\w+)(-(view|accept|decline))?(_(\w{2}(-\w{2})?))(_(.+))?$`)

var (
	getInvite = func(ctx context.Context, inviteCode string) (models4debtus.Invite, error) {
		return dal4debtus.Default.Invite.GetInvite(ctx, nil, inviteCode)
	}
	getReceiptByID = func(ctx context.Context, receiptID string) error {
		_, err := dal4debtus.Default.Receipt.GetReceiptByID(ctx, nil, receiptID)
		return err
	}
	delaySetUserPreferredLocale = delays4userus.DelaySetUserPreferredLocale
	showReceipt                 = dtb_transfer.ShowReceipt
	acknowledgeReceipt          = dtb_transfer.AcknowledgeReceipt
)

func StartInBotAction(whc botsfw.WebhookContext, startParams []string) (m botmsg.MessageFromBot, err error) {
	switch len(startParams) {
	case 0:

	case 1:
		if matched := reInviteOrReceiptCodeFromStart.FindStringSubmatch(startParams[0]); matched != nil {
			return startByLinkCode(whc, matched)
		}
	default:
		err = cmds4anybot.ErrUnknownStartParam
	}
	return
}

func startByLinkCode(whc botsfw.WebhookContext, matches []string) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "startByLinkCode() => matches: %v", matches)
	chatEntity := whc.ChatData()
	entityType := matches[1]
	entityCode := matches[2]
	operation := matches[4]
	localeCode5 := matches[6]
	//gaClientId := matches[8]
	if localeCode5 != "" {
		if len(localeCode5) == 2 {
			localeCode5 = common4all.Locale2to5(localeCode5)
		}
		if err = whc.SetLocale(localeCode5); err != nil {
			return
		}
		chatEntity.SetPreferredLanguage(localeCode5)
		if err = delaySetUserPreferredLocale(ctx, time.Second, whc.AppUserID(), localeCode5); err != nil {
			return
		}
	}
	switch entityType {
	case "receipt":
		return startReceipt(whc, entityCode, operation, localeCode5)
	case "invite":
		return startInvite(whc, entityCode, operation, localeCode5)
	default:
		err = cmds4anybot.ErrUnknownStartParam
	}
	return
}

func startInvite(whc botsfw.WebhookContext, inviteCode, operation, localeCode5 string) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	var invite models4debtus.Invite
	if invite, err = getInvite(ctx, inviteCode); err != nil {
		if dal.IsNotFound(err) {
			return whc.NewMessage(fmt.Sprintf("Unknown invite code: %v", inviteCode)), nil
		}
		return
	}
	logus.Debugf(ctx, "Invite(%v): ClaimedCount=%v, MaxClaimsCount=%v", inviteCode, invite.Data.ClaimedCount, invite.Data.MaxClaimsCount)
	if invite.Data.MaxClaimsCount == 0 || invite.Data.ClaimedCount < invite.Data.MaxClaimsCount {
		return handleInviteOnStart(whc, inviteCode, invite)
	} else {
		m = whc.NewMessage(fmt.Sprintf("Known & already claimed invite code: %v", inviteCode))
	}
	return m, err
}

func startReceipt(whc botsfw.WebhookContext, receiptID, operation, localeCode5 string) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	if receiptID == "" {
		return m, fmt.Errorf("receiptID is empty")
	} else if err = getReceiptByID(ctx, receiptID); err != nil {
		return
	}
	switch operation {
	case "view":
		if err = whc.SetLocale(localeCode5); err != nil {
			return
		}
		return showReceipt(whc, receiptID)
	default:
		return acknowledgeReceipt(whc, receiptID, operation)
	}
}
