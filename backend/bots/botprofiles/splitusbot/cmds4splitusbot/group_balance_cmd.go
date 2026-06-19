package cmds4splitusbot

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/shared_splitus"
	"github.com/sneat-co/debtus/backend/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-bots/pkg/bots/sneatbots/facade4bots"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-core-modules/spaceus/facade4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-translations/trans"
)

var groupBalanceCommand = botsfw.Command{
	Code:       "group_balance",
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   []string{"/balance"},
	Action:     shared_splitus.NewSplitusSpaceAction(groupBalanceAction),
	CallbackAction: shared_splitus.NewSplitusSpaceCallbackAction(func(whc botsfw.WebhookContext, callbackUrl *url.URL, splitusSpace models4splitus.SplitusSpaceEntry) (m botmsg.MessageFromBot, err error) {
		return groupBalanceAction(whc, splitusSpace)
	}),
}

func groupBalanceAction(whc botsfw.WebhookContext, splitusSpace models4splitus.SplitusSpaceEntry) (m botmsg.MessageFromBot, err error) {
	var buf bytes.Buffer
	writeMembers := func(members []briefs4splitus.SpaceSplitMember) {
		for i, m := range members {
			_, _ = fmt.Fprintf(&buf, " %d. %v:", i+1, m.Name)
			for currency, amount := range m.Balance {
				if amount < 0 {
					amount *= -1
				}
				fmt.Fprintf(&buf, " %v %v,", amount, currency)
			}
			buf.Truncate(buf.Len() - 1)
			buf.WriteString("\n")
		}
	}
	groupMembers := splitusSpace.Data.GetGroupMembers()
	sponsors, debtors := getGroupSponsorsAndDebtors(groupMembers)

	ctx := whc.Context()

	spaceID := coretypes.SpaceID(splitusSpace.Key.Parent().ID.(string))
	user := facade4bots.GetUserContext(whc)
	var space dbo4spaceus.SpaceEntry
	ctxWithUser := facade.NewContextWithUser(ctx, user)
	if space, err = facade4spaceus.GetSpace(ctxWithUser, spaceID); err != nil {
		return
	}

	buf.WriteString(whc.Translate(trans.MT_GROUP_LABEL, space.Data.Title))
	buf.WriteString("\n")

	buf.WriteString("\n")
	buf.WriteString(whc.Translate(trans.MT_SPONSORS_HEADER))
	buf.WriteString("\n")
	writeMembers(sponsors)

	buf.WriteString("\n")
	buf.WriteString(whc.Translate(trans.MT_DEBTORS_HEADER))
	buf.WriteString("\n")
	writeMembers(debtors)

	m.Text = buf.String()
	m.Format = botmsg.FormatHTML
	m.IsEdit = whc.Input().InputType() == botinput.TypeCallbackQuery

	m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{
				Text: "Settle up",
				URL:  bothelper.StartTelegramBotUrl(whc.GetBotCode(), SettleGroupAskForCounterpartyCommandCode, "splitusSpace="+splitusSpace.ID),
			},
		},
	)
	return
}

func getGroupSponsorsAndDebtors(members []briefs4splitus.SpaceSplitMember, excludeMemberIDs ...string) (sponsors, debtors []briefs4splitus.SpaceSplitMember) {
	sponsors = make([]briefs4splitus.SpaceSplitMember, 0, len(members))
	debtors = make([]briefs4splitus.SpaceSplitMember, 0, len(members))

	for _, m := range members {
		for _, id := range excludeMemberIDs {
			if m.ID == id {
				continue
			}
		}
		for _, v := range m.Balance {
			if v > 0 {
				sponsors = append(sponsors, m)
			} else if v < 0 {
				debtors = append(debtors, m)
			}
		}
	}
	return
}

//func removeGroupMemberByID(members []models.SpaceSplitMember, excludeMemberID string) ([]models.SpaceSplitMember) {
//	for i, m := range members {
//		if m.ContactID == excludeMemberID {
//			return append(members[:i], members[i+1:]...)
//		}
//	}
//	return models.SpaceSplitMember{}, members
//}
