package facade4debtus

import (
	"context"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
)

func TestSaveFeedback(t *testing.T) {
	ctx := context.Background()

	t.Run("inserts_feedback_and_updates_user", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		user := dbo4userus.NewUserEntry("u1")
		user.Data.Email = "u1@example.com"
		seedRecords(t, db, user.Record)

		var feedback models4debtus.Feedback
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			feedback, _, err = SaveFeedback(ctx, tx, "", &models4debtus.FeedbackData{
				UserStrID: "u1",
				Rate:      "5",
			})
			return err
		})
		if err != nil {
			t.Fatalf("SaveFeedback() returned error: %v", err)
		}
		if feedback.Created.IsZero() {
			t.Error("feedback.Created should be set")
		}

		savedUser := dbo4userus.NewUserEntry("u1")
		if err = db.Get(ctx, savedUser.Record); err != nil {
			t.Fatalf("failed to read user: %v", err)
		}
		if savedUser.Data.LastFeedbackRate != "5" {
			t.Errorf("user.LastFeedbackRate = %q, want 5", savedUser.Data.LastFeedbackRate)
		}
		if savedUser.Data.LastFeedbackAt.IsZero() {
			t.Error("user.LastFeedbackAt should be set")
		}
	})

	t.Run("with_feedback_id_and_preset_created_uses_setmulti", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		user := dbo4userus.NewUserEntry("u1")
		seedRecords(t, db, user.Record)

		presetCreated := time.Now().Add(-time.Hour)
		var feedback models4debtus.Feedback
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			feedback, _, err = SaveFeedback(ctx, tx, "f1", &models4debtus.FeedbackData{
				UserStrID: "u1",
				Rate:      "4",
				Created:   presetCreated,
			})
			return err
		})
		if err != nil {
			t.Fatalf("SaveFeedback() returned error: %v", err)
		}
		if feedback.ID != "f1" {
			t.Errorf("feedback.ID = %q, want f1", feedback.ID)
		}
		savedUser := dbo4userus.NewUserEntry("u1")
		if err = db.Get(ctx, savedUser.Record); err != nil {
			t.Fatalf("failed to read user: %v", err)
		}
		if !savedUser.Data.LastFeedbackAt.Equal(presetCreated) {
			t.Errorf("user.LastFeedbackAt = %v, want %v", savedUser.Data.LastFeedbackAt, presetCreated)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
			_, _, err = SaveFeedback(ctx, tx, "", &models4debtus.FeedbackData{UserStrID: "missing", Rate: "5"})
			return err
		})
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})

	t.Run("panics", func(t *testing.T) {
		for name, f := range map[string]func(){
			"nil_ctx": func() {
				_, _, _ = SaveFeedback(nil, nil, "", &models4debtus.FeedbackData{UserStrID: "u1", Rate: "5"}) //nolint:staticcheck // testing nil ctx panic
			},
			"nil_entity": func() { _, _, _ = SaveFeedback(ctx, nil, "", nil) },
			"no_user_id": func() { _, _, _ = SaveFeedback(ctx, nil, "", &models4debtus.FeedbackData{Rate: "5"}) },
			"no_rate":    func() { _, _, _ = SaveFeedback(ctx, nil, "", &models4debtus.FeedbackData{UserStrID: "u1"}) },
		} {
			t.Run(name, func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic")
					}
				}()
				f()
			})
		}
	})
}
