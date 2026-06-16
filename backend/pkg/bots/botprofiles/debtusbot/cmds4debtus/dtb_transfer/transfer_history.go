package dtb_transfer

import (
	"bytes"
	"fmt"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/utm"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-bots/pkg/bots/utm4bots"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"

	//ptime "github.com/yaa110/go-persian-calendar"
	"net/url"
	"strings"
	"time"

	"github.com/sneat-co/sneat-translations/emoji"
)

const HistoryTopLimit = 5
const HistoryMoreLimit = 10

const debtsHistoryCommandCode = "debts_history"

var historyCommand = botsfw.Command{
	Code:     debtsHistoryCommandCode,
	Icon:     emoji.HISTORY_ICON,
	Title:    trans.COMMAND_TEXT_HISTORY,
	Commands: trans.Commands(trans.COMMAND_HISTORY, emoji.HISTORY_ICON), // TODO: Check icon!
	Titles:   map[string]string{botsfw.ShortTitle: emoji.HISTORY_ICON},  // TODO: Check icon!
	TextAction: func(whc botsfw.WebhookContext, text string) (m botmsg.MessageFromBot, err error) {
		return showHistoryCard(whc, "", HistoryTopLimit)
	},
	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
		spaceRef := bothelper.GetSpaceRefFromUrl(callbackUrl)
		if m, err = showHistoryCard(whc, spaceRef, HistoryMoreLimit); err != nil {
			return
		}
		keyboard := m.Keyboard
		if m, err = whc.NewEditMessage(m.Text, m.Format); err != nil {
			return
		}
		m.Keyboard = keyboard

		return
	},
}

func showHistoryCard(whc botsfw.WebhookContext, spaceRef coretypes.SpaceRef, limit int) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()

	var transfers []models4debtus.TransferEntry
	var hasMore bool

	if appUserID := whc.AppUserID(); appUserID != "" {
		if transfers, hasMore, err = dal4debtus.Default.Transfer.LoadTransfersByUserID(ctx, appUserID, 0, limit); err != nil {
			return m, err
		}
	}
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	if len(transfers) == 0 {
		m = whc.NewMessage(whc.Translate(trans.MESSAGE_TEXT_HISTORY_NO_RECORDS) + common4debtus.HORIZONTAL_LINE + dtb_general.AdSlot(whc, UtmCampaignTransferHistory))
	} else {
		m = whc.NewMessage(whc.Translate(
			trans.MESSAGE_TEXT_HISTORY_LIST,
			whc.Translate(trans.MESSAGE_TEXT_HISTORY_HEADER),
			len(transfers),
			transferHistoryRows(whc, transfers),
		) + common4debtus.HORIZONTAL_LINE + dtb_general.AdSlot(whc, UtmCampaignTransferHistory))
		if hasMore {
			//api4transfers = api4transfers[:limit]
			utmParams := utm4bots.FillUtmParams(whc, utm.Params{Campaign: UtmCampaignTransferHistory})
			inlineKeyboard = append(inlineKeyboard, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonURL(
					whc.Translate(trans.INLINE_BUTTON_SHOW_FULL_HISTORY),
					//fmt.Sprintf("transfer-history?offset=%v", len(api4transfers)),
					fmt.Sprintf("https://debtus.app/%v/history?user=%v#%v", whc.Locale().SiteCode(), whc.AppUserID(), utmParams),
				),
			})
		}
	}
	inlineKeyboard = append(inlineKeyboard, []tgbotapi.InlineKeyboardButton{
		{
			Text:         "🔙 " + whc.Translate(trans.BackToDebtsMenu),
			CallbackData: "debts",
		},
	})
	m.Keyboard = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: inlineKeyboard,
	}
	m.Format = botmsg.FormatHTML
	m.DisableWebPagePreview = true
	return m, nil
}

const UtmCampaignTransferHistory = "transfer_history"

func transferHistoryRows(whc botsfw.WebhookContext, transfers []models4debtus.TransferEntry) string {
	var s bytes.Buffer
	ctx := whc.Context()
	for _, transfer := range transfers {
		isCreator := whc.AppUserID() == transfer.Data.CreatorUserID
		var counterpartyName string
		if isCreator {
			counterpartyName = transfer.Data.Counterparty().ContactName
		} else {
			counterpartyName = transfer.Data.Creator().ContactName
		}
		amount := fmt.Sprintf(`<a href="%v">%s</a>`,
			common4debtus.GetTransferUrlForUser(
				ctx,
				transfer.ID,
				whc.AppUserID(),
				whc.Locale(),
				utm4bots.NewParams(whc, "history"),
			),
			transfer.Data.GetAmount(),
		)
		if (isCreator && transfer.Data.Direction() == models4debtus.TransferDirectionUser2Counterparty) || (!isCreator && transfer.Data.Direction() == models4debtus.TransferDirectionCounterparty2User) {
			s.WriteString(whc.Translate(trans.MESSAGE_TEXT_HISTORY_ROW_FROM_USER_WITH_NAME, shortDate(transfer.Data.DtCreated, whc), counterpartyName, amount))
		} else {
			s.WriteString(whc.Translate(trans.MESSAGE_TEXT_HISTORY_ROW_TO_USER_WITH_NAME, shortDate(transfer.Data.DtCreated, whc), counterpartyName, amount))
		}

		if transfer.Data.HasInterest() {
			s.WriteString("\n")
			common4debtus.WriteTransferInterest(&s, transfer, whc)
		}
		s.WriteString("\n\n")
	}
	return strings.TrimSpace(s.String())
}

var transferHistoryCallbackCommand = botsfw.NewCallbackCommand("transfer_history", callbackTransferHistory)

func callbackTransferHistory(whc botsfw.WebhookContext, _ *url.URL) (botmsg.MessageFromBot, error) {
	return whc.NewMessage("TODO: Show more history records"), nil
}

func shortDate(t time.Time, translator i18n.SingleLocaleTranslator) string {
	switch translator.Locale().Code5 {
	case i18n.LocaleCodeEnUS:
		return t.Format("02 Jan 2006")
	//case i18n.LocaleCodeFaIR:
	//	pt := ptime.New(t)
	//	return pt.Format("dd MMM yyyy")
	default:
		month := t.Format("Jan")
		return fmt.Sprintf("%v %v %v", t.Format("02"), translator.Translate(month), t.Format("2006"))
	}
}
