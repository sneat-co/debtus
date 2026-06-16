package reminders

import (
	"context"
	"fmt"
	"time"

	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dal4reminders"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/reminders/delay4reminders"
	"github.com/strongo/i18n"
	"github.com/strongo/logus"
)

func sendReminderByEmail(ctx context.Context, reminder dbo4reminders.Reminder, emailTo string, transfer models4debtus.TransferEntry, user dbo4userus.UserEntry) (err error) {
	logus.Debugf(ctx, "sendReminderByEmail(reminder.ContactID=%v, emailTo=%v)", reminder.ID, emailTo)

	emailMessage := emails.Email{
		From: common4all.FROM_REMINDER,
		To: []string{
			emailTo, // Required
		},
		Subject: "Due payment notification",
		Text:    fmt.Sprintf("Hi %v, you have a due payment to %v: %v%v.", transfer.Data.Counterparty().ContactName, user.Data.Names.UserName, transfer.Data.AmountInCents, transfer.Data.Currency),
	}

	var emailClient emails.Client

	if emailClient, err = emailing.GetEmailClient(ctx); err != nil {
		return
	}

	var sent emails.Sent
	sent, err = emailClient.Send(ctx, emailMessage)

	sentAt := time.Now()

	var errDetails string
	if err != nil {
		errDetails = err.Error()
	}
	var emailMessageID string
	if sent != nil {
		emailMessageID = sent.MessageID()
	}

	if err = dal4reminders.SetReminderIsSent(ctx, reminder.ID, sentAt, 0, emailMessageID, i18n.LocaleCodeEnUS, errDetails); err != nil {
		if err = delay4reminders.DelaySetReminderIsSent(ctx, reminder.ID, sentAt, 0, emailMessageID, i18n.LocaleCodeEnUS, errDetails); err != nil {
			return fmt.Errorf("failed to delay setting reminder as sent: %w", err)
		}
	}

	if err != nil {
		// Print the error, cast err to awserr.Error to get the ByCode and
		// Message from an error.
		return fmt.Errorf("failed to send email using AWS SES: %w", err)
	}

	// Pretty-print the response data.
	logus.Debugf(ctx, "AWS SES output (for Reminder=%v): %v", reminder.ID, sent)
	return nil
}
