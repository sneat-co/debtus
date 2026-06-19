package admin

import (
	"context"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

func TestSendFeedbackToAdmins(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("should panic")
		}
	}()
	_ = SendFeedbackToAdmins(context.Background(), "", models4debtus.Feedback{})
}
