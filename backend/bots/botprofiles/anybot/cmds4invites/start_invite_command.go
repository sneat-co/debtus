package cmds4invites

import (
	"errors"
	"fmt"
	"net/url"
	"sort"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-core-modules/contactusmodels/const4contactus"
	"github.com/sneat-co/sneat-core-modules/invitus/dbo4invitus"
	"github.com/sneat-co/sneat-core-modules/invitus/facade4invitus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dbo4spaceus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
	"github.com/sneat-co/sneat-translations/trans"
)

const startInviteCommandPrefix = "invite="

var claimPersonalInvite = facade4invitus.ClaimPersonalInvite

var getContactusSpace = dal4contactus.GetContactusSpace

func startInviteCommandAction(whc botsfw.WebhookContext, _ string, startUrl *url.URL) (handled bool, m botmsg.MessageFromBot, err error) {
	if startUrl == nil {
		return
	}
	q := startUrl.Query()

	var request facade4invitus.ClaimPersonalInviteRequest

	if request.SpaceID = bothelper.GetSpaceIdFromUrlQuery(q); request.SpaceID == "" {
		m.Text = "⚠️ Request for invite is missing required 's' (space) parameter value"
		return
	}
	if request.InviteID = q.Get("invite"); request.InviteID == "" {
		m.Text = "⚠️ Request for invite is missing required 'invite' parameter value"
		return
	}
	if request.Pin = q.Get("pin"); request.Pin == "" {
		m.Text = "⚠️ Request for invite is missing required 'pin' parameter value"
		return
	}

	if request.Operation = facade4invitus.InviteClaimOperation(q.Get("o")); request.Operation == "" {
		m.Text = "⚠️ Request for invite is missing required 'o' (operation) parameter value"
		return
	}

	ctx := facade.NewContextWithUserID(whc.Context(), whc.AppUserID())

	var response facade4invitus.ClaimPersonalInviteResponse
	contactusSpace := dal4contactus.NewContactusSpaceEntry(request.SpaceID)
	if request.Operation == "view" {
		var db dal.DB
		if db, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
		response.Space = dbo4spaceus.NewSpaceEntry(request.SpaceID)
		response.Invite = facade4invitus.NewInviteEntry(request.InviteID)
		if err = db.GetMulti(ctx, []dal.Record{
			response.Invite.Record,
			response.Space.Record,
			contactusSpace.Record,
		}); err != nil {
			return
		}
		if response.Invite.Data.SpaceID != request.SpaceID {
			m.Text = fmt.Sprintf(
				"⚠️ Request for invite has wrong 'space' parameter value=%s, the invite is issued for space=%s",
				request.SpaceID, response.Invite.Data.SpaceID)
			return
		}
		if response.Invite.Data.Pin != request.Pin {
			m.Text = "⚠️ Request for invite has wrong 'pin' parameter value"
			return
		}
	} else {
		if response, err = claimPersonalInvite(ctx, request); err != nil {
			if request.Operation == facade4invitus.InviteClaimOperationDecline && errors.Is(err, facade4invitus.ErrInviteAlreadyAccepted) {
				m.Text = "⚠️ You can't decline an already accepted invite"
				err = nil
				return
			}
			if errors.Is(err, facade4invitus.ErrInviteExpired) {
				m.Text = "⚠️ Invite has expired: " + err.Error()
				err = nil
				return
			}
			if errors.Is(err, facade4invitus.ErrInvitePinDoesNotMatch) {
				m.Text = "⚠️ Invite PIN does not match"
				err = nil
				return
			}
			return
		}
		if err = getContactusSpace(ctx, nil, contactusSpace); err != nil {
			return
		}
	}
	handled = true
	m.Format = botmsg.FormatHTML

	switch response.Invite.Data.Status {
	case dbo4invitus.InviteStatusAccepted:
		if response.Space.Data.Type == coretypes.SpaceTypeFamily {
			m.Text = "You've accepted the invite to join a family space"
		} else {
			m.Text = fmt.Sprintf("You've accepted the invite to join %s space", response.Space.Data.Type)
		}
	case dbo4invitus.InviteStatusDeclined:
		if response.Space.Data.Type == coretypes.SpaceTypeFamily {
			m.Text = "You've declined the invite to join a family space"
		} else {
			m.Text = fmt.Sprintf("You've declined the invite to join %s space", response.Space.Data.Type)
		}
	default:
		m.Text = fmt.Sprintf("You've been invited by %s to join %s space",
			response.Invite.Data.From.Title, response.Space.Data.Type)
	}

	members := contactusSpace.Data.GetSortedContactBriefsByRoles(const4contactus.SpaceMemberRoleMember)

	sort.Slice(members, func(i, j int) bool {
		c1, c2 := members[i], members[j]
		return c1.Names.GetFullName() < c2.Names.GetFullName()
	})

	if response.Space.Data.Type == coretypes.SpaceTypeFamily {
		m.Text += "\n\n<b>Family members</b>:"
	} else {
		m.Text += "\n\n<b>Members</b>:"
	}
	for _, member := range members { // TODO: Reuse some function to show ordered list of space members
		var e string
		switch member.Gender {
		case dbmodels.GenderMale:
			e = "♂️"
		case dbmodels.GenderFemale:
			e = "♀️"
		default:
			e = "👤"
		}
		m.Text += fmt.Sprintf("\n\t- %s %s", e, member.GetTitle())
	}

	if !response.Invite.Data.IsClaimed() {
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				{
					Text:         "✅ " + whc.Translate(trans.BtnTextAcceptInvite),
					CallbackData: fmt.Sprintf("invite?id=%s&o=accept", response.Invite.ID),
				},
				{
					Text:         "❌ " + whc.Translate(trans.BtnTextDeclineInvite),
					CallbackData: fmt.Sprintf("invite?id=%s&o=decline", response.Invite.ID),
				},
			},
		)
	}
	return
}
