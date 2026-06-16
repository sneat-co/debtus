package dtb_common

import (
	"context"
	"errors"
	"testing"

	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"go.uber.org/mock/gomock"
)

func TestGetUserWithNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("should panic")
		}
	}()
	if _, err := GetUser(nil); err != nil {
		t.Error("unexpected error", err)
	}
}

func TestGetUser_EmptyAppUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("")

	_, err := GetUser(whc)
	if !dal.IsNotFound(err) {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestGetUser_LoadsUserFromDB(t *testing.T) {
	ctx := context.Background()
	db := dalgo2memory.NewDB()

	user := dbo4userus.NewUserEntry("u1")
	user.Data.Email = "u1@example.com"
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, user.Record)
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("u1")
	whc.EXPECT().Context().Return(ctx)
	whc.EXPECT().DB().Return(db)

	got, err := GetUser(whc)
	if err != nil {
		t.Fatalf("GetUser() returned error: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("user.ID = %v, want u1", got.ID)
	}
	if got.Data.Email != "u1@example.com" {
		t.Errorf("user.Data.Email = %v, want u1@example.com", got.Data.Email)
	}
}

func TestGetUser_UserNotFound(t *testing.T) {
	ctx := context.Background()
	db := dalgo2memory.NewDB()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("missing")
	whc.EXPECT().Context().Return(ctx)
	whc.EXPECT().DB().Return(db)

	_, err := GetUser(whc)
	if err == nil || !errors.Is(err, dal.ErrRecordNotFound) && !dal.IsNotFound(err) {
		t.Errorf("expected not-found error, got: %v", err)
	}
}
