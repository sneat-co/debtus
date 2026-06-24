package dtb_transfer

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/contactus-ext/backend/contactusmodels/briefs4contactus"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/debtusdal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

const dalTestSpaceID coretypes.SpaceID = "s1"

// TestMain wires test-only seams for the whole package:
//   - a non-panicking token4auth.IssueBotToken stub (token4auth is a package-level
//     var) so any code path that builds a signed user/transfer/contact URL does
//     not panic during tests;
//   - dal4debtus.Default populated with the real dalgo-backed DAL (built
//     atomically by debtusdal.NewDAL, no global side effects) so DAL-backed
//     handlers resolve their data layer instead of dereferencing a nil locator.
func TestMain(m *testing.M) {
	origToken := token4auth.IssueBotToken
	token4auth.IssueBotToken = func(_ context.Context, _, _, _ string) (string, error) {
		return "test-token", nil
	}
	origDAL := dal4debtus.Default
	dal4debtus.Default = debtusdal.NewDAL()

	code := m.Run()

	token4auth.IssueBotToken = origToken
	dal4debtus.Default = origDAL
	os.Exit(code)
}

// seedDalRecords stores records into the in-memory DB via a real transaction.
func seedDalRecords(t *testing.T, db dal.DB, records ...dal.Record) {
	t.Helper()
	err := db.RunReadwriteTransaction(context.Background(), func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.SetMulti(ctx, records)
	})
	if err != nil {
		t.Fatalf("failed to seed records: %v", err)
	}
}

// newBalanceWhc wires a mock WebhookContext able to drive balanceAction:
// it acts as a translator, linker source and text-input context.
func newBalanceWhc(t *testing.T, ctrl *gomock.Controller, appUserID string) *mock_botsfw.MockWebhookContext {
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
	// balanceAction reads whc.Input().InputType(); a plain text message → not a callback.
	whc.EXPECT().Input().Return(fakeTextMsg{text: ""}).AnyTimes()
	return whc
}

// --- balanceAction (transfer_balance_cmd.go) ---

func TestBalanceAction_ZeroBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	debtusSpace := models4debtus.NewDebtusSpaceEntry(dalTestSpaceID)
	contactusSpace := dal4contactus.NewContactusSpaceEntry(dalTestSpaceID)
	seedDalRecords(t, db, debtusSpace.Record, contactusSpace.Record)

	whc := newBalanceWhc(t, ctrl, "u1")
	spaceRef := coretypes.NewSpaceRef(coretypes.SpaceTypeFamily, dalTestSpaceID)

	m, err := balanceAction(whc, spaceRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "BALANCE_IS_ZERO") {
		t.Errorf("expected balance-is-zero message, got %q", m.Text)
	}
}

func TestBalanceAction_NonZeroBalanceWithContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	debtusSpace := models4debtus.NewDebtusSpaceEntry(dalTestSpaceID)
	debtusSpace.Data.Balance = money.Balance{money.CurrencyUSD: 1000}
	debtusSpace.Data.Contacts = map[string]*models4debtus.DebtusContactBrief{
		"c1": {Status: models4debtus.DebtusContactStatusActive, Balance: money.Balance{money.CurrencyUSD: 1000}},
	}

	contactusSpace := dal4contactus.NewContactusSpaceEntry(dalTestSpaceID)
	contactusSpace.Data.Contacts = map[string]*briefs4contactus.ContactBrief{
		"c1": {Names: &person.NameFields{FirstName: "John"}},
	}
	seedDalRecords(t, db, debtusSpace.Record, contactusSpace.Record)

	whc := newBalanceWhc(t, ctrl, "u1")
	spaceRef := coretypes.NewSpaceRef(coretypes.SpaceTypeFamily, dalTestSpaceID)

	m, err := balanceAction(whc, spaceRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "John") {
		t.Errorf("expected contact name in balance output, got %q", m.Text)
	}
	if !strings.Contains(m.Text, "BALANCE_HEADER") {
		t.Errorf("expected balance header, got %q", m.Text)
	}
}

func TestBalanceAction_NonZeroBalanceNoContactsIsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	debtusSpace := models4debtus.NewDebtusSpaceEntry(dalTestSpaceID)
	debtusSpace.Data.Balance = money.Balance{money.CurrencyUSD: 1000}
	// No contacts → integrity error branch.
	contactusSpace := dal4contactus.NewContactusSpaceEntry(dalTestSpaceID)
	seedDalRecords(t, db, debtusSpace.Record, contactusSpace.Record)

	whc := newBalanceWhc(t, ctrl, "u1")
	spaceRef := coretypes.NewSpaceRef(coretypes.SpaceTypeFamily, dalTestSpaceID)

	_, err := balanceAction(whc, spaceRef)
	if err == nil {
		t.Fatal("expected an integrity error when balance is non-zero but there are no contacts")
	}
	if !strings.Contains(err.Error(), "integrity issue") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- dueReturnsCallbackAction (due_returns_cmd.go) ---

func newDueReturnsWhc(t *testing.T, ctrl *gomock.Controller) *mock_botsfw.MockWebhookContext {
	t.Helper()
	platform := mock_botsfw.NewMockBotPlatform(ctrl)
	platform.EXPECT().ID().Return("telegram").AnyTimes()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("u1").AnyTimes()
	whc.EXPECT().BotPlatform().Return(platform).AnyTimes()
	whc.EXPECT().GetBotCode().Return("debtusbot").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocaleEnUS).AnyTimes()
	whc.EXPECT().Translate(gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, _ ...any) string { return key }).AnyTimes()
	whc.EXPECT().CommandText(gomock.Any(), gomock.Any()).DoAndReturn(func(title, icon string) string { return icon + " " + title }).AnyTimes()
	whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(text string, _ any) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: text}}, nil
	}).AnyTimes()
	return whc
}

func TestDueReturnsCallbackAction_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sneattesting.SetupMemoryDB(t)

	whc := newDueReturnsWhc(t, ctrl)
	m, err := dueReturnsCallbackAction(whc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "DUE_RETURNS_EMPTY") {
		t.Errorf("expected due-returns-empty message, got %q", m.Text)
	}
	if m.Keyboard == nil {
		t.Error("expected a balance keyboard")
	}
}

func TestDueReturnsCallbackAction_ListsOverdueAndDue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	mkDueTransfer := func(id string, dueOn time.Time) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      "USD",
			AmountInCents: 1000,
			IsOutstanding: true,
			BothUserIDs:   []string{"u1", "u2"},
			DtDueOn:       dueOn,
			FromJson:      `{"userID":"u1","contactID":"c1","contactName":"Alice"}`,
			ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
		}
		td.DtCreated = time.Now().Add(-time.Hour)
		return models4debtus.NewTransfer(id, td)
	}
	overdue := mkDueTransfer("tOverdue", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	due := mkDueTransfer("tDue", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	seedDalRecords(t, db, overdue.Record, due.Record)

	whc := newDueReturnsWhc(t, ctrl)
	m, err := dueReturnsCallbackAction(whc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "OVERDUE_RETURNS_HEADER") {
		t.Errorf("expected overdue header in output, got %q", m.Text)
	}
	if !strings.Contains(m.Text, "DUE_RETURNS_HEADER") {
		t.Errorf("expected due header in output, got %q", m.Text)
	}
}

// --- showHistoryCard (transfer_history.go) with seeded transfers ---

func TestShowHistoryCard_ListsTransfersWithHasMore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	mkTransfer := func(id string, created time.Time) models4debtus.TransferEntry {
		td := &models4debtus.TransferData{
			CreatorUserID: "u1",
			Currency:      "USD",
			AmountInCents: 1000,
			BothUserIDs:   []string{"u1", "u2"},
			FromJson:      `{"userID":"u1","contactID":"c1","contactName":"Alice"}`,
			ToJson:        `{"userID":"u2","contactID":"c2","contactName":"Bob"}`,
		}
		td.DtCreated = created
		return models4debtus.NewTransfer(id, td)
	}
	t1 := mkTransfer("t1", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	t2 := mkTransfer("t2", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	seedDalRecords(t, db, t1.Record, t2.Record)

	whc := newBalanceWhc(t, ctrl, "u1")
	// showHistoryCard builds an UTM "show full history" link via whc.MustBotChatID.
	whc.EXPECT().MustBotChatID().Return("chat1").AnyTimes()

	// limit=1 with 2 seeded transfers → hasMore=true → "show full history" row.
	spaceRef := coretypes.NewSpaceRef(coretypes.SpaceTypeFamily, dalTestSpaceID)
	m, err := showHistoryCard(whc, spaceRef, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "HISTORY_LIST") {
		t.Errorf("expected the history-list message, got %q", m.Text)
	}
	if m.Keyboard == nil {
		t.Fatal("expected a keyboard with the show-full-history and back buttons")
	}
}

// --- balanceTextAction (transfer_balance_cmd.go) ---

func TestBalanceTextAction_EmptyAppUserIDReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().AppUserID().Return("").AnyTimes()

	_, err := balanceTextAction(whc, "")
	if err == nil {
		t.Fatal("expected an error when AppUserID is empty")
	}
	if !strings.Contains(err.Error(), "AppUserID is empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBalanceTextAction_LoadsUserAndShowsZeroBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := sneattesting.SetupMemoryDB(t)

	user := dbo4userus.NewUserEntry("u1")
	user.Data.Names = &person.NameFields{FirstName: "Alice"}
	familyBrief := &dbo4userus.UserSpaceBrief{}
	familyBrief.Type = coretypes.SpaceTypeFamily
	user.Data.Spaces = map[string]*dbo4userus.UserSpaceBrief{
		string(dalTestSpaceID): familyBrief,
	}

	debtusSpace := models4debtus.NewDebtusSpaceEntry(dalTestSpaceID)
	contactusSpace := dal4contactus.NewContactusSpaceEntry(dalTestSpaceID)
	seedDalRecords(t, db, user.Record, debtusSpace.Record, contactusSpace.Record)

	whc := newBalanceWhc(t, ctrl, "u1")

	m, err := balanceTextAction(whc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "BALANCE_IS_ZERO") {
		t.Errorf("expected balance-is-zero message, got %q", m.Text)
	}
}
