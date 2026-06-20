package dtb_transfer

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/sneat-co/debtus/backend/bots/botprofiles/debtusbot/dtb_common"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dal4reminders"
	"github.com/sneat-co/debtus/backend/debtus/reminders/dbo4reminders"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/i18n"
	"go.uber.org/mock/gomock"
)

// newReminderWhc wires a mock WebhookContext able to drive reminder/return flows
// whose helpers go through EditReminderMessage -> TextReceiptForTransfer.
func newReminderWhc(t *testing.T, ctrl *gomock.Controller, appUserID, env string) *mock_botsfw.MockWebhookContext {
	t.Helper()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()

	settings := &botsfw.BotSettings{Env: env}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return(appUserID).AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().GetBotSettings().Return(settings).AnyTimes()
	whc.EXPECT().Environment().Return("local").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}
	}).AnyTimes()
	// Text input → EditReminderMessage takes the NewMessage + SetMainMenuKeyboard branch.
	whc.EXPECT().Input().Return(fakeTextMsg{text: ""}).AnyTimes()
	return whc
}

// validTransfer builds a transfer where appUserID "u1" is the creator and both
// parties have user IDs, so TextReceiptForTransfer's autodetect resolves cleanly.
func validTransfer(id string, amount money.Amount) models4debtus.TransferEntry {
	return models4debtus.NewTransfer(id, models4debtus.NewTransferData(
		"u1",
		false,
		amount,
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1", ContactName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2", ContactName: "Bob"},
	))
}

// persistableTransfer builds a transfer whose From/To are stored as JSON so the
// entity survives a DB round-trip (TransferData.From() panics if FromJson is empty).
func persistableTransfer(id string, amount money.Amount) models4debtus.TransferEntry {
	td := &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      amount.Currency,
		AmountInCents: amount.Value,
		IsOutstanding: true,
		FromJson:      `{"userID":"u1","contactID":"c1","contactName":"Alice"}`,
		ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
	}
	td.DtCreated = time.Now()
	return models4debtus.NewTransfer(id, td)
}

// --- askWhenToRemindAgain (callback_reminder_return.go) ---

func TestAskWhenToRemindAgain_BuildsKeyboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newReminderWhc(t, ctrl, "u1", "prod")

	transfer := validTransfer("t1", money.NewAmount("USD", 1000))
	m, err := askWhenToRemindAgain(whc, "rem1", transfer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.IsEdit {
		t.Error("expected the reminder message to be an edit")
	}
	if m.Keyboard == nil {
		t.Fatal("expected a reschedule keyboard")
	}
}

func TestAskWhenToRemindAgain_DevAddsFewMinutesRow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whcProd := newReminderWhc(t, ctrl, "u1", "prod")
	whcDev := newReminderWhc(t, ctrl, "u1", "dev")

	transfer := validTransfer("t1", money.NewAmount("USD", 1000))

	mProd, err := askWhenToRemindAgain(whcProd, "rem1", transfer)
	if err != nil {
		t.Fatalf("prod unexpected error: %v", err)
	}
	mDev, err := askWhenToRemindAgain(whcDev, "rem1", transfer)
	if err != nil {
		t.Fatalf("dev unexpected error: %v", err)
	}

	prodRows := inlineRowCount(t, mProd)
	devRows := inlineRowCount(t, mDev)
	if devRows != prodRows+1 {
		t.Errorf("expected dev keyboard to have one extra row: prod=%d dev=%d", prodRows, devRows)
	}
}

// --- processNoReturn (callback_reminder_return.go) delegates to askWhenToRemindAgain ---

func TestProcessNoReturn_DelegatesToAskWhenToRemindAgain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newReminderWhc(t, ctrl, "u1", "prod")

	transfer := validTransfer("t1", money.NewAmount("USD", 1000))
	m, err := processNoReturn(whc, "rem1", transfer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.IsEdit || m.Keyboard == nil {
		t.Error("expected processNoReturn to produce a reschedule keyboard edit message")
	}
}

// --- ProcessFullReturn (callback_reminder_return.go): already-fully-returned branch ---

func TestProcessFullReturn_AlreadyFullyReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newReminderWhc(t, ctrl, "u1", "prod")

	// A return transfer that is fully returned → GetOutstandingValue == 0.
	td := models4debtus.NewTransferData(
		"u1",
		true, // IsReturn
		money.NewAmount("USD", 1000),
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", ContactID: "c1", ContactName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", ContactID: "c2", ContactName: "Bob"},
	)
	transfer := models4debtus.NewTransfer("t1", td)

	m, err := ProcessFullReturn(whc, transfer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected an already-fully-returned message")
	}
}

// --- ProcessReturnAnswer end-to-end (ReturnedNothing) ---

func TestProcessReturnAnswer_NothingReturnedReschedules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	transfer := persistableTransfer("t1", money.NewAmount("USD", 1000))
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusSent,
		TargetID: "t1",
	})
	seedDalRecords(t, db, transfer.Record, reminder.Record)

	whc := newReminderWhc(t, ctrl, "u1", "prod")

	u, _ := url.Parse(dtb_common.CallbackDebtReturnedPath + "?reminder=rem1&how-much=" + dtb_common.ReturnedNothing)
	m, err := ProcessReturnAnswer(whc, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.IsEdit || m.Keyboard == nil {
		t.Error("expected a reschedule keyboard edit message for nothing-returned answer")
	}

	// The reminder must have been marked as used.
	got, err := dal4reminders.GetReminderByID(context.Background(), db, "rem1")
	if err != nil {
		t.Fatalf("GetReminderByID: %v", err)
	}
	if got.Data.Status != dbo4reminders.ReminderStatusUsed {
		t.Errorf("reminder status = %q, want used", got.Data.Status)
	}
}

func TestProcessReturnAnswer_UnknownHowMuchReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	transfer := persistableTransfer("t1", money.NewAmount("USD", 1000))
	reminder := dbo4reminders.NewReminder("rem1", &dbo4reminders.ReminderDbo{
		Status:   dbo4reminders.ReminderStatusSent,
		TargetID: "t1",
	})
	seedDalRecords(t, db, transfer.Record, reminder.Record)

	whc := newReminderWhc(t, ctrl, "u1", "prod")

	u, _ := url.Parse(dtb_common.CallbackDebtReturnedPath + "?reminder=rem1&how-much=bogus")
	_, err := ProcessReturnAnswer(whc, u)
	if err == nil {
		t.Fatal("expected an error for an unknown how-much value")
	}
	if !strings.Contains(err.Error(), "unknown how-much") {
		t.Errorf("unexpected error: %v", err)
	}
}

// inlineRowCount returns the number of rows in a message's inline keyboard.
func inlineRowCount(t *testing.T, m botmsg.MessageFromBot) int {
	t.Helper()
	kb, ok := m.Keyboard.(*tgbotapi.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("expected *tgbotapi.InlineKeyboardMarkup, got %T", m.Keyboard)
	}
	return len(kb.InlineKeyboard)
}
