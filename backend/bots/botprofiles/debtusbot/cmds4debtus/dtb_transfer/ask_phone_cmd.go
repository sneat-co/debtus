package dtb_transfer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/update"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/analytics4bots"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-bots/pkg/bots/sneatbots/facade4bots"
	"github.com/sneat-co/debtus/backend/debtus/common4debtus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/general4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/sms"
	"github.com/sneat-co/sneat-translations/emoji"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
	"github.com/strongo/strongoapp/with"
	//"github.com/sneat-co/sneat-go-backend/debtusbot/gae_app/invites"
)

const askPhoneNumberForReceiptCommandCode = "ask_phone_number_for_receipt"

func cleanPhoneNumber(phoneNumber string) string {
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, "(", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, ")", "")
	return phoneNumber
}

var askPhoneNumberForReceiptCommand = botsfw.Command{
	Code:       askPhoneNumberForReceiptCommandCode,
	InputTypes: []botinput.Type{botinput.TypeText},
	Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {
		ctx := whc.Context()
		userCtx := facade4bots.GetUserContext(whc)
		ctxWithUser := facade.NewContextWithUser(ctx, userCtx)
		return m, dal4userus.RunUserWorker(ctxWithUser, true,
			func(ctx facade.ContextWithUser, tx dal.ReadwriteTransaction, params *dal4userus.UserWorkerParams) (err error) {
				logus.Debugf(ctx, "askPhoneNumberForReceiptCommand.Action()")

				input := whc.Input()

				var (
					mt             string
					phoneNumberStr string
					phoneNumber    int64
				)

				contact, isContactMessage := input.(botinput.ContactMessage)

				if isContactMessage {
					if contact == nil {
						m = whc.NewMessageByCode(trans.MESSAGE_TEXT_INVALID_PHONE_NUMBER)
						return nil
					}
					if params.User.Data.Names.FirstName == contact.GetFirstName() && params.User.Data.Names.LastName == contact.GetLastName() {
						phoneNumberStr = cleanPhoneNumber(contact.GetPhoneNumber())
						if phoneNumber, err = strconv.ParseInt(phoneNumberStr, 10, 64); err != nil {
							logus.Warningf(ctx, "Failed to parse contact's phone number: [%v]", phoneNumberStr)
							return err
						} else if len(params.User.Data.Phones) == 0 {
							phoneKey := strconv.FormatInt(phoneNumber, 10)
							params.User.Data.Phones = make(map[string]*with.CommunicationChannelProps)
							params.User.Data.Phones[phoneKey] = &with.CommunicationChannelProps{
								Verified: true,
							}
							params.User.Record.MarkAsChanged()
							params.UserUpdates = append(params.UserUpdates,
								update.ByFieldPath([]string{with.PhonesFieldName, phoneKey}, params.User.Data.Phones))
						}
						m = whc.NewMessage(trans.MESSAGE_TEXT_YOU_CAN_SEND_RECEIPT_TO_YOURSELF_BY_SMS)
						return nil
					}
					mt = contact.GetPhoneNumber()
				} else {
					mt = whc.Input().(botinput.TextMessage).Text()
				}

				if twilioTestNumber, ok := common4debtus.TwilioTestNumbers[mt]; ok {
					logus.Debugf(ctx, "Using predefined test number [%v]: %v", mt, twilioTestNumber)
					phoneNumberStr = twilioTestNumber
				} else {
					phoneNumberStr = cleanPhoneNumber(mt)
				}

				if phoneNumber, err = strconv.ParseInt(phoneNumberStr, 10, 64); err != nil {
					m = whc.NewMessageByCode(trans.MESSAGE_TEXT_INVALID_PHONE_NUMBER)
					return nil
				}

				chatEntity := whc.ChatData()

				awaitingUrl, err := url.Parse(chatEntity.GetAwaitingReplyTo())
				if err != nil {
					return fmt.Errorf("failed to parse chat state as URL: %w", err)
				}

				transferID := awaitingUrl.Query().Get(WizardParamTransfer)
				if transferID == "" {
					return fmt.Errorf("empty transferID")
				}
				transfer, err := facade4debtus.Transfers.GetTransferByID(ctx, tx, transferID)
				if err != nil {
					return fmt.Errorf("failed to get transfer by ContactID: %w", err)
				}
				spaceID := params.User.Data.GetFamilySpaceID()
				counterparty, err := facade4debtus.GetDebtusSpaceContactByID(ctx, tx, spaceID, transfer.Data.Counterparty().ContactID)
				if err != nil {
					return err
				}
				phoneContact := dto4contactus.PhoneContact{PhoneNumber: phoneNumber, PhoneNumberConfirmed: false}

				m, err = sendReceiptBySms(whc, tx, spaceID, phoneContact, transfer, counterparty)
				return err
			})

	},
}

const SMS_STATUS_MESSAGE_ID_PARAM_NAME = "SmsStatusMessageId"

//const SMS_STATUS_MESSAGE_UPDATES_COUNT_PARAM_NAME = "SmsStatusUpdatesCount"

func sendReceiptBySms(whc botsfw.WebhookContext, tx dal.ReadwriteTransaction, spaceID coretypes.SpaceID, phoneContact dto4contactus.PhoneContact, transfer models4debtus.TransferEntry, counterparty models4debtus.DebtusSpaceContactEntry) (m botmsg.MessageFromBot, err error) {
	ctx := whc.Context()

	if transfer.Data == nil {
		if transfer, err = facade4debtus.Transfers.GetTransferByID(ctx, tx, transfer.ID); err != nil {
			return m, err
		}
	}

	whc.ChatData() //TODO: Workaround to make whc.GetAppUser() working
	appUser, err := whc.AppUserData()
	if err != nil {
		return m, err
	}
	user := appUser.(interface{ FullName() string })
	//if err != nil {
	//	return
	//}

	var (
		smsText string
		//inviteCode string
	)

	receiptData := models4debtus.NewReceiptEntity(whc.AppUserID(), transfer.ID, transfer.Data.Counterparty().UserID, whc.Locale().Code5, "sms", strconv.FormatInt(phoneContact.PhoneNumber, 10), general4debtus.CreatedOn{
		CreatedOnPlatform: whc.BotPlatform().ID(),
		CreatedOnID:       whc.GetBotCode(),
	})
	var receipt models4debtus.ReceiptEntry
	if receipt, err = dal4debtus.Default.Receipt.CreateReceipt(ctx, receiptData); err != nil {
		return m, err
	}

	receiptUrl := common4debtus.GetReceiptUrl(receipt.ID, common4debtus.GetWebsiteHost(receiptData.CreatedOnID))

	stdContact, err := dal4contactus.GetContactByID(ctx, tx, spaceID, counterparty.ID)
	if err != nil {
		return m, err
	}
	if stdContact.Data.GetUserID() == "" {
		//related := fmt.Sprintf("%v=%v", models.TransfersCollection, transferID)
		//inviteKey, invite, err := invites.CreatePersonalInvite(whc, whc.AppUserID(), invites.InviteBySms, strconv.FormatInt(phoneContact.PhoneNumber, 10), whc.BotPlatform().ContactID(), whc.GetBotCode(), related)
		//if err != nil {
		//	logus.Errorf(ctx, "Failed to create invite: %v", err)
		//	return m, err
		//}
		//inviteCode = inviteKey.StringID()
	} else {
		// TODO: personalize receipt URL via anybot.GetReceiptUrlForUser(...) once that helper is restored.
		return m, fmt.Errorf("%w: sending SMS receipt to a registered user (counterpartyUserID=%s)",
			errors.ErrUnsupported, stdContact.Data.GetUserID())
	}

	// You've got $10 from Jack
	// You've given $10 to Jack

	switch transfer.Data.Direction() {
	case models4debtus.TransferDirectionUser2Counterparty:
		smsText = fmt.Sprintf(whc.Translate(trans.SMS_RECEIPT_YOU_GOT), transfer.Data.GetAmount(), user.FullName())
	case models4debtus.TransferDirectionCounterparty2User:
		smsText = fmt.Sprintf(whc.Translate(trans.SMS_RECEIPT_YOU_GAVE), transfer.Data.GetAmount(), user.FullName())
	default:
		return m, errors.New("Unknown direction: " + string(transfer.Data.Direction()))
	}
	smsText += "\n\n" + whc.Translate(trans.SMS_CLICK_TO_CONFIRM_OR_DECLINE, receiptUrl)

	chatEntity := whc.ChatData()

	var (
		smsStatusMessageID    string
		smsStatusMessageIntID int

		//smsStatusMessageUpdatesCount int
	)

	var createSmsStatusMessage = func() error {
		var msgSmsStatus botmsg.MessageFromBot
		mt := whc.Translate(trans.MESSAGE_TEXT_SMS_QUEUING_FOR_SENDING, phoneContact.PhoneNumberAsString())
		//logus.Debugf(ctx, "whc.InputTypes(): %v, botsfw.WebhookInputCallbackQuery: %v, MessageID: %v", whc.InputTypes(), botsfw.WebhookInputCallbackQuery, whc.InputCallbackQuery().GetMessage().IntID())
		if whc.Input().InputType() == botinput.TypeCallbackQuery {
			//logus.Debugf(ctx, "editMessage.MessageID: %v", editMessage.MessageID)
			if msgSmsStatus, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
				return err
			}
		} else {
			msgSmsStatus = whc.NewMessage(mt)
		}
		smsStatusMsg, err := whc.Responder().SendMessage(ctx, msgSmsStatus, botsfw.BotAPISendMessageOverHTTPS)
		if err != nil {
			return err
		}
		smsStatusMessageID = smsStatusMsg.Message.GetMessageID()
		smsStatusMessageIntID, _ = strconv.Atoi(smsStatusMessageID)
		chatEntity.AddWizardParam(SMS_STATUS_MESSAGE_ID_PARAM_NAME, smsStatusMessageID)
		return nil
	}

	if err = createSmsStatusMessage(); err != nil {
		return m, err
	}
	//if smsStatusMessageID, err = strconv.Atoi(chatEntity.GetWizardParam(SMS_STATUS_MESSAGE_ID_PARAM_NAME)); err != nil {
	//	if err = createSmsStatusMessage(); err != nil {
	//		return m, err
	//	}
	//}
	//if smsStatusMessageUpdatesCount, err = strconv.Atoi(chatEntity.GetWizardParam(SMS_STATUS_MESSAGE_UPDATES_COUNT_PARAM_NAME)); err == nil {
	//	if smsStatusMessageUpdatesCount > 2 {
	//		if err = createSmsStatusMessage(); err != nil {
	//			return m, err
	//		}
	//		chatEntity.AddWizardParam(SMS_STATUS_MESSAGE_UPDATES_COUNT_PARAM_NAME, "1")
	//	} else {
	//		chatEntity.AddWizardParam(SMS_STATUS_MESSAGE_UPDATES_COUNT_PARAM_NAME, strconv.Itoa(smsStatusMessageUpdatesCount + 1))
	//	}
	//} else {
	//	chatEntity.AddWizardParam(SMS_STATUS_MESSAGE_UPDATES_COUNT_PARAM_NAME, "1")
	//}

	tgChatID, err := strconv.ParseInt(whc.MustBotChatID(), 10, 64)

	if err != nil {
		return m, fmt.Errorf("failed to parse whc.BotChatID() to int: %w", err)
	}

	if lastTwilioSmsese, err := dal4debtus.Default.Twilio.GetLastTwilioSmsesForUser(ctx, tx, whc.AppUserID(), phoneContact.PhoneNumberAsString(), 1); err != nil {
		err = fmt.Errorf("failed to check latest SMS records: %w", err)
		return m, err
	} else if len(lastTwilioSmsese) > 0 {
		smsRecord := lastTwilioSmsese[0]
		if smsRecord.Data.To == phoneContact.PhoneNumberAsString() && (smsRecord.Data.Status == "delivered" || smsRecord.Data.Status == "queued") {
			// TODO: Do smarter check for limit
			m.Text = emoji.ERROR_ICON + " " + fmt.Sprintf("Exceeded limit for sending SMS to same number: %v", phoneContact.PhoneNumberAsString())
			logus.Warningf(ctx, m.Text)
			return m, err
		}
	}
	// TODO: Create SMS record before sending to ensure we don't spam user in case of bug after the API call.

	isTestSender, smsResponse, twilioException, err := sms.SendSms(whc.Context(), whc.GetBotSettings().Env == "prod", phoneContact.PhoneNumberAsString(), smsText)
	if err != nil {
		return m, fmt.Errorf("failed to send SMS: %w", err)
	}
	//sms := anybot.Sms{
	//	DtCreated: smsResponse.DateCreated,
	//	DtUpdate: smsResponse.DateUpdate,
	//	DtSent: smsResponse.DateSent,
	//	InviteCode: inviteCode,
	//	To: smsResponse.To,
	//	From: smsResponse.From,
	//	Status: smsResponse.Status,
	//}
	//if smsResponse.Price != nil {
	//	sms.Price = *smsResponse.Price
	//}

	if twilioException != nil {
		twilioExceptionStr, _ := json.Marshal(twilioException)
		logus.Errorf(ctx, "Failed to send SMS via Twilio: %v", string(twilioExceptionStr))
		mt, tryAnotherNumber := sms.TwilioExceptionToMessage(whc, whc, twilioException)
		if tryAnotherNumber {
			logus.Infof(ctx, "Twilio identified invalid phone number, need to try another one.")
			if m, err = whc.NewEditMessage(mt, botmsg.FormatText); err != nil {
				return
			}
			m.EditMessageUID = telegram.NewChatMessageUID(tgChatID, smsStatusMessageIntID)
			return
		}
		if counterparty.Data.PhoneNumber == phoneContact.PhoneNumber {
			var counterparty models4debtus.DebtusSpaceContactEntry
			counterparty, err = facade4debtus.GetDebtusSpaceContactByID(ctx, tx, spaceID, transfer.Data.Counterparty().ContactID)
			if err != nil {
				return
			}
			if counterparty.Data.PhoneNumber != phoneContact.PhoneNumber {
				counterparty.Data.PhoneNumber = phoneContact.PhoneNumber
				err = facade4debtus.SaveContact(ctx, counterparty)
			}
		}
		if m, err = whc.NewEditMessage(fmt.Sprintf("<b>Exception</b>\n%v\n\n<b>SMS text</b>\n%v", twilioException, smsText), botmsg.FormatHTML); err != nil {
			return
		}
		m.EditMessageUID = telegram.NewChatMessageUID(tgChatID, smsStatusMessageIntID)
		m.DisableWebPagePreview = true
		err = dtb_general.SetMainMenuKeyboard(whc, &m)
		return
	}

	smsResponseStr, _ := json.Marshal(smsResponse)
	logus.Debugf(ctx, "Twilio response: %v", string(smsResponseStr))

	analytics4bots.ReceiptSentFromBot(whc, "sms")

	if _, err = dal4debtus.Default.Twilio.SaveTwilioSms(
		whc.Context(),
		smsResponse,
		transfer,
		phoneContact,
		whc.AppUserID(),
		tgChatID,
		smsStatusMessageIntID,
	); err != nil {
		return
	}

	mt := whc.Translate(trans.MESSAGE_TEXT_SMS_QUEUED_FOR_SENDING, phoneContact.PhoneNumberAsString())

	if isTestSender {
		mt += "\n\n<b>SMS text</b>\n" + smsText
	}

	if m, err = whc.NewEditMessage(mt, botmsg.FormatHTML); err != nil {
		return
	}
	m.EditMessageUID = telegram.NewChatMessageUID(tgChatID, smsStatusMessageIntID)
	m.DisableWebPagePreview = true

	if _, err := whc.Responder().SendMessage(ctx, m, botsfw.BotAPISendMessageOverHTTPS); err != nil {
		err = fmt.Errorf("failed to send bot response message over HTTPS: %w", err)
		return m, err
	}

	return dtb_general.MainMenuCommand.Action(whc)
}
