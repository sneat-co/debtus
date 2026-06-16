package dtb_transfer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bots-go-framework/bots-fw/mocks/mock_botsfw"
	"github.com/crediterra/money"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
	"go.uber.org/mock/gomock"
)

func TestProcessReturnCommand_NegativeReturnValueReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)

	_, err := processReturnCommand(whc, -100)
	if err == nil {
		t.Fatal("expected an error for negative returnValue, got nil")
	}
	if !strings.Contains(err.Error(), "returnValue < 0") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessPartialReturn_UnrelatedUserReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().AppUserID().Return("user-3").AnyTimes()

	transferData := models4debtus.NewTransferData("user-1", false, money.NewAmount("EUR", 100),
		&models4debtus.TransferCounterpartyInfo{UserID: "user-1"},
		&models4debtus.TransferCounterpartyInfo{UserID: "user-2", ContactID: "contact-2"},
	)
	transfer := models4debtus.NewTransfer("t1", transferData)

	_, err := ProcessPartialReturn(whc, transfer)
	if err == nil {
		t.Fatal("expected an error for a user unrelated to the transfer, got nil")
	}
	if !strings.Contains(err.Error(), "not in") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateTransferFromBot_ReturnToTransferIDWithoutIsReturn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	whc := mock_botsfw.NewMockWebhookContext(ctrl)
	whc.EXPECT().Context().Return(context.Background()).AnyTimes()

	_, err := CreateTransferFromBot(whc, false, "t1", models4debtus.TransferDirectionUser2Counterparty,
		models4debtus.TransferCounterpartyInfo{}, money.Amount{}, time.Time{}, models4debtus.NoInterest())
	if err == nil {
		t.Fatal("expected an error when returnToTransferID is set but isReturn is false, got nil")
	}
	if !strings.Contains(err.Error(), "isReturn") {
		t.Errorf("unexpected error message: %v", err)
	}
}
