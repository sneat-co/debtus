package cmds4invites

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-translations/trans"
)

const createInviteCommandCode = "create_invite"

var inviteCommand = botsfw.Command{
	Code:       createInviteCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Commands:   []string{dtb_general.InvitesShotCommandText, "/Пригласить_друга", "/create_invite"},
	Replies: []botsfw.Command{
		AskInviteAddressTelegramCommand,
		AskInviteAddressEmailCommand,
		AskInviteAddressSmsCommand,
	},
	Action: func(whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		m := whc.NewMessageByCode(trans.MESSAGE_TEXT_ABOUT_INVITES)
		m.Keyboard = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonSwitchInlineQuery(AskInviteAddressTelegramCommand.DefaultTitle(whc), "/invite"),
				},
				{
					{
						Text:         AskInviteAddressSmsCommand.DefaultTitle(whc),
						CallbackData: "create_invite?by=sms",
					},
					{
						Text:         AskInviteAddressEmailCommand.DefaultTitle(whc),
						CallbackData: "create_invite?by=email",
					},
				},
			},
		}
		whc.ChatData().SetAwaitingReplyTo(createInviteCommandCode)
		return m, nil
	},
}
