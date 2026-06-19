package dtb_general

import (
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/botsfwconst"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
)

const InvitesShotCommandText = emoji.PRESENT_ICON

// These commands are required for the main menu because of circular references
var _lendCommand = botsfw.Command{Code: "lend", Title: trans.COMMAND_TEXT_GAVE, Icon: emoji.GIVE_ICON}
var _borrowCommand = botsfw.Command{Code: "borrow", Title: trans.COMMAND_TEXT_GOT, Icon: emoji.TAKE_ICON}
var _returnCommand = botsfw.Command{Code: "return", Title: trans.COMMAND_TEXT_RETURN, Icon: emoji.RETURN_BACK_ICON}

func MainMenuKeyboardOnReceiptAck(whc botsfw.WebhookContext) *tgbotapi.ReplyKeyboardMarkup {
	return mainMenuTelegramKeyboard(whc, getMainMenuParams(whc, true))
}

type mainMenuParams struct {
	showBalanceAndHistory bool
	showReturn            bool
}

func getMainMenuParams(_ botsfw.WebhookContext, onReceiptAck bool) (params mainMenuParams) {
	//var (
	//	user *models.DebutsAppUserDataOBSOLETE
	//	isAppUser bool
	//)
	//ctx := whc.Context()
	//if userEntity, err := whc.AppUserData(); err != nil {
	//	logus.Errorf(c, "Failed to get user: %v", err)
	//} else if user, isAppUser = userEntity.(*models.DebutsAppUserDataOBSOLETE); !isAppUser {
	//	logus.Errorf(c, "Failed to case user to *models.DebutsAppUserDataOBSOLETE: %T", userEntity)
	//} else if onReceiptAck || !user.Balance().IsZero() {
	//	params.showReturn = true
	//}
	params.showBalanceAndHistory = onReceiptAck //|| (user != nil && user.CountOfTransfers > 0)
	return
}

func debtsMainMenuInlineKeyboard(whc botsfw.WebhookContext, spaceRef coretypes.SpaceRef) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{
			{
				Text:         _lendCommand.DefaultTitle(whc),
				CallbackData: "lend",
			},
			{
				Text:         _borrowCommand.DefaultTitle(whc),
				CallbackData: "borrow",
			},
		},
		{
			{
				Text:         "👫 " + whc.Translate(trans.COMMAND_TEXT_BALANCE),
				CallbackData: bothelper.GetSpaceCallbackData("debts_balance", spaceRef),
			},
			{
				Text:         "👫 " + whc.Translate(trans.COMMAND_TEXT_HISTORY),
				CallbackData: bothelper.GetSpaceCallbackData("debts_history", spaceRef),
			},
		},
		{
			{
				Text:         "👫 " + whc.Translate(trans.DebtsRelatedContacts),
				CallbackData: "debtus_contacts",
			},
		},
		{
			{
				Text: "💻 " + whc.Translate(trans.OpenInApp),
				WebApp: &tgbotapi.WebAppInfo{
					Url: bothelper.GetSpacePageUrl(spaceRef, "debtus"),
				},
			},
		},
	}
}
func mainMenuTelegramKeyboard(whc botsfw.WebhookContext, params mainMenuParams) *tgbotapi.ReplyKeyboardMarkup {
	firstRow := []string{
		_lendCommand.DefaultTitle(whc),
		_borrowCommand.DefaultTitle(whc),
	}

	if params.showReturn {
		firstRow = append(firstRow, _returnCommand.DefaultTitle(whc))
	}

	buttonRows := make([][]string, 0, 3)
	buttonRows = append(buttonRows, firstRow)

	if params.showBalanceAndHistory {
		buttonRows = append(buttonRows, []string{
			whc.CommandText(trans.COMMAND_TEXT_BALANCE, emoji.BALANCE_ICON),
			//whc.CommandText(trans.COMMAND_TEXT_CONTACTS, emoji.MAN_AND_WOMAN),
			whc.CommandText(trans.COMMAND_TEXT_HISTORY, emoji.HISTORY_ICON),
		})
	}

	buttonRows = append(buttonRows, []string{
		//whc.CommandText(trans.COMMAND_TEXT_SETTING, emoji.SETTINGS_ICON),
		//whc.CommandText(trans.COMMAND_TEXT_HIGH_FIVE, emoji.BULB_ICON),
		//whc.CommandText(trans.COMMAND_TEXT_HELP, emoji.HELP_ICON),
		emoji.SETTINGS_ICON,
		emoji.MAN_AND_WOMAN,
		emoji.PUBLIC_LOUDSPEAKER,
		emoji.STAR_ICON,
		emoji.HELP_ICON,
	})

	return tgbotapi.NewReplyKeyboardUsingStrings(buttonRows)
}
func SetMainMenuKeyboard(whc botsfw.WebhookContext, m *botmsg.MessageFromBot) error {
	params := getMainMenuParams(whc, true)
	switch botPlatformID := botsfwconst.Platform(whc.BotPlatform().ID()); botPlatformID {
	case telegram.PlatformID:
		m.Keyboard = mainMenuTelegramKeyboard(whc, params)
		return nil
	//case viber.PlatformID:
	//m.Keyboard = mainMenuViberKeyboard(whc, params)
	//case fbm.PlatformID:
	//if m.Text != "" {
	//	panic("FBM does not support message text and attachments in the same request.")
	//}
	//m.FbmAttachment = mainMenuFbmAttachment(whc, params)
	default:
		return fmt.Errorf("unsupported platform id=%s", botPlatformID)
	}
}

//func mainMenuFbmAttachment(whc botsfw.WebhookContext, params mainMenuParams) *fbmbotapi.RequestAttachment {
//	attachment := &fbmbotapi.RequestAttachment{
//		ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//		Payload: fbmbotapi.NewListTemplate(
//			fbmbotapi.TopElementStyleCompact,
//			fbmbotapi.NewRequestElementWithDefaultAction(
//				"Debtus",
//				"Tracks personal debts (auto-reminders to your debtors)",
//				fbmbotapi.NewDefaultActionWithWebURL(fbmbotapi.RequestWebURLAction{MessengerExtensions: true, URL: "https://debtus.app/app/?page=debts&lang=ru"}),
//				fbmbotapi.NewRequestWebURLButtonWithRatio(emoji.CURRENCY_EXCAHNGE_ICON+" Record new debt", "https://debtus.app/app/?page=new-debt&lang=ru", "full"),
//			),
//			fbmbotapi.NewRequestElementWithDefaultAction(
//				"Current balance",
//				"You owe $100",
//				fbmbotapi.NewDefaultActionWithWebURL(fbmbotapi.RequestWebURLAction{MessengerExtensions: true, URL: "https://debtus.app/app/?page=debts&lang=ru"}),
//				fbmbotapi.NewRequestWebURLButtonWithRatio(emoji.BALANCE_ICON+" Record return", "https://debtus.app/app/?page=return&lang=ru", "full"),
//			),
//			fbmbotapi.NewRequestElementWithDefaultAction(
//				"History",
//				"Last transfer: $100 to Jack Smith",
//				fbmbotapi.NewDefaultActionWithWebURL(fbmbotapi.RequestWebURLAction{MessengerExtensions: true, URL: "https://debtus.app/app/?page=history&lang=ru"}),
//				fbmbotapi.NewRequestWebURLButtonWithRatio(emoji.HISTORY_ICON+" View full history", "https://debtus.app/app/?page=history&lang=ru", "full"),
//			),
//			fbmbotapi.NewRequestElementWithDefaultAction(
//				"Settings",
//				"You can change language, notification preferences, etc.",
//				fbmbotapi.NewDefaultActionWithWebURL(fbmbotapi.RequestWebURLAction{MessengerExtensions: true, URL: "https://debtus.app/app/?page=debts&lang=ru"}),
//				fbmbotapi.NewRequestWebURLButtonWithRatio(emoji.SETTINGS_ICON+" Edit my preferences", "https://debtus.app/app/?page=settings&lang=ru", "full"),
//			),
//		),
//	}
//	logus.Debugf(whc.Context(), "First element: %v", attachment.Payload.RequestAttachmentListTemplate.Elements[0])
//	return attachment
//}

//const (
//	UTM_CAMPAIGN_BOT_MAIN_MENU = "bot_main_menu"
//)

//func mainMenuViberKeyboard(whc botsfw.WebhookContext, params mainMenuParams) *viberinterface.Keyboard {
//	var buttons []viberinterface.Button
//	lendingText := _lendCommand.DefaultTitle(whc)
//	borrowText := _borrowCommand.DefaultTitle(whc)
//	const (
//		maxColumns = 6
//		in3columns = maxColumns / 3
//		in2columns = maxColumns / 2
//	)
//	if params.showReturn {
//		returnText := _returnCommand.DefaultTitle(whc)
//		buttons = []viberinterface.Button{
//			{
//				Columns:    in3columns,
//				BgColor:    debtusviberbots.ButtonBgColor,
//				Text:       lendingText,
//				ActionType: viberinterface.ActionTypeOpenUrl,
//				ActionBody: anybot.GetNewDebtPageUrl(whc, models.TransferDirectionUser2Counterparty, UTM_CAMPAIGN_BOT_MAIN_MENU),
//			},
//			{
//				Columns:    in3columns,
//				BgColor:    debtusviberbots.ButtonBgColor,
//				Text:       borrowText,
//				ActionType: viberinterface.ActionTypeOpenUrl,
//				ActionBody: anybot.GetNewDebtPageUrl(whc, models.TransferDirectionCounterparty2User, UTM_CAMPAIGN_BOT_MAIN_MENU),
//			},
//			{Columns: in3columns, ActionBody: returnText, Text: returnText, BgColor: debtusviberbots.ButtonBgColor},
//		}
//	} else {
//		buttons = []viberinterface.Button{
//			{Columns: in2columns, ActionBody: lendingText, Text: lendingText, BgColor: debtusviberbots.ButtonBgColor},
//			{Columns: in2columns, ActionBody: borrowText, Text: borrowText, BgColor: debtusviberbots.ButtonBgColor},
//		}
//	}
//	if params.showBalanceAndHistory {
//		userID := whc.AppUserID()
//		locale := whc.Locale()
//		balanceUrl := anybot.GetBalanceUrlForUser(userID, locale, whc.BotPlatform().ContactID(), whc.GetBotCode())
//		historyUrl := anybot.GetHistoryUrlForUser(userID, locale, whc.BotPlatform().ContactID(), whc.GetBotCode())
//		buttons = append(buttons, []viberinterface.Button{
//			{Columns: in2columns, ActionType: "open-url", ActionBody: balanceUrl, Text: whc.CommandText(trans.COMMAND_TEXT_BALANCE, emoji.BALANCE_ICON), BgColor: debtusviberbots.ButtonBgColor},
//			{Columns: in2columns, ActionType: "open-url", ActionBody: historyUrl, Text: whc.CommandText(trans.COMMAND_TEXT_HISTORY, emoji.HISTORY_ICON), BgColor: debtusviberbots.ButtonBgColor},
//		}...)
//	}
//	{ // Last row
//		settings := whc.CommandText(trans.COMMAND_TEXT_SETTING, emoji.SETTINGS_ICON)
//		rate := whc.CommandText(trans.COMMAND_TEXT_HIGH_FIVE, emoji.STAR_ICON)
//		help := whc.CommandText(trans.COMMAND_TEXT_HELP, emoji.HELP_ICON)
//		buttons = append(buttons, []viberinterface.Button{
//			{Columns: in3columns, ActionBody: settings, Text: settings, BgColor: debtusviberbots.ButtonBgColor},
//			{Columns: in3columns, ActionBody: rate, Text: rate, BgColor: debtusviberbots.ButtonBgColor},
//			{Columns: in3columns, ActionBody: help, Text: help, BgColor: debtusviberbots.ButtonBgColor},
//		}...)
//	}
//
//	return viberinterface.NewKeyboard(debtusviberbots.KeyboardBgColor, false, buttons...)
//}
