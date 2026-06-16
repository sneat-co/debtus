package dtb_general

import (
	"context"
	"fmt"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/strongo/strongoapp"
)

var deleteAll = func(ctx context.Context, botCode, botChatID string) error {
	return dal4debtus.Default.Admin.DeleteAll(ctx, botCode, botChatID)
}

var deleteAllCommand = botsfw.Command{
	Code:       "delete_all",
	InputTypes: []botinput.Type{botinput.TypeText},
	Icon:       emoji.MAIN_MENU_ICON,
	Commands:   []string{"/delete_all"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		botSettings := whc.GetBotSettings()
		if botSettings.Env != strongoapp.LocalHostEnv && botSettings.Env != "dev" {
			return whc.NewMessage(fmt.Sprintf("This command supported just in development, got botSettings.Env: %v", botSettings.Env)), nil
		} else if botSettings.Env == "prod" {
			return whc.NewMessage("This command supported production environment"), nil
		}

		// We create a success message ahead of actual operation as keyboard creation will fail once user deleted.
		m = whc.NewMessage("Deleted all records")
		if err = SetMainMenuKeyboard(whc, &m); err != nil {
			return
		}

		var chatID string
		if chatID, err = whc.Input().BotChatID(); err != nil {
			return
		}

		if err = deleteAll(whc.Context(), botSettings.Code, chatID); err != nil {
			return
		}

		return
	},
}
