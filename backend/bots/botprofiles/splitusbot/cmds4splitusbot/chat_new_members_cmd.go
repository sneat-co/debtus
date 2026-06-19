package cmds4splitusbot

import (
	"context"
	"fmt"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsdal"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot/cmds4anybot"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/splitusbot/facade4splitusbot"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/sneat-co/sneat-translations/trans"
)

const NewChatMembersCommandCode = "new_chat_members"

var newChatMembersCommand = botsfw.Command{
	Code:       NewChatMembersCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()

		newMembersMessage := whc.Input().(botinput.NewChatMembersMessage)

		newMembers := newMembersMessage.NewChatMembers()

		{ // filter out sneatbots
			j := 0
			for _, member := range newMembers {
				if !member.IsBotUser() {
					newMembers[j] = member
					j += 1
				}
			}
			newMembers = newMembers[:j]
		}

		if len(newMembers) == 0 {
			return
		}

		var newUsers []facade4splitusbot.NewUser

		db := whc.DB()
		// Get or create related user records
		if err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			for _, chatMember := range newMembers {
				tgChatMember := chatMember.(tgbotapi.ChatMember)
				var botUser botsdal.BotUser
				if botUser, err = whc.GetBotUser(); err != nil && !dal.IsNotFound(err) {
					return
				}
				if !botUser.Record.Exists() {
					botUser.Data = &botsfwmodels.PlatformUserBaseDbo{
						BotBaseData: botsfwmodels.BotBaseData{
							DtCreated: time.Now(),
						},
					}
					if err = tx.Set(ctx, botUser.Record); err != nil {
						return
					}
				}
				newUsers = append(newUsers, facade4splitusbot.NewUser{
					Name:             tgChatMember.GetFullName(),
					PlatformUserData: botUser.Data,
					ChatMember:       chatMember,
				})
			}
			return
		}); err != nil {
			return
		}

		var splitusSpace models4splitus.SplitusSpaceEntry
		//if splitusSpace, err = shared_space.GetSpaceEntryByCallbackUrl(whc, nil); err != nil {
		//	return
		//}
		var spaceID coretypes.SpaceID = "TODO-implement-determining-space-id" //
		if splitusSpace, newUsers, err = facade4splitusbot.AddUsersToTheGroupAndOutstandingBills(whc.Context(), spaceID, newUsers); err != nil {
			return
		}

		if len(newUsers) == 0 {
			return
		}

		createWelcomeMsg := func(member botinput.Actor) botmsg.MessageFromBot {
			m := whc.NewMessageByCode(trans.MESSAGE_TEXT_USER_JOINED_GROUP, member.GetFirstName())
			m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					{
						Text: whc.CommandText(trans.SettingsButtonText, cmds4anybot.SettingsEmoji),
						URL:  fmt.Sprintf("https:/t.me/%v?start=splitusSpace-%v", whc.GetBotCode(), splitusSpace.ID),
					},
				},
			)

			return m
		}
		m = createWelcomeMsg(newUsers[0].ChatMember)

		if len(newUsers) > 1 {
			responder := whc.Responder()
			ctx := whc.Context()
			for _, newUser := range newUsers {
				if _, err = responder.SendMessage(ctx, createWelcomeMsg(newUser.ChatMember), botsfw.BotAPISendMessageOverHTTPS); err != nil {
					return
				}
			}
		}
		return
	},
}
