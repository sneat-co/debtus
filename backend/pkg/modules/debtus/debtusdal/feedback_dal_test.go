package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func TestFeedbackDalGae_GetFeedbackByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_feedback_when_exists", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewFeedback("42", &models4debtus.FeedbackData{}).Record)
		})
		if err != nil {
			t.Fatalf("failed to seed feedback: %v", err)
		}
		feedback, err := NewFeedbackDal().GetFeedbackByID(ctx, db, "42")
		if err != nil {
			t.Fatalf("GetFeedbackByID() returned error: %v", err)
		}
		if feedback.ID != "42" {
			t.Errorf("feedback.ID = %v, want 42", feedback.ID)
		}
	})

	t.Run("returns_not_found_for_missing_feedback", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		_, err := NewFeedbackDal().GetFeedbackByID(ctx, db, "99")
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})

	t.Run("uses_facade_db_when_tx_is_nil", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4debtus.NewFeedback("7", &models4debtus.FeedbackData{}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		// Pass nil tx — should fall back to facade.GetSneatDB
		feedback, err := NewFeedbackDal().GetFeedbackByID(ctx, nil, "7")
		if err != nil {
			t.Fatalf("GetFeedbackByID(nil tx) returned error: %v", err)
		}
		if feedback.ID != "7" {
			t.Errorf("feedback.ID = %v, want 7", feedback.ID)
		}
	})
}
