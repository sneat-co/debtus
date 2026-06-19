package dtb_retention

import (
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

var setAccessGranted = botsfw.SetAccessGranted

var DeleteUserCommand = botsfw.Command{
	Code:       "delete-user",
	InputTypes: []botinput.Type{botinput.TypeText},
	Icon:       emoji.NO_ENTRY_SIGN_ICON,
	Commands:   []string{"/deleteuser"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		err = setAccessGranted(whc, false)
		if err != nil {
			m = whc.NewMessageByCode(trans.MESSAGE_TEXT_FAILED_TO_DELETE_USER, err)
			if err = dtb_general.SetMainMenuKeyboard(whc, &m); err != nil {
				return m, err
			}
			return m, nil
		} else {
			m = whc.NewMessageByCode(trans.MESSAGE_TEXT_USER_DELETED)
			return m, nil
		}
	},
}
