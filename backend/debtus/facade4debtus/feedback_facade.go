package facade4debtus

import (
	"context"
	"fmt"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/strongo/logus"
)

func SaveFeedback(ctx context.Context, tx dal.ReadwriteTransaction, feedbackID string, feedbackEntity *models4debtus.FeedbackData) (feedback models4debtus.Feedback, user dbo4userus.UserEntry, err error) {
	if ctx == nil {
		panic("ctx == nil")
	}
	logus.Debugf(ctx, "FeedbackDal.SaveFeedback(feedbackEntity:%v)", feedbackEntity)
	if feedbackEntity == nil {
		panic("feedbackEntity == nil")
	}
	if feedbackEntity.UserStrID == "" {
		panic("feedbackEntity.UserStrID is empty string")
	}
	if feedbackEntity.Rate == "" {
		panic("feedbackEntity.Rate is empty string")
	}
	if feedbackID == "" {
		feedback = models4debtus.NewFeedbackWithIncompleteKey(feedbackEntity)
	} else {
		feedback = models4debtus.NewFeedback(feedbackID, feedbackEntity)
	}
	user = dbo4userus.NewUserEntry(feedbackEntity.UserStrID)
	if err = dal4userus.GetUser(ctx, tx, user); err != nil {
		return
	}
	user.Data.LastFeedbackRate = feedbackEntity.Rate
	if feedbackEntity.Created.IsZero() {
		now := time.Now()
		user.Data.LastFeedbackAt = now
		feedbackEntity.Created = now
	} else {
		user.Data.LastFeedbackAt = feedbackEntity.Created
	}
	if feedbackID == "" {
		if err = tx.Insert(ctx, feedback.Record); err != nil {
			err = fmt.Errorf("failed to insert feedback entity: %w", err)
			return
		}
		if err = tx.Set(ctx, user.Record); err != nil {
			err = fmt.Errorf("failed to put user entity to datastore: %w", err)
		}
		return
	}
	if err = tx.SetMulti(ctx, []dal.Record{feedback.Record, user.Record}); err != nil {
		err = fmt.Errorf("failed to put feedback & user entities to datastore: %w", err)
	}
	return
}
