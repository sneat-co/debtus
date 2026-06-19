package dtb_transfer

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/dtb_common"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dal4reminders"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"

	"github.com/sneat-co/sneat-translations/emoji"
)

var returnCallbackCommand = botsfw.NewCallbackCommand(dtb_common.CallbackDebtReturnedPath, ProcessReturnAnswer)

func ProcessReturnAnswer(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	//
	ctx := whc.Context()
	logus.Debugf(ctx, "ProcessReturnAnswer()")
	q := callbackUrl.Query()
	reminderID := q.Get("reminder")
	if reminderID == "" {
		return m, fmt.Errorf("missing reminder ContactID")
	}
	var transferID string
	if reminder, err := dal4reminders.SetReminderStatus(ctx, reminderID, "", dbo4reminders.ReminderStatusUsed, time.Now()); err != nil {
		return m, err
	} else {
		transferID = reminder.Data.TargetID
	}

	howMuch := q.Get("how-much")
	transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, nil, transferID)
	if err != nil {
		return m, err
	}
	switch howMuch {
	case "":
		return m, fmt.Errorf("missing how-much parameter")
	case dtb_common.ReturnedFully:
		return ProcessFullReturn(whc, transfer)
	case dtb_common.ReturnedPartially:
		return ProcessPartialReturn(whc, transfer)
	case dtb_common.ReturnedNothing:
		return processNoReturn(whc, reminderID, transfer)
	default:
		return m, fmt.Errorf("unknown how-much parameter value: %v", howMuch)
	}
}

const commandCodeEnableReminderAgain = "enable_reminder_again"

var enableReminderAgainCallbackCommand = botsfw.NewCallbackCommand(commandCodeEnableReminderAgain, func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()
	logus.Debugf(ctx, "enableReminderAgainCallbackCommand()")
	q := callbackUrl.Query()
	var (
		reminderID string
		transfer   models4debtus.TransferEntry
	)
	if reminderID = q.Get("reminder"); reminderID == "" {
		err = fmt.Errorf("parameter 'reminder' is empty")
		return
	}
	if transfer.ID = q.Get("transfer"); transfer.ID == "" {
		err = fmt.Errorf("parameter 'transfer' is empty")
		return
	}

	if transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, nil, transfer.ID); err != nil {
		return
	}

	return askWhenToRemindAgain(whc, reminderID, transfer)
})

func ProcessFullReturn(whc botsfw.WebhookContext, transfer models4debtus.TransferEntry) (m botmsg.MessageFromBot, err error) {
	amountValue := transfer.Data.GetOutstandingValue(time.Now())
	if amountValue == 0 {
		return dtb_general.EditReminderMessage(whc, transfer, whc.Translate(trans.MESSAGE_TEXT_TRANSFER_ALREADY_FULLY_RETURNED))
	} else if amountValue < 0 {
		err = fmt.Errorf("data integrity error -> transfer.GetOutstandingValue():%v < 0", amountValue)
		return
	}

	amount := money.NewAmount(transfer.Data.GetAmount().Currency, amountValue)

	var (
		counterpartyID string
		direction      models4debtus.TransferDirection
	)
	userID := whc.AppUserID()
	if transfer.Data.CreatorUserID == userID {
		counterpartyID = transfer.Data.Counterparty().ContactID
		switch transfer.Data.Direction() {
		case models4debtus.TransferDirectionCounterparty2User:
			direction = models4debtus.TransferDirectionUser2Counterparty
		case models4debtus.TransferDirectionUser2Counterparty:
			direction = models4debtus.TransferDirectionCounterparty2User
		default:
			return m, fmt.Errorf("transfer %v has unknown direction '%v'", transfer.ID, transfer.Data.Direction())
		}
	} else if transfer.Data.Counterparty().UserID == userID {
		switch transfer.Data.Direction() {
		case models4debtus.TransferDirectionCounterparty2User:
		case models4debtus.TransferDirectionUser2Counterparty:
		default:
			return m, fmt.Errorf("transfer %v has unknown direction '%v'", transfer.ID, transfer.Data.Direction())
		}
		counterpartyID = transfer.Data.Creator().ContactID
		direction = transfer.Data.Direction()
	}

	if m, err = dtb_general.EditReminderMessage(whc, transfer, whc.Translate(trans.MESSAGE_TEXT_REPLIED_DEBT_RETURNED_FULLY)); err != nil {
		return
	}

	if _, err = whc.Responder().SendMessage(whc.Context(), m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
		return m, err
	}

	if m, err = CreateReturnAndShowReceipt(whc, transfer.ID, counterpartyID, direction, amount); err != nil {
		return m, err
	}

	reportReminderIsActed(whc, "reminder-acted-returned-fully")

	//TODO: edit message
	return m, err
}

func ProcessPartialReturn(whc botsfw.WebhookContext, transfer models4debtus.TransferEntry) (botmsg.MessageFromBot, error) {
	var counterpartyID string
	switch whc.AppUserID() {
	case transfer.Data.CreatorUserID:
		counterpartyID = transfer.Data.Counterparty().ContactID
	case transfer.Data.Counterparty().UserID:
		counterpartyID = transfer.Data.Creator().ContactID
	default:
		return botmsg.MessageFromBot{}, fmt.Errorf("whc.AppUserID()=%v not in (transfer.CreatorUserID=%v, transfer.Counterparty().UserID=%v)",
			whc.AppUserID(), transfer.Data.CreatorUserID, transfer.Data.Counterparty().UserID)
	}
	chatEntity := whc.ChatData()
	chatEntity.SetAwaitingReplyTo("")
	chatEntity.AddWizardParam(WizardParamCounterparty, counterpartyID)
	chatEntity.AddWizardParam(WizardParamTransfer, transfer.ID)
	chatEntity.AddWizardParam("currency", string(transfer.Data.Currency))

	reportReminderIsActed(whc, "reminder-acted-returned-partially")

	return askHowMuchHaveBeenReturnedCommand.Action(whc)
}

func askWhenToRemindAgain(whc botsfw.WebhookContext, reminderID string, transfer models4debtus.TransferEntry) (m botmsg.MessageFromBot, err error) {
	if m, err = dtb_general.EditReminderMessage(whc, transfer, whc.Translate(trans.MESSAGE_TEXT_ASK_WHEN_TO_REMIND_AGAIN)); err != nil {
		return
	}
	callbackData := fmt.Sprintf("%s?id=%s&in=%s", dtb_common.CallbackRemindAgain, reminderID, "%v")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{
				Text:         emoji.CALENDAR_ICON + " " + whc.Translate(trans.COMMAND_TEXT_SET_DATE),
				CallbackData: fmt.Sprintf("%s?id=%s", setNextReminderDateCommandCode, reminderID),
			},
		},
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_TOMORROW), CallbackData: fmt.Sprintf(callbackData, "24h")},
			{Text: whc.Translate(trans.COMMAND_TEXT_DAY_AFTER_TOMORROW), CallbackData: fmt.Sprintf(callbackData, "48h")},
		},
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_IN_1_WEEK), CallbackData: fmt.Sprintf(callbackData, "168h")},
			{Text: whc.Translate(trans.COMMAND_TEXT_IN_1_MONTH), CallbackData: fmt.Sprintf(callbackData, "720h")},
		},
		[]tgbotapi.InlineKeyboardButton{
			{Text: whc.Translate(trans.COMMAND_TEXT_DISABLE_REMINDER), CallbackData: fmt.Sprintf(callbackData, dtb_common.C_REMIND_IN_DISABLE)},
		},
	)

	if whc.GetBotSettings().Env == "dev" {
		keyboard.InlineKeyboard = append(
			[][]tgbotapi.InlineKeyboardButton{
				{
					{
						Text:         whc.Translate(trans.COMMAND_TEXT_IN_FEW_MINUTES),
						CallbackData: fmt.Sprintf(callbackData, "1m"),
					},
				},
			},
			keyboard.InlineKeyboard...,
		)
	}
	m.IsEdit = true
	m.Keyboard = keyboard
	return
}

func processNoReturn(whc botsfw.WebhookContext, reminderID string, transfer models4debtus.TransferEntry) (m botmsg.MessageFromBot, err error) {
	return askWhenToRemindAgain(whc, reminderID, transfer)
}

const setNextReminderDateCommandCode = "set_next_reminder_date"

var setNextReminderDateCallbackCommand = botsfw.Command{
	Code:       setNextReminderDateCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText, botinput.TypeCallbackQuery},
	Action: func(whc botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		m, date, err := processSetDate(whc)
		if !date.IsZero() {
			chatEntity := whc.ChatData()

			reminderID := chatEntity.GetWizardParam(WizardParamReminder)
			if err != nil {
				return m, fmt.Errorf("failed to decode reminder id: %w", err)
			}
			now := time.Now()
			sinceToday := now.Sub(now.Truncate(24 * time.Hour))

			date = date.Add(sinceToday)
			remindInDuration := date.Sub(now)
			return rescheduleReminder(whc, reminderID, remindInDuration)
		}
		return m, err
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()

		reminderID := callbackUrl.Query().Get("id")
		if reminderID == "" {
			return m, fmt.Errorf("missing reminder ContactID")
		}

		chatEntity := whc.ChatData()
		chatEntity.SetAwaitingReplyTo(setNextReminderDateCommandCode)
		chatEntity.AddWizardParam(WizardParamReminder, reminderID)

		reminder, err := dal4reminders.GetReminderByID(ctx, nil, reminderID)
		if err != nil {
			return m, fmt.Errorf("failed to get reminder by id: %w", err)
		}
		transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, nil, reminder.Data.TargetID)
		if err != nil {
			return m, fmt.Errorf("failed to get transfer by id: %w", err)
		}

		if m, err = dtb_general.EditReminderMessage(whc, transfer, whc.Translate(trans.MESSAGE_TEXT_ASK_WHEN_TO_REMIND_AGAIN)); err != nil {
			return
		}

		if _, err = whc.Responder().SendMessage(ctx, m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
			return m, err
		}

		m = whc.NewMessageByCode(trans.MESSAGE_TEXT_ASK_DATE_TO_REMIND)

		return m, err
	},
}
