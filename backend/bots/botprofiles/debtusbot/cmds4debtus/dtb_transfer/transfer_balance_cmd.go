package dtb_transfer

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	bots "github.com/sneat-co/debtus/backend/bots/botscompat"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
)

const debtsBalanceCommandCode = "debts_balance"

var debtsBalanceCommand = botsfw.Command{ //TODO: Write unit tests!
	Code:           debtsBalanceCommandCode,
	Title:          trans.COMMAND_TEXT_BALANCE,
	Icon:           emoji.BALANCE_ICON,
	Commands:       trans.Commands(trans.COMMAND_BALANCE),
	TextAction:     balanceTextAction,
	CallbackAction: balanceCallbackAction,
}

func balanceCallbackAction(whc botsfw.WebhookContext, u *url.URL) (m botmsg.MessageFromBot, err error) {
	spaceRef := bothelper.GetSpaceRefFromUrl(u)
	return balanceAction(whc, spaceRef)
}

func balanceTextAction(whc botsfw.WebhookContext, text string) (m botmsg.MessageFromBot, err error) {
	_ = text
	ctx := whc.Context()
	appUserID := whc.AppUserID()

	if appUserID == "" {
		err = fmt.Errorf("AppUserID is empty")
		return
	}
	user := dbo4userus.NewUserEntry(appUserID)

	if err = dal4userus.GetUser(ctx, nil, user); err != nil {
		return
	}

	spaceID := user.Data.GetFamilySpaceID()
	spaceRef := coretypes.NewSpaceRef(coretypes.SpaceTypeFamily, spaceID)
	return balanceAction(whc, spaceRef)
}

func balanceAction(whc botsfw.WebhookContext, spaceRef coretypes.SpaceRef) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()

	logus.Debugf(ctx, "debtsBalanceCommand.Action()")

	var buffer bytes.Buffer

	spaceID := spaceRef.SpaceID()

	debtusSpace := models4debtus.NewDebtusSpaceEntry(spaceID)
	if err = models4debtus.GetDebtusSpace(ctx, nil, debtusSpace); err != nil && !dal.IsNotFound(err) {
		return
	}

	contactusSpace := dal4contactus.NewContactusSpaceEntry(spaceID)
	if err = dal4contactus.GetContactusSpace(ctx, nil, contactusSpace); err != nil {
		return
	}

	if len(debtusSpace.Data.Balance) == 0 {
		if _, err = buffer.WriteString(whc.Translate(trans.MESSAGE_TEXT_BALANCE_IS_ZERO)); err != nil {
			return
		}
	} else {
		balanceMessageBuilder := NewBalanceMessageBuilder(whc)
		if len(debtusSpace.Data.Contacts) == 0 {
			return m, fmt.Errorf("integrity issue: UserEntry{ContactID=%s} has non zero balance and no contacts", whc.AppUserID())
		}
		buffer.WriteString(fmt.Sprintf("<b>%v</b>", whc.Translate(trans.MESSAGE_TEXT_BALANCE_HEADER)) + common4debtus.HORIZONTAL_LINE)
		linker := bots.NewLinkerFromWhc(whc)
		buffer.WriteString(balanceMessageBuilder.ByContact(ctx, linker, contactusSpace.Data.Contacts, debtusSpace.Data.Contacts))

		var thereAreFewDebtsForSingleCurrency = func() bool {
			//TODO: Duplicate call to Balance() - consider move inside BalanceMessageBuilder
			//logus.Debugf(ctx, "thereAreFewDebtsForSingleCurrency()")
			var currencies []money.CurrencyCode
			for _, counterparty := range debtusSpace.Data.Contacts {
				//logus.Debugf(ctx, "counterparty: %v", counterparty)
				for currency := range counterparty.Balance {
					//logus.Debugf(ctx, "currency: %v", currency)
					for _, curr := range currencies {
						//logus.Debugf(ctx, "curr: %v; curr == currency: %v", curr, curr == currency)
						if curr == currency {
							return true
						}
					}
					currencies = append(currencies, currency)
				}
			}
			//logus.Debugf(ctx, "thereAreFewDebtsForSingleCurrency: %v", currencies)
			return false
		}

		if len(debtusSpace.Data.Contacts) > 1 && thereAreFewDebtsForSingleCurrency() {
			userBalanceWithInterest, err := debtusSpace.Data.BalanceWithInterest(ctx, time.Now())
			if err != nil {
				m := fmt.Sprintf("Failed to get balance with interest for user %v: %v", whc.AppUserID(), err)
				logus.Errorf(ctx, m)
				buffer.WriteString(m)
			} else {
				buffer.WriteString("\n" + strings.Repeat("─", 16) + "\n" + balanceMessageBuilder.ByCurrency(true, userBalanceWithInterest))
			}
		}

		//if len(contacts) > 0 {
		//	//for i, counterparty := range contacts {
		//	//	telegramKeyboard = append(telegramKeyboard, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(counterparty.GetFullName(), fmt.Sprintf("transfer-history?counterparty=%v", counterpartyKeys[i].IntID()))})
		//	//}
		//	telegramKeyboard = append(telegramKeyboard, []tgbotapi.InlineKeyboardButton{
		//		tgbotapi.NewInlineKeyboardButtonData("<", fmt.Sprintf("balance?counterparty=%v", counterpartyKeys[len(counterpartyKeys)-1].IntID())),
		//		tgbotapi.NewInlineKeyboardButtonData(">", fmt.Sprintf("balance?counterparty=%v", counterpartyKeys[0].IntID())),
		//	})
		//}	}
		if debtusSpace.Data.HasDueTransfers {
			m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					{
						Text:         whc.Translate(trans.COMMAND_TEXT_DUE_RETURNS),
						CallbackData: dueReturnsCallbackCommandCode,
					},
				},
				[]tgbotapi.InlineKeyboardButton{
					{
						Text:         whc.Translate(trans.COMMAND_TEXT_INVITE_FIREND),
						CallbackData: "invite",
					},
				},
			)
		}
	}

	buffer.WriteString(common4debtus.HORIZONTAL_LINE)
	//buffer.WriteString(dtb_general.AdSlot(whc, "balance"))
	const THUMB_UP = "👍"
	buffer.WriteString(THUMB_UP + " " + whc.Translate(trans.MESSAGE_TEXT_PLEASE_HELP_MAKE_IT_BETTER))
	if whc.Input().InputType() == botinput.TypeCallbackQuery {
		if m, err = whc.NewEditMessage(buffer.String(), botmsg.FormatHTML); err != nil {
			return
		}
	} else {
		m = whc.NewMessage(buffer.String())
		m.Format = botmsg.FormatHTML
	}

	m.DisableWebPagePreview = true

	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			{
				Text:         "🔙 " + whc.Translate(trans.BackToDebtsMenu),
				CallbackData: "debts",
			},
		},
	}
	m.Keyboard = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	//err = whc.Responder().SendMessage(ctx, m, botsfw.BotAPISendMessageOverHTTPS)
	return m, err
	//SetMainMenuKeyboard(whc, &m) - Bad idea! Need to cleanup AwaitingReplyTo
}
