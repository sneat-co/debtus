package dtb_transfer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/cmds4debtus/dtb_general"
	"github.com/sneat-co/sneat-bots/pkg/bots/bothelper"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dto4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/facade4contactus"
	"github.com/sneat-co/sneat-core-modules/spaceus/dto4spaceus"
	"github.com/sneat-co/sneat-core-modules/userus/const4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-go-core/models/dbmodels"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/logus"
	"github.com/strongo/strongoapp/appuser"
	"github.com/strongo/strongoapp/person"
)

const newCounterpartyCommandCode = "new_counterparty"

func newCounterpartyCommand(nextCommand botsfw.Command) botsfw.Command {
	return botsfw.Command{
		Code:  newCounterpartyCommandCode,
		Title: trans.COMMAND_TEXT_NEW_COUNTERPARTY,
		/* We don't need to specify input type as it's invoked only on specific awaitingReplyTo
		InputTypes: []botinput.Type{
			botinput.TypeText,
			botinput.TypeContact,
		},
		*/
		Replies: []botsfw.Command{nextCommand},
		Action: func(whc botsfw.WebhookContext) (m botmsg.MessageFromBot, err error) {

			ctx := facade.NewContextWithUserID(whc.Context(), whc.AppUserID())

			chatEntity := whc.ChatData()
			var spaceRef coretypes.SpaceRef
			if spaceRef, err = bothelper.GetCurrentSpaceRef(whc); err != nil {
				return
			}
			spaceID := spaceRef.SpaceID()
			if chatEntity.IsAwaitingReplyTo(newCounterpartyCommandCode) {
				contactusSpace := dal4contactus.NewContactusSpaceEntry(spaceID)

				input := whc.Input()
				input.LogRequest()

				//var debtusContact models4debtus.DebtusSpaceContactEntry

				var (
					contactDetails        dto4contactus.ContactDetails
					existingContact       bool
					counterpartyContactID string
				)

				switch input := input.(type) {
				case botinput.TextMessage:
					mt := strings.TrimSpace(input.Text())
					if mt == "." {
						return dtb_general.MainMenuAction(whc, "", false)
					}
					if mt == "" {
						return m, errors.New("failed to get userContactJson details: mt is empty && inputMessage == nil")
					}
					if _, err = strconv.ParseFloat(mt, 64); err == nil {
						// User entered a number
						return whc.NewMessageByCode(trans.MESSAGE_TEXT_CONTACT_NAME_IS_NUMBER), nil
					}
					contactDetails = dto4contactus.ContactDetails{
						NameFields: person.NameFields{
							UserName: mt,
						},
					}
				case botinput.ContactMessage:
					if input == nil {
						return m, errors.New("failed to get WebhookContactMessage: contactMessage == nil")
					}

					contactDetails = dto4contactus.ContactDetails{
						NameFields: person.NameFields{
							FirstName: input.GetFirstName(),
							LastName:  input.GetLastName(),
						},
						//Username: username,
					}
					phoneStr := input.GetPhoneNumber()
					if phoneNum, err := strconv.ParseInt(phoneStr, 10, 64); err != nil {
						logus.Warningf(ctx, "Failed to parse phone string to int (%v)", phoneStr)
					} else {
						contactDetails.PhoneContact = dto4contactus.PhoneContact{
							PhoneNumber:          phoneNum,
							PhoneNumberConfirmed: true,
						}
					}

					contactBotUserID := input.GetBotUserID()
					if contactBotUserID != "" {
						contactDetails.TelegramUserID, err = strconv.ParseInt(input.GetBotUserID(), 10, 64) // TODO: check we are on Telegram
						if err != nil {
							err = fmt.Errorf("failed to parse contactBotUserID: %w", err)
							return
						}
					}
					var telegramUserID string
					if contactDetails.TelegramUserID != 0 {
						telegramUserID = strconv.FormatInt(contactDetails.TelegramUserID, 10)
					}

					if contactDetails.TelegramUserID != 0 {
						for contactID, contactBrief := range contactusSpace.Data.Contacts {
							var tgAccount *appuser.AccountKey
							if tgAccount, err = contactBrief.GetAccount(const4userus.TelegramAuthProvider, ""); err != nil {
								return
							}
							if tgAccount != nil && tgAccount.ID == telegramUserID {
								logus.Debugf(ctx, "Matched debtusContact my TelegramUserID=%d", contactDetails.TelegramUserID)
								existingContact = true
								counterpartyContactID = contactID
							}
						}
					}
				default:
					err = fmt.Errorf("unknown input, expected text or debtusContact message, got: %T", input)
					return
				}

				var db dal.DB
				if db, err = facade.GetSneatDB(ctx); err != nil {
					return
				}
				if !existingContact {
					if err = dal4contactus.GetContactusSpace(ctx, db, contactusSpace); err != nil {
						return
					}

					contactFullName := contactDetails.FullName()

					for _, userContact := range contactusSpace.Data.Contacts {
						if userContact.Names.FullName == contactFullName {
							m.Text = whc.Translate(trans.MESSAGE_TEXT_ALREADY_HAS_CONTACT_WITH_SUCH_NAME)
							return
						}
					}
				}

				if !existingContact {
					createContactRequest := dto4contactus.CreateContactRequest{
						Type:   briefs4contactus.ContactTypePerson,
						Status: dbmodels.StatusActive,
						SpaceRequest: dto4spaceus.SpaceRequest{
							SpaceID: spaceID,
						},
						Person: &dto4contactus.CreatePersonRequest{
							ContactBase: briefs4contactus.ContactBase{
								Status: dbmodels.StatusActive,
								ContactBrief: briefs4contactus.ContactBrief{
									Gender:   dbmodels.GenderUnknown,
									AgeGroup: dbmodels.AgeGroupUnknown,
									Type:     briefs4contactus.ContactTypePerson,
									Names:    &contactDetails.NameFields,
								},
							},
						},
					}
					//createContactRequest.Person.Phones = append(createContactRequest.Person.Phones)
					var counterpartyContact dal4contactus.ContactEntry

					if counterpartyContact, err = facade4contactus.CreateContact(ctx, false, createContactRequest); err != nil {
						err = fmt.Errorf("failed to create contact: %w", err)
						return
					}
					counterpartyContactID = counterpartyContact.ID
					//if _, contactusSpace, _, err = facade4debtus.CreateContact(ctx, tx, userID, spaceID, contactDetails); err != nil {
					//	return m, err
					//}
				}
				if counterpartyContactID == "" {
					return m, errors.New("debtusContact.ContactID is empty string")
				}
				chatEntity.AddWizardParam(WizardParamCounterparty, counterpartyContactID)
				return nextCommand.Action(whc)
				//m = whc.NewMessageByCode(fmt.Sprintf("DebtusSpaceContactEntry Created: %v", counterpartyKey))
			} else {
				m = whc.NewMessageByCode(trans.MESSAGE_TEXT_ASK_NEW_COUNTERPARTY_NAME)
				m.Format = botmsg.FormatHTML
				m.Keyboard = tgbotapi.NewHideKeyboard(true)
				chatEntity.PushStepToAwaitingReplyTo(newCounterpartyCommandCode)
			}
			return m, err
		},
	}
}
