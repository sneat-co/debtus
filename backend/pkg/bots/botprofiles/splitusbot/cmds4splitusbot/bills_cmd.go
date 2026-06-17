package cmds4splitusbot

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

const billsCommandCode = "bills"

var billsCommand = botsfw.Command{
	Code:       billsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   trans.Commands(trans.COMMAND_TEXT_BILLS, "/"+billsCommandCode),
	Icon:       emoji.CLIPBOARD_ICON,
	Title:      trans.COMMAND_TEXT_BILLS,
	Action:     billsAction,
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return billsAction(whc)
	},
}

func billsAction(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	var isInGroup bool
	if isInGroup, err = whc.IsInGroup(); err != nil {
		return
	} else if !isInGroup {
		userID := whc.AppUserID()
		userSplitus := models4splitus.NewSplitusUserEntry(userID)

		var db dal.DB
		if db, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
		if err = db.Get(ctx, userSplitus.Record); err != nil {
			return
		}
		if len(userSplitus.Data.OutstandingBills) == 0 {
			m.Text = whc.Translate("You have no outstanding bills.")
			return
		}

		buf := new(bytes.Buffer)
		_, _ = fmt.Fprintf(buf, "<b>%v</b>\n", whc.Translate("Outstanding bills"))
		var i int
		for _, billJson := range userSplitus.Data.GetOutstandingBills() {
			i++
			_, _ = fmt.Fprintf(buf, "\n%v. %v: %v %v", i, billJson.Name, billJson.Total, billJson.Currency)
		}
		m.Text = buf.String()
		m.Format = botmsg.FormatHTML
		keyboard := tgbotapi.NewInlineKeyboardMarkup()
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			[]tgbotapi.InlineKeyboardButton{{
				Text:         whc.CommandText(trans.COMMAND_TEXT_SETTLE_BILLS, emoji.GREEN_CHECKBOX),
				CallbackData: settleBillsCommandCode,
			}},
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonSwitchInlineQuery(
					whc.CommandText(trans.COMMAND_TEXT_NEW_BILL, emoji.MEMO_ICON),
					"",
				),
			},
			[]tgbotapi.InlineKeyboardButton{
				bothelper.NewGroupTelegramInlineButton(whc, 0),
			},
		)
		m.Keyboard = keyboard
		return
	}
	m.Format = botmsg.FormatHTML
	err = errors.New("not implemented yet")

	//var space dal4spaceus.SpaceEntry
	//if space, err = shared_space.GetSpaceEntryByUrl(whc, nil); err != nil {
	//	return
	//}

	//if space.Data.OutstandingBillsCount == 0 {
	//	mt := "This space has no outstanding bills"
	//	switch whc.InputType() {
	//	case botsfw.WebhookInputCallbackQuery:
	//		m.BotMessage = telegram.CallbackAnswer(tgbotapi.AnswerCallbackQueryConfig{Text: mt})
	//	case botsfw.WebhookInputText:
	//		m.Text = mt
	//	default:
	//		err = errors.New("unknown input type")
	//	}
	//	return
	//}
	//
	//buf := new(bytes.Buffer)
	//buf.WriteString("<b>Outstanding bills</b>\n\n")
	//
	//outstandingBills := space.Data.GetOutstandingBills()
	//
	//var i int
	//for billID, bill := range outstandingBills {
	//	i++
	//	_, _ = fmt.Fprintf(buf, `  %d. <a href="%s">%v</a>`+"\n", i, StartTelegramBotUrl(whc.GetBotCode(), "bill", "id="+billID), bill.UserTitle)
	//}
	//
	//_, _ = fmt.Fprintf(buf, "\nSend /split@%v to close the bills.\nThe debts records will be available in @DebtusBot.", whc.GetBotCode())
	//
	//m.Text = buf.CallbackData()
	return
}
