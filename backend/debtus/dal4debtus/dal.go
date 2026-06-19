package dal4debtus

import (
	"context"
	"net/http"
)

// DAL groups all Debtus data-access interfaces behind a single value so that
// dependencies are explicit and constructed atomically. It replaces the
// package-level service-locator vars (dal4debtus.Default.Transfer, dal4debtus.Default.Receipt, ...),
// whose partial registration caused nil-panics at a distance (see
// docs/DEBTUS-ARCHITECTURE-REVIEW.md, R1).
//
// Construction lives in debtusdal.NewDAL() — it cannot live here because
// debtusdal imports dtdal for the interface types.
type DAL struct {
	Contact  ContactDal
	Feedback FeedbackDal
	Receipt  ReceiptDal
	Reminder ReminderDal
	Transfer TransferDal
	Twilio   TwilioDal
	Invite   InviteDal
	Admin    AdminDal
	Reward   RewardDal

	HttpClient func(ctx context.Context) *http.Client
}

// Default is the transitional process-wide instance, assigned exactly once by
// debtusdal.RegisterDal via debtusdal.NewDAL(). Unlike the legacy per-service
// globals it is all-or-nothing: NewDAL populates every field, and
// debtusdal's completeness test fails if any field is nil.
//
// Migration path: consumers move from the legacy globals to Default, then
// facades take a DAL parameter explicitly and Default remains only at
// composition roots (bot command and HTTP handler wiring).
var Default DAL
