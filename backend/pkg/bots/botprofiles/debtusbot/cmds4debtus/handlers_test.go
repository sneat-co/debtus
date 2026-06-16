package cmds4debtus

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botinput"
	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"go.uber.org/mock/gomock"
)

// fakeChosenInlineResult implements both botinput.InputMessage and botinput.ChosenInlineResult
// so it can be returned from whc.Input() and asserted to ChosenInlineResult.
type fakeChosenInlineResult struct {
	query           string
	inlineMessageID string
}

func (f fakeChosenInlineResult) GetQuery() string                 { return f.query }
func (f fakeChosenInlineResult) GetResultID() string              { return "" }
func (f fakeChosenInlineResult) GetInlineMessageID() string       { return f.inlineMessageID }
func (f fakeChosenInlineResult) GetSender() botinput.User         { return nil }
func (f fakeChosenInlineResult) GetRecipient() botinput.Recipient { return nil }
func (f fakeChosenInlineResult) GetTime() time.Time               { return time.Time{} }
func (f fakeChosenInlineResult) InputType() botinput.Type         { return botinput.TypeChosenInlineResult }
func (f fakeChosenInlineResult) MessageIntID() int                { return 0 }
func (f fakeChosenInlineResult) MessageStringID() string          { return "" }
func (f fakeChosenInlineResult) BotChatID() (string, error)       { return "", nil }
func (f fakeChosenInlineResult) Chat() botinput.Chat              { return nil }
func (f fakeChosenInlineResult) LogRequest()                      {}
func (f fakeChosenInlineResult) GetFrom() botinput.Sender         { return nil }

var _ botinput.ChosenInlineResult = fakeChosenInlineResult{}

func TestChosenResultQueryHandler_unknownPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeInput := fakeChosenInlineResult{query: "unknown?foo=bar", inlineMessageID: "msg1"}

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeInput).AnyTimes()

	handled, _, err := chosenResultQueryHandler(whc, fakeInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled for unknown path")
	}
}

func TestChosenResultQueryHandler_urlParseError(t *testing.T) {
	origUrlParse := urlParse
	urlParse = func(rawURL string) (*url.URL, error) {
		return nil, errors.New("parse error")
	}
	t.Cleanup(func() { urlParse = origUrlParse })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeInput := fakeChosenInlineResult{query: "bad url", inlineMessageID: "msg1"}
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeInput).AnyTimes()

	_, _, err := chosenResultQueryHandler(whc, fakeInput)
	if err == nil {
		t.Fatal("expected error from urlParse seam")
	}
}

func TestChosenResultQueryHandler_receiptPath(t *testing.T) {
	origFn := onInlineChosenCreateReceipt
	onInlineChosenCreateReceipt = func(_ botsfw.WebhookContext, _ string, _ *url.URL) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "receipt-ok"}}, nil
	}
	t.Cleanup(func() { onInlineChosenCreateReceipt = origFn })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeInput := fakeChosenInlineResult{query: "receipt", inlineMessageID: "msg42"}
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()
	whc.EXPECT().Input().Return(fakeInput).AnyTimes()

	handled, m, err := chosenResultQueryHandler(whc, fakeInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled=true for receipt path")
	}
	if m.Text != "receipt-ok" {
		t.Errorf("unexpected message text: %q", m.Text)
	}
}

// Ensure mock_botinput is used (it may be used indirectly in other tests)
var _ = mock_botinput.NewMockInlineQuery

func TestInlineQueryHandler_noMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	inlineQuery.EXPECT().GetQuery().Return("no match").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	handled, _, err := inlineQueryHandler(whc, inlineQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("expected not handled for unmatched query")
	}
}

func TestInlineQueryHandler_receiptPrefix(t *testing.T) {
	origFn := inlineSendReceipt
	inlineSendReceipt = func(_ botsfw.WebhookContext) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "receipt-sent"}}, nil
	}
	t.Cleanup(func() { inlineSendReceipt = origFn })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	inlineQuery.EXPECT().GetQuery().Return("receipt?id=abc123").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	handled, m, err := inlineQueryHandler(whc, inlineQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled=true for receipt?id= prefix")
	}
	if m.Text != "receipt-sent" {
		t.Errorf("unexpected text: %q", m.Text)
	}
}

func TestInlineQueryHandler_amountMatch(t *testing.T) {
	origFn := inlineNewRecord
	inlineNewRecord = func(_ botsfw.WebhookContext, _ []string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{TextMessageFromBot: botmsg.TextMessageFromBot{Text: "new-record"}}, nil
	}
	t.Cleanup(func() { inlineNewRecord = origFn })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	inlineQuery.EXPECT().GetQuery().Return("42.50 coffee").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	handled, m, err := inlineQueryHandler(whc, inlineQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected handled=true for amount-match query")
	}
	if m.Text != "new-record" {
		t.Errorf("unexpected text: %q", m.Text)
	}
}

func TestInlineQueryHandler_amountMatchError(t *testing.T) {
	origFn := inlineNewRecord
	inlineNewRecord = func(_ botsfw.WebhookContext, _ []string) (botmsg.MessageFromBot, error) {
		return botmsg.MessageFromBot{}, errors.New("record error")
	}
	t.Cleanup(func() { inlineNewRecord = origFn })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inlineQuery := mock_botinput.NewMockInlineQuery(ctrl)
	inlineQuery.EXPECT().GetQuery().Return("100 lunch").AnyTimes()

	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	_, _, err := inlineQueryHandler(whc, inlineQuery)
	if err == nil {
		t.Fatal("expected error from inlineNewRecord seam")
	}
}
