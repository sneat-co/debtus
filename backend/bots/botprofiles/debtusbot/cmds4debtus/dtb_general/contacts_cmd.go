package dtb_general

import (
	"net/url"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
)

const debtusContactsCommandCode = "debtus_contacts"

const debtorsCommandCode = "debtors"
const creditorsCommandCode = "creditors"

var debtusContactsCommand = botsfw.Command{
	Code:       debtusContactsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   []string{"/debtus_contacts"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, nil, debtusContactsCommandCode)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, callbackUrl, debtusContactsCommandCode)
	},
}

var debtorsCommand = botsfw.Command{
	Code:       debtorsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   []string{"/debtors"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, nil, debtorsCommandCode)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, callbackUrl, debtorsCommandCode)
	},
}

var creditorsCommand = botsfw.Command{
	Code:       creditorsCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Commands:   []string{"/creditors"},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, nil, creditorsCommandCode)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		return debtusContactsAction(whc, callbackUrl, creditorsCommandCode)
	},
}

func debtusContactsAction(whc botsfw.WebhookContext, callbackUrl *url.URL, commandCode string) (m botmsg.MessageFromBot, err error) {
	if callbackUrl != nil {
		if m, err = whc.NewEditMessage(m.Text, botmsg.FormatHTML); err != nil {
			return
		}
	}
	debtusContactsButton := tgbotapi.InlineKeyboardButton{
		Text:         "Debts contacts",
		CallbackData: debtusContactsCommandCode,
	}
	debtorsButton := tgbotapi.InlineKeyboardButton{
		Text:         "Debtors",
		CallbackData: debtorsCommandCode,
	}
	creditorsButton := tgbotapi.InlineKeyboardButton{
		Text:         "Creditors",
		CallbackData: creditorsCommandCode,
	}
	backButton := []tgbotapi.InlineKeyboardButton{
		{
			Text:         "🔙 Back to Debts",
			CallbackData: "debts",
		},
	}
	switch commandCode {
	case debtorsCommandCode:
		m.Text = "<b>Debtors</b>"
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{creditorsButton, debtusContactsButton},
			backButton,
		)
	case creditorsCommandCode:
		m.Text = "<b>Creditors</b>"
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{debtorsButton, debtusContactsButton},
			backButton,
		)
	case debtusContactsCommandCode:
		m.Text = "<b>Debts related contacts</b>"
		m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{creditorsButton, debtorsButton},
			backButton,
		)
	}
	return
	//var user *models.DebutsAppUserDataOBSOLETE
	//if appUser, err := whc.AppUserData(); err != nil {
	//	return m, err
	//} else {
	//	user = appUser.(*models.DebutsAppUserDataOBSOLETE)
	//}
	//var buffer bytes.Buffer
	//buffer.WriteString(fmt.Sprintf("<b>%v</b>\n", whc.Translate(trans.COMMAND_TEXT_CONTACTS)))
	//linker := anybot.NewLinkerFromWhc(whc)
	//contacts := user.Contacts()
	//numFormat := "%0" + strconv.Itoa(len(strconv.Itoa(len(contacts)))) + "d. "
	//if len(contacts) == 0 {
	//	buffer.WriteString(whc.Translate(trans.MESSAGE_TEXT_YOU_HAVE_NO_CONTACTS))
	//} else {
	//	for i, contact := range contacts {
	//		buffer.WriteString(fmt.Sprintf(numFormat, i+1))
	//		buffer.WriteString(fmt.Sprintf(`<a href="%v">%v</a>`, linker.UrlToContact(contact.ContactID), html.EscapeString(contact.UserTitle)))
	//		if contact.Status != "" && contact.Status != models.STATUS_ACTIVE {
	//			buffer.WriteString(" (")
	//			buffer.WriteString(contact.Status)
	//			buffer.WriteString(")")
	//		}
	//		buffer.WriteString("\n")
	//	}
	//}
	//keyboard := tgbotapi.NewInlineKeyboardMarkup(
	//	[]tgbotapi.InlineKeyboardButton{
	//		{
	//			Text:         whc.CommandText(trans.COMMAND_TEXT_REFRESH, emoji.REFRESH_ICON),
	//			CallbackData: debtorsCommandCode + "?do=refresh",
	//		},
	//	},
	//)
	//buffer.WriteString(fmt.Sprintf("\n\nRefreshed on: %v", time.Now()))
	//m = whc.NewMessage(buffer.CallbackData())
	//m.Keyboard = keyboard
	//m.IsEdit = whc.InputType() == botsfw.WebhookInputCallbackQuery
	////if callbackUrl.Query().Get("do") == "refresh" {
	////	if m, err = bot.SendRefreshOrNothingChanged(whc, m); err != nil {
	////		return
	////	}
	////}
	//return
}

//const CONTACT_DETAILS_COMMAND = "contact_details"
//
//var ContactDetailsCommand = botsfw.Command{
//	Code:     debtorsCommandCode,
//	Commands: trans.Commands(debtorsCommandCode),
//	CallbackAction: func(whc botsfw.WebhookContext, _ *url.URL) (m botmsg.MessageFromBot, err error) {
//		keyboard := tgbotapi.NewInlineKeyboardMarkup(
//			[]tgbotapi.InlineKeyboardButton{
//				{
//					Text:         whc.CommandText(trans.COMMAND_TEXT_LANGUAGE, emoji.EARTH_ICON),
//					CallbackData: SETTINGS_LOCALE_LIST_CALLBACK_PATH,
//				},
//			},
//		)
//		messageText := whc.NewMessageByCode(trans.MESSAGE_TEXT_CONTACT_DETAILS)
//		m.TelegramEditMessageText = telegram.EditMessageOnCallbackQuery(whc.Input().(botsfw.WebhookCallbackQuery), "HTML", messageText)
//		m.TelegramEditMessageText.ReplyMarkup = keyboard
//		return
//	},
//}
//
//const DELETE_CONTACT_COMMAND = "delete_contact"
//
//var DeleteContactCommand = botsfw.Command{
//	Code:     DELETE_CONTACT_COMMAND,
//	Commands: trans.Commands(debtorsCommandCode),
//	CallbackAction: func(whc botsfw.WebhookContext, _ *url.URL) (m botmsg.MessageFromBot, err error) {
//
//		return
//	},
//}
//
//const EDIT_CONTACT_NAME_COMMAND = "edit_contact_name"
//
//var EditContactNameCommand = botsfw.Command{
//	Code:     EDIT_CONTACT_NAME_COMMAND,
//	Commands: trans.Commands(debtorsCommandCode),
//	CallbackAction: func(whc botsfw.WebhookContext, _ *url.URL) (m botmsg.MessageFromBot, err error) {
//
//		return
//	},
//}
