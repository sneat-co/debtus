package dtb_transfer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

// newReceiptWhc wires a mock WebhookContext for ShowReceipt: translator, linker
// source and a non-callback text input.
func newReceiptWhc(t *testing.T, ctrl *gomock.Controller, appUserID string) *mock_botsfw.MockWebhookContext {
	t.Helper()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return(appUserID).AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Environment().Return(strongoapp.LocalHostEnv).AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) botmsg.MessageFromBot {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}
	}).AnyTimes()
	whc.EXPECT().MustBotChatID().Return("chat1").AnyTimes()
	whc.EXPECT().Input().Return(fakeTextMsg{text: ""}).AnyTimes()
	return whc
}

// memoryReceiptDal is a minimal dal4debtus.ReceiptDal that resolves the global
// in-memory DB when called with a nil ReadSession, mirroring the fakeReceiptDal
// pattern in facade4debtus tests (the production ReceiptDal requires a non-nil
// tx, which ShowReceipt does not provide).
type memoryReceiptDal struct{}

func (memoryReceiptDal) GetReceiptByID(ctx context.Context, tx dal.ReadSession, id string) (models4debtus.ReceiptEntry, error) {
	receipt := models4debtus.NewReceipt(id, nil)
	if tx == nil {
		db, err := facade.GetSneatDB(ctx)
		if err != nil {
			return receipt, err
		}
		tx = db
	}
	return receipt, tx.Get(ctx, receipt.Record)
}

func (memoryReceiptDal) UpdateReceipt(ctx context.Context, tx dal.ReadwriteTransaction, receipt models4debtus.ReceiptEntry) error {
	return tx.Set(ctx, receipt.Record)
}

func (memoryReceiptDal) MarkReceiptAsSent(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (memoryReceiptDal) CreateReceipt(_ context.Context, _ *models4debtus.ReceiptDbo) (models4debtus.ReceiptEntry, error) {
	return models4debtus.ReceiptEntry{}, nil
}

func (memoryReceiptDal) DelayedMarkReceiptAsSent(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (memoryReceiptDal) DelayCreateAndSendReceiptToCounterpartyByTelegram(_ context.Context, _, _, _ string) error {
	return nil
}

func useMemoryReceiptDal(t *testing.T) {
	t.Helper()
	old := dal4debtus.Default.Receipt
	t.Cleanup(func() { dal4debtus.Default.Receipt = old })
	dal4debtus.Default.Receipt = memoryReceiptDal{}
}

// --- ShowReceipt (callback_receipt_view_pm.go) ---

func TestShowReceipt_AttemptToViewOwn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)
	useMemoryReceiptDal(t)

	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		Status:        models4debtus.ReceiptStatusSent,
		SpaceID:       dalTestSpaceID,
		TransferID:    "t1",
		CreatorUserID: "u1",
	})
	seedDalRecords(t, db, receipt.Record)

	whc := newReceiptWhc(t, ctrl, "u1") // viewer == creator
	m, err := ShowReceipt(whc, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "ATTEMPT_TO_VIEW_OWN") {
		t.Errorf("expected attempt-to-view-own message, got %q", m.Text)
	}
}

func TestShowReceipt_ViewByCounterpartyShowsAcknowledgeButtons(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	// Transfer created by u1; the creator has no contact ID so ShowReceipt loads
	// the creator user record to build the counterparty name.
	td := &models4debtus.TransferData{
		CreatorUserID: "u1",
		Currency:      money.CurrencyUSD,
		AmountInCents: 1000,
		BothUserIDs:   []string{"u1", "u2"},
		FromJson:      `{"userID":"u1","contactName":"Alice"}`,
		ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
	}
	useMemoryReceiptDal(t)
	td.DtCreated = time.Now().Add(-time.Hour)
	transfer := models4debtus.NewTransfer("t1", td)

	receipt := models4debtus.NewReceipt("r1", &models4debtus.ReceiptDbo{
		Status:             models4debtus.ReceiptStatusSent,
		SpaceID:            dalTestSpaceID,
		TransferID:         "t1",
		CreatorUserID:      "u1",
		CounterpartyUserID: "u2",
	})

	creator := dbo4userus.NewUserEntry("u1")
	creator.Data.Names = &person.NameFields{FirstName: "Alice"}
	seedDalRecords(t, db, transfer.Record, receipt.Record, creator.Record)

	whc := newReceiptWhc(t, ctrl, "u2") // viewer is the counterparty

	responder := mock_botsfw.NewMockWebhookResponder(ctrl)
	responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()
	whc.EXPECT().Responder().Return(responder).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(text string, _ any) (botmsg.MessageFromBot, error) {
			return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}, nil
		}).AnyTimes()

	m, err := ShowReceipt(whc, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After successfully sending to the counterparty, ShowReceipt edits the
	// message to the "sent / viewed by counterparty" confirmation.
	if !strings.Contains(m.Text, "RECEIPT_SENT_THROW_TELEGRAM") {
		t.Errorf("expected the receipt-sent confirmation, got %q", m.Text)
	}

	// The receipt must now be recorded as viewed by the counterparty.
	got, err := dal4debtus.Default.Receipt.GetReceiptByID(context.Background(), nil, "r1")
	if err != nil {
		t.Fatalf("GetReceiptByID: %v", err)
	}
	found := false
	for _, id := range got.Data.ViewedByUserIDs {
		if id == "u2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected receipt to be marked viewed by u2, got %v", got.Data.ViewedByUserIDs)
	}
}
