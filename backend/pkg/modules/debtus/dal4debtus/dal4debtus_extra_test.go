package dal4debtus

import (
	"context"
	"errors"
	"testing"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/recordset"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

// minimalReadSession is a minimal dal.ReadSession that returns a canned record.
type minimalReadSession struct {
	getErr error
}

func (s *minimalReadSession) Exists(_ context.Context, _ *dal.Key) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *minimalReadSession) Get(_ context.Context, record dal.Record) error {
	if s.getErr != nil {
		return s.getErr
	}
	record.SetError(nil) // marks as loaded/existing
	return nil
}

func (s *minimalReadSession) GetMulti(_ context.Context, _ []dal.Record) error {
	return errors.New("not implemented")
}

func (s *minimalReadSession) ExecuteQueryToRecordsReader(_ context.Context, _ dal.Query) (dal.RecordsReader, error) {
	return nil, errors.New("not implemented")
}

func (s *minimalReadSession) ExecuteQueryToRecordsetReader(_ context.Context, _ dal.Query, _ ...recordset.Option) (dal.RecordsetReader, error) {
	return nil, errors.New("not implemented")
}

func TestGetDebtusUser_success(t *testing.T) {
	tx := &minimalReadSession{}
	user := models4debtus.NewDebtusUserEntry("u1")
	if err := GetDebtusUser(context.Background(), tx, user); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDebtusUser_error(t *testing.T) {
	tx := &minimalReadSession{getErr: errors.New("db error")}
	user := models4debtus.NewDebtusUserEntry("u1")
	if err := GetDebtusUser(context.Background(), tx, user); err == nil {
		t.Error("expected error from Get")
	}
}

func TestTransferSourceBot_PopulateTransfer_telegram(t *testing.T) {
	src := NewTransferSourceBot("telegram", "DebtusBot", "99999")
	data := models4debtus.NewTransferData(
		"u1",
		false,
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	)
	src.PopulateTransfer(data)

	if data.CreatedOnPlatform != "telegram" {
		t.Errorf("CreatedOnPlatform = %q, want telegram", data.CreatedOnPlatform)
	}
	if data.Creator().TgBotID != "DebtusBot" {
		t.Errorf("TgBotID = %q, want DebtusBot", data.Creator().TgBotID)
	}
	if data.Creator().TgChatID != 99999 {
		t.Errorf("TgChatID = %d, want 99999", data.Creator().TgChatID)
	}
}

func TestTransferSourceBot_PopulateTransfer_telegram_invalidChatID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on invalid chatID")
		}
	}()
	src := TransferSourceBot{platform: "telegram", botID: "DebtusBot", chatID: "notanumber"}
	data := models4debtus.NewTransferData(
		"u1",
		false,
		money.Amount{Currency: "USD", Value: 100},
		&models4debtus.TransferCounterpartyInfo{UserID: "u1", UserName: "Alice"},
		&models4debtus.TransferCounterpartyInfo{UserID: "u2", UserName: "Bob"},
	)
	src.PopulateTransfer(data)
}
