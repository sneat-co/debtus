package debtusdal

import (
	"context"
	"net/http"

	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/dal4debtus"
)

// NewDAL returns a fully populated dal4debtus.DAL backed by the dalgo
// implementations in this package. Every field must be assigned here —
// TestNewDAL_populatesEveryField fails otherwise, replacing the legacy
// risk of nil service-locator globals panicking at a distance.
func NewDAL() dal4debtus.DAL {
	return dal4debtus.DAL{
		Contact:  NewContactDal(),
		Feedback: NewFeedbackDal(),
		Receipt:  NewReceiptDal(),
		Reminder: NewReminderDal(),
		Transfer: NewTransferDal(),
		Twilio:   NewTwilioDal(),
		Invite:   NewInviteDal(),
		Admin:    NewAdminDal(),
		Reward:   NewRewardDal(),
		HttpClient: func(ctx context.Context) *http.Client {
			return http.DefaultClient
		},
	}
}
