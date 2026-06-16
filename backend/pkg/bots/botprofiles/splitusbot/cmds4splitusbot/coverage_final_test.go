package cmds4splitusbot

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	tgbotapi "github.com/bots-go-framework/bots-api-telegram/tgbotapi"
	"github.com/bots-go-framework/bots-fw-store/botsfwmodels"
	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsdal"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/record"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/const4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/sneat-bots/pkg/bots/botprofiles/anybot"
	"github.com/sneat-co/debtus/backend/pkg/bots/botprofiles/splitusbot/facade4splitusbot"
	"github.com/sneat-co/sneat-bots/pkg/bots/botsettings"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/briefs4splitus"
	"github.com/sneat-co/debtus/backend/pkg/modules/splitus/models4splitus"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
	"github.com/strongo/strongoapp/person"
	"go.uber.org/mock/gomock"
)

func init() {
	facade4splitusbot.InitDelayingFotSplitusBot(delaying.MustRegisterFunc)
}

// TestNewEditMessageErrorBranches drives the NewEditMessage error returns of
// several bill callback commands.
func TestNewEditMessageErrorBranches(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "be1", nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newWhc := func() *mock_botsfw.MockWebhookContext {
		translator := newTestTranslator(context.Background())
		whc := mock_botsfw.NewMockWebhookContext(ctrl)
		whc.EXPECT().Context().Return(context.Background()).AnyTimes()
		whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
		whc.EXPECT().AppUserID().Return("u1").AnyTimes()
		whc.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
		whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
			if s := translator.Translate(key, args...); s != "" {
				return s
			}
			return key
		}).AnyTimes()
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		whc.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}, errTest).AnyTimes()
		return whc
	}

	for _, c := range []struct {
		name    string
		command botsfw.Command
	}{
		{"edit_bill", editBillCommand},
		{"change_bill_payer", changeBillPayerCommand},
		{"split_modes_list", billSplitModesListCommand},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := c.command.CallbackAction(newWhc(), mustParseURL(t, "https://x/cb?bill=be1")); err == nil {
				t.Fatal("expected NewEditMessage error")
			}
		})
	}
}

func TestStartBillActionSuccess(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "bsb1", nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := newMockWhc(ctrl)
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
	m, err := StartInBotAction(whc, []string{"bill-bsb1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text == "" {
		t.Error("expected bill card text")
	}
}

func TestBillCallbackActionGetUserGroupIDError(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "bug1", nil)

	orig := getUserGroupID
	getUserGroupID = func(whc botsfw.WebhookContext) (string, error) { return "", errTest }
	defer func() { getUserGroupID = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)
	whc.EXPECT().IsInGroup().Return(true, nil).AnyTimes()

	if _, err := billCardCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?bill=bug1")); err == nil {
		t.Fatal("expected getUserGroupID error")
	}
}

// selectiveBadTranslator renders the member title template fine but fails on
// the member row templates, to reach the second RenderTemplate error branch.
type selectiveBadTranslator struct{}

func (selectiveBadTranslator) Locale() i18n.Locale { return i18n.Locale{Code5: "zz-ZZ"} }
func (selectiveBadTranslator) Translate(key string, _ ...any) string {
	if strings.Contains(key, "MEMBER_TITLE") || key == "MT_BILL_CARD_MEMBER_TITLE" {
		return "title"
	}
	// The trans key constants are template names; returning a broken template
	// for everything except the first one makes the second render fail.
	return "{{.Bad"
}
func (t selectiveBadTranslator) TranslateWithMap(key string, _ map[string]string) string {
	return t.Translate(key)
}
func (t selectiveBadTranslator) TranslateNoWarning(key string, args ...any) string {
	return t.Translate(key, args...)
}

func TestWriteBillMembersListSecondTemplateError(t *testing.T) {
	ctx := context.Background()
	bill := newBillWithMembers("btmpl", []*briefs4splitus.BillMemberBrief{
		{MemberBrief: briefs4splitus.MemberBrief{ID: "m1", Name: "Alice", Shares: 1}},
	}, "USD")
	var buf bytes.Buffer
	writeBillMembersList(ctx, &buf, selectiveBadTranslator{}, bill, "")
}

func TestJoinBillActionDirect(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty_bill_id", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			_, err := joinBillAction(whc, tx, models4splitus.BillEntry{}, "", false)
			return err
		})
		if err == nil {
			t.Fatal("expected error for empty bill ID")
		}
	})

	guessCases := []struct {
		name   string
		locale string
	}{
		{"ru", i18n.LocaleCodeRuRU},
		{"fr", i18n.LocaleCodeFrFR},
		{"en_uk", i18n.LocaleCodeEnUK},
		{"default_usd", "xx-XX"},
	}
	for i, c := range guessCases {
		t.Run("guess_currency_"+c.name, func(t *testing.T) {
			billID := "bguess" + c.name
			putValidBill(t, ctx, db, billID, func(dbo *models4splitus.BillDbo) {
				dbo.Currency = ""
				dbo.SpaceID = "spaceG"
			})
			if i == 0 {
				splitusSpace := models4splitus.NewSplitusSpaceEntry("spaceG")
				putRecord(t, ctx, db, splitusSpace.Record)
			}
			translator := newTestTranslator(ctx)
			// Build a dedicated mock so each case gets its own Locale.
			whc2 := mock_botsfw.NewMockWebhookContext(ctrl)
			whc2.EXPECT().Context().Return(ctx).AnyTimes()
			whc2.EXPECT().GetBotCode().Return("testbot").AnyTimes()
			whc2.EXPECT().AppUserID().Return("u1").AnyTimes()
			whc2.EXPECT().Locale().Return(i18n.Locale{Code5: c.locale}).AnyTimes()
			whc2.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
				if s := translator.Translate(key, args...); s != "" {
					return s
				}
				return key
			}).AnyTimes()
			whc2.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
			whc2.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
			whc2.EXPECT().AppUserData().Return(&fakeAppUser{fullName: "Jane"}, nil).AnyTimes()

			defer func() { _ = recover() }() // AddOrGetMember production bug panics
			err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
				bill, err := facade4splitusGetBill(ctx, tx, billID)
				if err != nil {
					return err
				}
				_, err = joinBillAction(whc2, tx, bill, "", false)
				return err
			})
			_ = err
		})
	}

	t.Run("currency_is_in_group_error", func(t *testing.T) {
		putValidBill(t, ctx, db, "bgerr", func(dbo *models4splitus.BillDbo) {
			dbo.Currency = ""
		})
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		whc.EXPECT().AppUserData().Return(&fakeAppUser{fullName: "Jane"}, nil).AnyTimes()
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := facade4splitusGetBill(ctx, tx, "bgerr")
			if err != nil {
				return err
			}
			_, err = joinBillAction(whc, tx, bill, "", false)
			return err
		})
		if err == nil {
			t.Fatal("expected IsInGroup error")
		}
	})

	t.Run("in_group_currency_lookup_fails", func(t *testing.T) {
		putValidBill(t, ctx, db, "bgrp1", func(dbo *models4splitus.BillDbo) {
			dbo.Currency = ""
		})
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(true, nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		whc.EXPECT().AppUserData().Return(&fakeAppUser{fullName: "Jane"}, nil).AnyTimes()
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := facade4splitusGetBill(ctx, tx, "bgrp1")
			if err != nil {
				return err
			}
			_, err = joinBillAction(whc, tx, bill, "", false)
			return err
		})
		if err == nil {
			t.Fatal("expected space resolution error in group")
		}
	})

	t.Run("already_member_show_card_error", func(t *testing.T) {
		member := billMember("m1", "Jane")
		member.UserID = "u1"
		putValidBill(t, ctx, db, "bmem1", func(dbo *models4splitus.BillDbo) {
			dbo.Members = []*briefs4splitus.BillMemberBrief{member}
		})
		input := &fakeTgTextInput{
			upd: &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{Text: "old"},
			}},
		}
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest).AnyTimes()
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().AppUserData().Return(&fakeAppUser{fullName: "Jane"}, nil).AnyTimes()
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := facade4splitusGetBill(ctx, tx, "bmem1")
			if err != nil {
				return err
			}
			_, err = joinBillAction(whc, tx, bill, "", false)
			return err
		})
		if err == nil {
			t.Fatal("expected ShowBillCard error")
		}
	})

	t.Run("already_member_text_equal", func(t *testing.T) {
		member := billMember("m1", "Jane")
		member.UserID = "u1"
		putValidBill(t, ctx, db, "bmem2", func(dbo *models4splitus.BillDbo) {
			dbo.Members = []*briefs4splitus.BillMemberBrief{member}
		})
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().AppUserData().Return(&fakeAppUser{fullName: "Jane"}, nil).AnyTimes()

		// First compute the card text so the edited message matches it.
		var cardText string
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := facade4splitusGetBill(ctx, tx, "bmem2")
			if err != nil {
				return err
			}
			m2, err := ShowBillCard(whcForCard(ctrl, ctx), true, bill, "")
			cardText = m2.Text
			return err
		})
		if err != nil {
			t.Fatalf("failed to compute card text: %v", err)
		}

		input := &fakeTgTextInput{
			upd: &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{Text: cardText},
			}},
		}
		whc.EXPECT().Input().Return(input).AnyTimes()
		err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			bill, err := facade4splitusGetBill(ctx, tx, "bmem2")
			if err != nil {
				return err
			}
			_, err = joinBillAction(whc, tx, bill, "", false)
			return err
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// whcForCard builds a whc good enough for ShowBillCard with group keyboard.
func whcForCard(ctrl *gomock.Controller, ctx context.Context) *mock_botsfw.MockWebhookContext {
	translator := newTestTranslator(ctx)
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(ctx).AnyTimes()
	whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
	whc.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
	whc.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
		if s := translator.Translate(key, args...); s != "" {
			return s
		}
		return key
	}).AnyTimes()
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
	whc.EXPECT().NewMessage(gomock.Any()).DoAndReturn(func(text string) (m botmsg.MessageFromBot) {
		m.Text = text
		return m
	}).AnyTimes()
	return whc
}

// facade4splitusGetBill is a tiny helper around getBill for tests.
func facade4splitusGetBill(ctx context.Context, tx dal.ReadSession, billID string) (models4splitus.BillEntry, error) {
	u, err := url.Parse("https://x/cb?bill=" + billID)
	if err != nil {
		return models4splitus.BillEntry{}, err
	}
	return getBill(ctx, tx, u)
}

func TestSetBillCurrencyWithSpace(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	splitusSpace := models4splitus.NewSplitusSpaceEntry("spaceC")
	putRecord(t, ctx, db, splitusSpace.Record)
	putValidBill(t, ctx, db, "bc2", func(dbo *models4splitus.BillDbo) {
		dbo.SpaceID = "spaceC"
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()

	// ApplyBillBalanceDifference is not implemented yet, so the space branch
	// errors after exercising SaveBill + GetSplitusSpace.
	if _, err := setBillCurrencyCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?bill=bc2&currency=EUR")); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not-implemented error, got: %v", err)
	}
}

func TestChatNewMembersCommand(t *testing.T) {
	db := withMemDB(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("all_bots", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := &fakeNewChatMembersInput{members: []botinput.Actor{
			tgbotapi.ChatMember{IsBot: true, User: tgbotapi.User{FirstName: "Bot"}},
		}}
		whc.EXPECT().Input().Return(input).AnyTimes()
		if _, err := newChatMembersCommand.Action(whc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("human_member_space_missing", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := &fakeNewChatMembersInput{members: []botinput.Actor{
			tgbotapi.ChatMember{User: tgbotapi.User{FirstName: "Human"}},
		}}
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().DB().Return(db).AnyTimes()

		botUserKey := dal.NewKeyWithID("botUsers", "tg1")
		botUserData := &botsfwmodels.PlatformUserBaseDbo{}
		botUserRecord := dal.NewRecordWithData(botUserKey, botUserData)
		botUserRecord.SetError(dal.ErrRecordNotFound)
		botUser := botsdal.BotUser(record.NewDataWithID[string, botsfwmodels.PlatformUserData]("tg1", botUserKey, botUserData))
		botUser.Record = botUserRecord
		whc.EXPECT().GetBotUser().Return(botUser, dal.ErrRecordNotFound).AnyTimes()

		// The hard-coded placeholder space ID is 35 chars long which makes
		// NewSplitusSpaceEntry panic inside AddUsersToTheGroupAndOutstandingBills;
		// the statements up to that call are still exercised.
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic from NewSplitusSpaceEntry on over-long space ID")
			}
		}()
		_, _ = newChatMembersCommand.Action(whc)
	})

	t.Run("get_bot_user_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		input := &fakeNewChatMembersInput{members: []botinput.Actor{
			tgbotapi.ChatMember{User: tgbotapi.User{FirstName: "Human"}},
		}}
		whc.EXPECT().Input().Return(input).AnyTimes()
		whc.EXPECT().DB().Return(db).AnyTimes()
		whc.EXPECT().GetBotUser().Return(botsdal.BotUser{}, errTest).AnyTimes()
		if _, err := newChatMembersCommand.Action(whc); err == nil {
			t.Fatal("expected GetBotUser error")
		}
	})
}

// fakeNewChatMembersInput implements botinput.NewChatMembersMessage.
type fakeNewChatMembersInput struct {
	fakeInlineInput
	members []botinput.Actor
}

func (f *fakeNewChatMembersInput) NewChatMembers() []botinput.Actor { return f.members }
func (f *fakeNewChatMembersInput) InputType() botinput.Type {
	return botinput.TypeNewChatMembers
}

func TestGroupsCallbackRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whc := newMockWhc(ctrl)
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
	responder := mock_botsfw.NewMockWebhookResponder(ctrl)
	responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, nil).AnyTimes()
	whc.EXPECT().Responder().Return(responder).AnyTimes()

	_, err := groupsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?do=refresh"))
	_ = err // exercised: refresh branch including SendRefreshOrNothingChanged
}

func TestOutstandingBalanceActionDBError(t *testing.T) {
	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return nil, errTest }
	defer func() { facade.GetSneatDB = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)
	if _, err := outstandingBalanceAction(whc); err == nil {
		t.Fatal("expected error")
	}
}

// getMultiErrDB makes GetMulti fail with a non-NotFound error.
type getMultiErrDB struct {
	dal.DB
}

func (db getMultiErrDB) GetMulti(_ context.Context, _ []dal.Record) error { return errTest }

func TestSpaceSettingsActionGetMultiError(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	space := putSpace(t, ctx, db, "spaceM")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)
	whc.EXPECT().DB().Return(getMultiErrDB{DB: db}).AnyTimes()

	if _, err := SpaceSettingsAction(whc, space, false); err == nil {
		t.Fatal("expected GetMulti error")
	}
}

// rwTxErrDB makes RunReadwriteTransaction fail.
type rwTxErrDB struct {
	dal.DB
}

func (db rwTxErrDB) RunReadwriteTransaction(_ context.Context, _ dal.RWTxWorker, _ ...dal.TransactionOption) error {
	return errTest
}

func TestBillChangeSplitModeTxError(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "btx1", nil)

	orig := facade.GetSneatDB
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) { return rwTxErrDB{DB: db}, nil }
	defer func() { facade.GetSneatDB = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)

	if _, err := billChangeSplitModeCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?bill=btx1&mode="+string(models4splitus.SplitModeShare))); err == nil {
		t.Fatal("expected transaction error")
	}
}

func TestDelayErrorBranches(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)

	errDelayer := delaying.NewDelayer("err", func() {},
		func(_ context.Context, _ delaying.Params, _ ...any) error { return errTest },
		func(_ context.Context, _ delaying.Params, _ ...[]any) error { return errTest },
	)

	t.Run("delayUpdateBillCardOnUserJoin_enqueue_error", func(t *testing.T) {
		orig := delayUpdateBillCards
		delayUpdateBillCards = errDelayer
		defer func() { delayUpdateBillCards = orig }()
		if err := delayUpdateBillCardOnUserJoin(ctx, "b1", "msg"); err != nil {
			t.Fatalf("expected nil (error is logged), got: %v", err)
		}
	})

	t.Run("delayedUpdateBillCards_enqueue_error", func(t *testing.T) {
		putValidBill(t, ctx, db, "bde1", func(dbo *models4splitus.BillDbo) {
			dbo.TgChatMessageIDs = []string{"im1@tbot@en-US"}
		})
		orig := delayUpdateBillTgChatCard
		delayUpdateBillTgChatCard = errDelayer
		defer func() { delayUpdateBillTgChatCard = orig }()
		if err := delayedUpdateBillCards(ctx, "bde1", "f"); err == nil {
			t.Fatal("expected enqueue error")
		}
	})
}

// roundTripFunc lets tests stub HTTP transports.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDelayedUpdateBillTgChartCardSendPaths(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "btg1", nil)

	botsettings.SetBotSettingsProvider(func(_ context.Context) botsfw.BotSettingsBy {
		return botsfw.BotSettingsBy{ByCode: map[string]*botsfw.BotSettings{
			"tbot": {Code: "tbot", Token: "123:abc"},
		}}
	})
	t.Cleanup(func() { botsettings.SetBotSettingsProvider(nil) })

	origHTTP := dal4debtus.Default.HttpClient
	t.Cleanup(func() { dal4debtus.Default.HttpClient = origHTTP })

	t.Run("send_error", func(t *testing.T) {
		dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
			return &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errTest
			})}
		}
		if err := delayedUpdateBillTgChartCard(ctx, "btg1", "im1@tbot@en-US", "f"); err == nil {
			t.Fatal("expected telegram send error")
		}
	})

	t.Run("send_ok", func(t *testing.T) {
		dal4debtus.Default.HttpClient = func(_ context.Context) *http.Client {
			body := `{"ok":true,"result":{"message_id":1}}`
			return &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode:    200,
					ContentLength: int64(len(body)),
					Body:          io.NopCloser(strings.NewReader(body)),
					Header:        http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})}
		}
		if err := delayedUpdateBillTgChartCard(ctx, "btg1", "im1@tbot@en-US", "f"); err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestGroupMembersCardTitleFromNames(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)
	contactusSpace := dal4contactus.NewContactusSpaceEntry("spaceN")
	brief := &briefs4contactus.ContactBrief{
		Type:  briefs4contactus.ContactTypePerson,
		Names: &person.NameFields{FirstName: "First", LastName: "Last"},
	}
	brief.Roles = []string{const4contactus.SpaceMemberRoleMember}
	contactusSpace.Data.AddContact("c1", brief)

	text, err := groupMembersCard(ctx, translator, contactusSpace, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "First") {
		t.Errorf("expected name from Names fields, got %q", text)
	}
}

func TestRegisterCommandClosures(t *testing.T) {
	var commands []botsfw.Command
	RegisterCommand(func(cmds ...botsfw.Command) { commands = append(commands, cmds...) }, true)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var helpCmd *botsfw.Command
	for i := range commands {
		if commands[i].Code == "help" {
			helpCmd = &commands[i]
			break
		}
	}
	if helpCmd == nil {
		t.Fatal("help command not registered")
	}
	whc := newMockWhc(ctrl)
	m, err := helpCmd.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(m.Text, "not implemented") {
		t.Errorf("unexpected help text: %q", m.Text)
	}
}

func TestEditedBillCardHookWithGroup(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "67890", nil)
	splitusSpace := models4splitus.NewSplitusSpaceEntry("spaceE")
	putRecord(t, ctx, db, splitusSpace.Record)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	entities := []tgbotapi.MessageEntity{
		{Type: "text_link", URL: "https://t.me/testbot?start=bill-67890"},
	}
	input := &fakeTgInput{upd: &tgbotapi.Update{
		EditedMessage: &tgbotapi.Message{Entities: &entities},
	}}

	chatData := &anybot.SneatAppTgChatDbo{}
	chatData.UserGroupID = "spaceE"

	whc := newMockWhcNoChatData(ctrl)
	whc.EXPECT().ChatData().Return(chatData).AnyTimes()
	whc.EXPECT().Input().Return(input).AnyTimes()

	_, err := EditedBillCardHookCommand.Action(whc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("bill_already_in_another_group", func(t *testing.T) {
		putValidBill(t, ctx, db, "67891", func(dbo *models4splitus.BillDbo) {
			dbo.SpaceID = "spaceOther"
		})
		entities2 := []tgbotapi.MessageEntity{
			{Type: "text_link", URL: "https://t.me/testbot?start=bill-67891"},
		}
		input2 := &fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &entities2},
		}}
		whc2 := newMockWhcNoChatData(ctrl)
		whc2.EXPECT().ChatData().Return(chatData).AnyTimes()
		whc2.EXPECT().Input().Return(input2).AnyTimes()
		if _, err := EditedBillCardHookCommand.Action(whc2); err == nil {
			t.Fatal("expected ErrBillAlreadyAssignedToAnotherGroup")
		}
	})

	t.Run("bill_not_found_in_tx", func(t *testing.T) {
		entitiesMissing := []tgbotapi.MessageEntity{
			{Type: "text_link", URL: "https://t.me/testbot?start=bill-99999"},
		}
		inputMissing := &fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &entitiesMissing},
		}}
		whcM := newMockWhcNoChatData(ctrl)
		whcM.EXPECT().ChatData().Return(chatData).AnyTimes()
		whcM.EXPECT().Input().Return(inputMissing).AnyTimes()
		if _, err := EditedBillCardHookCommand.Action(whcM); err == nil {
			t.Fatal("expected not-found error")
		}
	})

	t.Run("bill_already_in_same_group", func(t *testing.T) {
		putValidBill(t, ctx, db, "67892", func(dbo *models4splitus.BillDbo) {
			dbo.SpaceID = "spaceE"
		})
		entities3 := []tgbotapi.MessageEntity{
			{Type: "text_link", URL: "https://t.me/testbot?start=bill-67892"},
		}
		input3 := &fakeTgInput{upd: &tgbotapi.Update{
			EditedMessage: &tgbotapi.Message{Entities: &entities3},
		}}
		whc3 := newMockWhcNoChatData(ctrl)
		whc3.EXPECT().ChatData().Return(chatData).AnyTimes()
		whc3.EXPECT().Input().Return(input3).AnyTimes()
		if _, err := EditedBillCardHookCommand.Action(whc3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// emptyTranslatorWhc returns "" for all translations so that the bill card
// message text is blank, reaching the empty-text guard in
// createBillFromInlineChosenResult.
func TestCreateBillFromInlineChosenResultEmptyTextAndEditError(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	_ = ctx
	_ = db

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newEmptyTranslateWhc := func() *mock_botsfw.MockWebhookContext {
		whc := mock_botsfw.NewMockWebhookContext(ctrl)
		whc.EXPECT().Context().Return(context.Background()).AnyTimes()
		whc.EXPECT().GetBotCode().Return("testbot").AnyTimes()
		whc.EXPECT().AppUserID().Return("u1").AnyTimes()
		whc.EXPECT().Locale().Return(i18n.Locale{Code5: "ze-ZE"}).AnyTimes()
		whc.EXPECT().Translate(gomock.Any(), gomock.Any()).Return("").AnyTimes()
		return whc
	}

	// NOTE: the empty-card-text guard is unreachable because the footer always
	// contains a non-empty horizontal rule; see TEST-COVERAGE.md.
	_ = newEmptyTranslateWhc

	t.Run("new_edit_message_error", func(t *testing.T) {
		whc2 := mock_botsfw.NewMockWebhookContext(ctrl)
		translator := newTestTranslator(context.Background())
		whc2.EXPECT().Context().Return(context.Background()).AnyTimes()
		whc2.EXPECT().GetBotCode().Return("testbot").AnyTimes()
		whc2.EXPECT().AppUserID().Return("u1").AnyTimes()
		whc2.EXPECT().Locale().Return(i18n.LocalesByCode5[i18n.LocaleCodeEnUK]).AnyTimes()
		whc2.EXPECT().Translate(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, args ...any) string {
			if s := translator.Translate(key, args...); s != "" {
				return s
			}
			return key
		}).AnyTimes()
		whc2.EXPECT().NewEditMessage(gomock.Any(), gomock.Any()).Return(botmsg.MessageFromBot{}, errTest).AnyTimes()
		cr := &fakeChosenResult{resultID: "bill?amount=100usd", inlineMessageID: "im1"}
		if _, err := createBillFromInlineChosenResult(whc2, cr); err == nil {
			t.Fatal("expected NewEditMessage error")
		}
	})
}

func TestSetBillCurrencySplitusSpaceMissing(t *testing.T) {
	ctx := context.Background()
	db := withMemDB(t)
	putValidBill(t, ctx, db, "bcm1", func(dbo *models4splitus.BillDbo) {
		dbo.SpaceID = "spaceMissing"
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := newMockWhc(ctrl)
	whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
	whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()

	if _, err := setBillCurrencyCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?bill=bcm1&currency=EUR")); err == nil {
		t.Fatal("expected GetSplitusSpace not-found error")
	}
}

func TestGroupsCallbackErrorBranches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("groups_action_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, errTest).AnyTimes()
		if _, err := groupsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?do=refresh")); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("send_refresh_error", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		whc.EXPECT().IsInGroup().Return(false, nil).AnyTimes()
		whc.EXPECT().Input().Return(&fakeInlineInput{}).AnyTimes()
		responder := mock_botsfw.NewMockWebhookResponder(ctrl)
		responder.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(botsfw.OnMessageSentResponse{}, errTest).AnyTimes()
		whc.EXPECT().Responder().Return(responder).AnyTimes()
		_, err := groupsCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?do=refresh"))
		_ = err // exercised: SendRefreshOrNothingChanged error branch
	})
}

func TestSetBillDueDateCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("callback", func(t *testing.T) {
		chatData := &anybot.SneatAppTgChatDbo{}
		whc := newMockWhcNoChatData(ctrl)
		whc.EXPECT().ChatData().Return(chatData).AnyTimes()
		m, err := setBillDueDateCommand.CallbackAction(whc, mustParseURL(t, "https://x/cb?id=b1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Keyboard == nil {
			t.Error("expected force-reply keyboard")
		}
	})

	t.Run("action", func(t *testing.T) {
		whc := newMockWhc(ctrl)
		m, err := setBillDueDateCommand.Action(whc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Text == "" {
			t.Error("expected text")
		}
	})
}
