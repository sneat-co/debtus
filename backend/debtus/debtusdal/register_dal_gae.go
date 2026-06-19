package debtusdal

import (
	"context"
	"net/http"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/facade4auth"
	"github.com/sneat-co/sneat-core-modules/auth/unsorted4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
	"github.com/sneat-co/sneat-go-core/facade"
)

func RegisterDal() {

	// R1: the DAL is constructed atomically — no partial registration, no
	// nil service-locator globals (see docs/DEBTUS-ARCHITECTURE-REVIEW.md, R1).
	dal4debtus.Default = NewDAL()

	unsorted4auth.User = facade4auth.NewUserDalGae()
	unsorted4auth.UserGoogle = facade4auth.NewUserGoogleDalGae()
	unsorted4auth.PasswordReset = facade4auth.NewPasswordResetDalGae()
	common4all.Email = NewEmailDal()
	unsorted4auth.UserGooglePlus = facade4auth.NewUserGooglePlusDalGae()
	unsorted4auth.UserEmail = facade4auth.NewUserEmailGaeDal()
	unsorted4auth.UserFacebook = facade4auth.NewUserFacebookDalGae()
	unsorted4auth.LoginPin = facade4auth.NewLoginPinDalGae()
	unsorted4auth.LoginCode = facade4auth.NewLoginCodeDalGae()
	//dal4debtus.HttpAppHost = apphostgae.NewHttpAppHostGAE()
}

type ApiBotHost struct {
}

func (h ApiBotHost) Context(r *http.Request) context.Context {
	return r.Context()
}

func (h ApiBotHost) GetHTTPClient(ctx context.Context) *http.Client {
	return dal4debtus.Default.HttpClient(ctx)
}

//func (h ApiBotHost) GetBotCoreStores(platform string, appContext botsfw.BotAppContext, r *http.Request) botsfwdal.DataAccess {
//	panic("Not implemented")
//}

func (h ApiBotHost) DB(ctx context.Context) (dal.DB, error) {
	return facade.GetSneatDB(ctx)
}
