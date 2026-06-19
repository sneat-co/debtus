package dal4splitus

import (
	"context"
	"errors"
	"testing"

	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/debtus/backend/splitus/models4splitus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetSplitusSpace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	space := models4splitus.NewSplitusSpaceEntry("space1")

	t.Run("success", func(t *testing.T) {
		tx := mock_dal.NewMockReadTransaction(ctrl)
		tx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil)
		err := GetSplitusSpace(ctx, tx, space)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		tx := mock_dal.NewMockReadTransaction(ctrl)
		tx.EXPECT().Get(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
		err := GetSplitusSpace(ctx, tx, space)
		assert.Error(t, err)
	})
}

func TestSaveSplitusSpace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	space := models4splitus.NewSplitusSpaceEntry("space1")

	t.Run("success", func(t *testing.T) {
		tx := mock_dal.NewMockReadwriteTransaction(ctrl)
		tx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
		err := SaveSplitusSpace(ctx, tx, space)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		tx := mock_dal.NewMockReadwriteTransaction(ctrl)
		tx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
		err := SaveSplitusSpace(ctx, tx, space)
		assert.Error(t, err)
	})
}
