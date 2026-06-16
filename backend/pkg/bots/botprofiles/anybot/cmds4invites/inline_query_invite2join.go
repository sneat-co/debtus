package cmds4invites

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/invitus/dbo4invitus"
	"github.com/sneat-co/sneat-core-modules/invitus/facade4invitus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dto4spaceus"
	"github.com/sneat-co/sneat-go-core/apicore"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
	"github.com/strongo/validation"
)

var createOrReuseInviteToContact = facade4invitus.CreateOrReuseInviteToContact

const inviteContactToJoinInlineQueryCommandCode = "inviteContactToJoin"

var inviteContactToJoinInlineQueryCommand = botsfw.Command{
	Code:              inviteContactToJoinInlineQueryCommandCode,
	InputTypes:        []botinput.Type{botinput.TypeInlineQuery},
	InlineQueryAction: inviteContactToJoinInlineQueryAction,
}

func inviteContactToJoinInlineQueryAction(whc botsfw.WebhookContext, inlineQuery botinput.InlineQuery, queryUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	userID := whc.AppUserID()
	ctx := facade.NewContextWithUserID(whc.Context(), userID)
	logus.Debugf(ctx, "handleInlineQueryInviteToJoinSpace()")

	botCode := whc.GetBotCode()

	spaceRef := bothelper.GetSpaceRefFromUrl(queryUrl)

	var contact dal4contactus.ContactEntry
	q := queryUrl.Query()
	contact.ID = q.Get("c")

	request := facade4invitus.InviteContactRequest{
		ContactRequest: dto4contactus.ContactRequest{
			ContactID: contact.ID,
			SpaceRequest: dto4spaceus.SpaceRequest{
				SpaceID: spaceRef.SpaceID(),
			},
		},
		To: dbo4invitus.InviteTo{
			InviteContact: dbo4invitus.InviteContact{
				Channel:   "telegram",
				ContactID: contact.ID,
			},
		},
		Send: false,
	}
	var resp facade4invitus.CreateInviteResponse
	if resp, err = createOrReuseInviteToContact(ctx, request,
		func() dbmodels.RemoteClientInfo {
			remoteClientInfo := apicore.GetRemoteClientInfo(whc.Request())
			botContext := whc.BotContext()
			hostPlatform := whc.BotPlatform().ID() + "@"
			remoteClientInfo.HostOrApp = hostPlatform + botContext.BotSettings.Code
			if remoteClientInfo.HostOrApp == hostPlatform {
				remoteClientInfo.HostOrApp += botContext.BotSettings.ID
			}
			return remoteClientInfo
		}); err != nil {
		if validation.IsBadRequestError(err) {
			err = fmt.Errorf("function CreateOrReuseInviteToContact got bad paramets: %v", err) // Do not user %w here
		}
		return
	}
	locale := q.Get("l")
	if locale == "" {
		locale = resp.Space.Data.GetPreferredLocale()
	}
	if locale == "" {
		var appUserData botsfwmodels.AppUserData
		if appUserData, err = whc.AppUserData(); err != nil {
			return
		}
		locale = appUserData.BotsFwAdapter().GetPreferredLocale()
	}
	if locale != "" {
		if err = whc.SetLocale(locale); err != nil {
			return
		}
	}

	title := fmt.Sprintf(whc.Translate(trans.InlineInviteToJoinFamilyTitle), botCode)
	description := whc.Translate(trans.InlineInviteToJoinFamilyDescription)

	youAreInvitedMsg := strings.Replace(whc.Translate(trans.YouAreInvitedToJoinFamilyMessage), "{BOT_ID}", botCode, 1)

	var textMessage tgbotapi.InputTextMessageContent
	textMessage.MessageText = youAreInvitedMsg
	textMessage.ParseMode = "HTML"

	urlToStartWithInInvite := bothelper.StartTelegramBotUrl(botCode, "invite",
		"pin="+resp.Invite.Data.Pin,
		"s="+string(spaceRef),
		"", // This is needed to have a separator at the end
	)

	replyMarkup := &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				{
					Text: "✅ " + whc.Translate(trans.BtnTextAcceptInvite),
					URL:  urlToStartWithInInvite + "o=accept",
				},
				{
					Text: "❌ " + whc.Translate(trans.BtnTextDeclineInvite),
					URL:  urlToStartWithInInvite + "o=decline",
				},
			},
			{
				{
					Text: "🔍 " + whc.Translate(trans.BtnViewFamilyAccount),
					URL:  urlToStartWithInInvite + "o=view",
				},
			},
		},
	}

	articleButton := tgbotapi.InlineQueryResultArticle{
		InlineQueryResultBase: tgbotapi.InlineQueryResultBase{
			ID:          "invite#" + resp.Invite.ID,
			Type:        "article",
			Title:       title,
			ReplyMarkup: replyMarkup,
		},
		Description:         description,
		InputMessageContent: textMessage,
	}

	_ = articleButton

	m.BotMessage = telegram.InlineBotMessage(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.GetInlineQueryID(),
		CacheTime:     1,
		IsPersonal:    true,

		// This button is not shown to users who already started the bot.
		// So it makes little sense to reply on it.
		//Button: &tgbotapi.InlineQueryResultsButton{
		//	Text:           "Review invite",
		//	StartParameter: "invite",
		//},

		Results: []tgbotapi.InlineQueryResult{
			articleButton,

			//tgbotapi.InlineQueryResultGIF{
			//	Type:                "gif",
			//	ID:                  "invite_gif",
			//	URL:                 "https://media0.giphy.com/media/v1.Y2lkPTc5MGI3NjExaGNoMXdjbjJoeWswYzZxZmEwaGdnazY2azQwcGVxNmU4cm1ncm9xNSZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/MFOn2TuPE5us5Pm3Nf/giphy.gif",
			//	ThumbURL:            "https://media0.giphy.com/media/MFOn2TuPE5us5Pm3Nf/200w.webp",
			//	Width:               384,
			//	Height:              520,
			//	Title:               title,
			//	Caption:             description,
			//	InputMessageContent: textMessage,
			//	ReplyMarkup:         replyMarkup,
			//},
		},
	})
	return
}
