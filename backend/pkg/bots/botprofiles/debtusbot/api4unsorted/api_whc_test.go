package api4unsorted

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
)

func TestApiWebhookContext_UnsupportedMethodsReturnErrors(t *testing.T) {
	whc := ApiWebhookContext{}

	for name, call := range map[string]func() error{
		"IsInGroup": func() error {
			_, err := whc.IsInGroup()
			return err
		},
		"AppUserData": func() error {
			_, err := whc.AppUserData()
			return err
		},
		"GetAppUser": func() error {
			_, err := whc.GetAppUser()
			return err
		},
		"NewEditMessage": func() error {
			_, err := whc.NewEditMessage("some text", botmsg.FormatHTML)
			return err
		},
		"UpdateLastProcessed": func() error {
			return whc.UpdateLastProcessed(nil)
		},
		"Responder.SendMessage": func() error {
			_, err := whc.Responder().SendMessage(context.Background(), botmsg.MessageFromBot{}, botsfw.BotAPISendMessageOverHTTPS)
			return err
		},
		"Responder.DeleteMessage": func() error {
			return whc.Responder().DeleteMessage(context.Background(), "msg1")
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, errNotSupportedByAPIWebhookContext) {
				t.Errorf("expected error wrapping errNotSupportedByAPIWebhookContext, got: %v", err)
			}
		})
	}
}

func TestHandleGetUserData_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/user/data/load-Contacts", nil)
	HandleGetUserData(r.Context(), w, r, token4auth.AuthInfo{})
	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
}

func TestCreateInvite_NotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api4debtus/invite/create", nil)
	CreateInvite(r.Context(), w, r)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
}
