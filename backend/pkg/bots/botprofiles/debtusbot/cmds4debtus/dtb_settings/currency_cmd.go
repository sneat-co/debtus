package dtb_settings

import (
	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const AskCurrencySettingCommandCode = "ask_currency_settings"

var AskCurrencySettingsCommand = botsfw.Command{
	Code:       AskCurrencySettingCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Replies:    []botsfw.Command{SetPrimaryCurrency},
	Commands:   []string{"\xF0\x9F\x92\xB1"},
	Icon:       emoji.CURRENCY_EXCAHNGE_ICON,
	Title:      trans.COMMAND_TEXT_SETTINGS_PRIMARY_CURRENCY,
	Action: func(whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		m := whc.NewMessageByCode(trans.MESSAGE_TEXT_ASK_PRIMARY_CURRENCY)
		m.Keyboard = tgbotapi.NewReplyKeyboardUsingStrings([][]string{
			{
				"€ - Euro ",
				"$ - USD",
				"₽ - RUB",
			},
			{
				"Other",
			},
		})
		whc.ChatData().SetAwaitingReplyTo(AskCurrencySettingCommandCode)
		return m, nil
	},
}

var runUserWorker = dal4userus.RunUserWorker

var SetPrimaryCurrency = botsfw.Command{
	Code:       "set_primary_currency",
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		logus.Debugf(ctx, "SetPrimaryCurrency.Action()")
		whc.ChatData().SetAwaitingReplyTo("")
		primaryCurrency := whc.Input().(botinput.TextMessage).Text()
		userID := whc.AppUserID()
		userContext := facade.NewUserContext(userID)
		ctxWithUser := facade.NewContextWithUser(ctx, userContext)
		err = runUserWorker(ctxWithUser, true,
			func(ctx facade.ContextWithUser, tx dal.ReadwriteTransaction, userWorkerParams *dal4userus.UserWorkerParams) error {
				userWorkerParams.UserUpdates, err = userWorkerParams.User.Data.SetPrimaryCurrency(money.CurrencyCode(primaryCurrency))
				return err
			})
		if err != nil {
			return m, err
		}
		return whc.NewMessageByCode(trans.MESSAGE_TEXT_PRIMARY_CURRENCY_IS_SET_TO, whc.Input().(botinput.TextMessage).Text()), err
	},
}
