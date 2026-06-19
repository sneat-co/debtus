package debtusdal

import (
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/unsorted4auth"
)

func TestRegisterDal(t *testing.T) {
	// Pre-clean
	dal4debtus.Default.Admin = nil
	dal4debtus.Default.Contact = nil
	//dal4debtus.Group = nil
	dal4debtus.Default.Twilio = nil
	dal4debtus.Default.HttpClient = nil
	dal4debtus.Default.Invite = nil
	unsorted4auth.LoginCode = nil
	unsorted4auth.LoginPin = nil
	//dal4debtus.Bill = nil
	dal4debtus.Default.Receipt = nil
	dal4debtus.Default.Reminder = nil
	dal4debtus.Default.Reward = nil
	dal4debtus.Default.Transfer = nil
	unsorted4auth.User = nil
	unsorted4auth.UserGooglePlus = nil
	unsorted4auth.UserFacebook = nil

	// Execute
	RegisterDal()
	// Assert
	if dal4debtus.Default.Admin == nil {
		t.Error("dal4debtus.Default.Admin == nil")
	}
	//if dal4debtus.Bill == nil {
	//	t.Error("dal4debtus.Bill == nil")
	//}
	if dal4debtus.Default.Contact == nil {
		t.Error("dal4debtus.DebtusSpaceContactEntry == nil")
	}
	if dal4debtus.Default.Receipt == nil {
		t.Error("dal4debtus.Default.Receipt == nil")
	}
	if dal4debtus.Default.Reminder == nil {
		t.Error("dal4debtus.Default.Reminder == nil")
	}
	if dal4debtus.Default.Reward == nil {
		t.Error("dal4debtus.Default.Reward == nil")
	}
	//if facade4auth.UserBrowser == nil {
	//	t.Error("dal4debtus.UserBrowser == nil")
	//}
	//if dal4debtus.Bill == nil {
	//	t.Error("dal4debtus.Bill == nil")
	//}
	if dal4debtus.Default.HttpClient == nil {
		t.Error("dal4debtus.Default.HttpClient == nil")
	}
	if dal4debtus.Default.Invite == nil {
		t.Error("dal4debtus.Default.Invite == nil")
	}
	//if dal4debtus.Group == nil {
	//	t.Error("dal4debtus.Default.Invite == nil")
	//}
	//}
	if dal4debtus.Default.Transfer == nil {
		t.Error("dal4debtus.Default.Transfer == nil")
	}
	if dal4debtus.Default.Twilio == nil {
		t.Error("dal4debtus.Default.Twilio == nil")
	}
	if unsorted4auth.User == nil {
		t.Error("dal4debtus.User == nil")
	}
	//if facade4auth.UserBrowser == nil {
	//	t.Error("dal4debtus.UserBrowser == nil")
	//}
	//if facade4auth.UserGaClient == nil {
	//	t.Error("dal4debtus.UserGaClient == nil")
	//}
	if unsorted4auth.UserGooglePlus == nil {
		t.Error("dal4debtus.UserGooglePlus == nil")
	}
	if unsorted4auth.UserFacebook == nil {
		t.Error("dal4debtus.UserFacebook == nil")
	}
	//if facade4auth.UserOneSignal == nil {
	//	t.Error("dal4debtus.UserOneSignal == nil")
	//}
}
