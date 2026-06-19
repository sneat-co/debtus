package debtusdal

import (
	"context"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
)

type FeedbackDal struct {
}

func NewFeedbackDal() FeedbackDal {
	return FeedbackDal{}
}

func (FeedbackDal) GetFeedbackByID(ctx context.Context, tx dal.ReadSession, feedbackID string) (feedback models4debtus.Feedback, err error) {
	if tx == nil {
		if tx, err = facade.GetSneatDB(ctx); err != nil {
			return
		}
	}
	feedback = models4debtus.NewFeedback(feedbackID, nil)
	return feedback, tx.Get(ctx, feedback.Record)
}
