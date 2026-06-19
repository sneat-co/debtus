package cmds4splitusbot

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/contactus/const4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"

	"context"

	"github.com/sneat-co/sneat-translations/emoji"
)

const groupMembersCommandCode = "group_members"

var groupMembersCommand = botsfw.Command{
	Code:       groupMembersCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   []string{"/members"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		var spaceID coretypes.SpaceID // TODO: implement persisted active space context
		return showGroupMembers(whc, spaceID, dal4contactus.ContactusSpaceEntry{}, false)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {

		ctx := whc.Context()

		spaceID := bothelper.GetSpaceIdFromUrl(callbackUrl)
		contactusSpace := dal4contactus.NewContactusSpaceEntry(spaceID)

		var db dal.DB
		if db, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
		if err = db.Get(ctx, contactusSpace.Record); err != nil {
			return
		}
		return showGroupMembers(whc, spaceID, contactusSpace, true)
	},
}

func groupMembersCard(
	_ context.Context,
	t i18n.SingleLocaleTranslator,
	contactusSpace dal4contactus.ContactusSpaceEntry,
	selectedMemberID int64,
) (text string, err error) {

	membersCount := contactusSpace.Data.GetContactsCount(const4contactus.SpaceMemberRoleMember)
	var buffer bytes.Buffer
	buffer.WriteString(t.Translate(trans.MESSAGE_TEXT_MEMBERS_CARD_TITLE, membersCount) + "\n\n")

	if membersCount > 0 {
		members := contactusSpace.Data.GetSortedContactBriefsByRoles(const4contactus.SpaceMemberRoleMember)
		if len(members) == 0 {
			msg := fmt.Sprintf("ERROR: contactusSpace members count is %d but no member briefs found", membersCount)
			buffer.WriteString("\n" + msg + "\n")
		}
		for i, member := range members {
			title := member.Title
			if title == "" && member.Names != nil {
				title = member.Names.GetFullName()
			}
			_, _ = fmt.Fprintf(&buffer, "  %d. %v\n", i+1, title)
		}
	}

	buffer.WriteString("\n" + t.Translate(trans.MESSAGE_TEXT_MEMBERS_CARD_FOOTER))

	return buffer.String(), nil
}

func showGroupMembers(whc botsfw.WebhookContext, spaceID coretypes.SpaceID, contactusSpace dal4contactus.ContactusSpaceEntry, isEdit bool) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	if spaceID == "" {
		logus.Errorf(ctx, "Space ID is empty")
		m.Text = "ERROR: Space id is empty"
		return
	}

	if m.Text, err = groupMembersCard(ctx, whc, contactusSpace, 0); err != nil {
		return
	}

	m.Format = botmsg.FormatHTML
	tgKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{
				Text:         whc.Translate(trans.BUTTON_TEXT_JOIN),
				CallbackData: joinSpaceCommandCode,
			},
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonSwitchInlineQuery(
				emoji.CONTACTS_ICON+" "+whc.Translate(trans.COMMAND_TEXT_INVITE_MEMBER),
				bothelper.GetSpaceCallbackCommandData(joinSpaceCommandCode, spaceID),
			),
		},
		[]tgbotapi.InlineKeyboardButton{
			{
				Text:         whc.CommandText(trans.SettingsButtonText, emoji.SETTINGS_ICON),
				CallbackData: bothelper.GetSpaceCallbackCommandData(cmds4anybot.SettingsCommandCode, spaceID),
			},
		},
	)
	m.Keyboard = tgKeyboard
	m.IsEdit = isEdit
	return
}
