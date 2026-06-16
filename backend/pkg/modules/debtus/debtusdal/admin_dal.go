package debtusdal

import (
	"context"
	"errors"

	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
)

type AdminDal struct {
}

func NewAdminDal() AdminDal {
	return AdminDal{}
}

func (AdminDal) LatestUsers(ctx context.Context) (users []dbo4userus.UserEntry, err error) {
	return nil, errors.New("not implemented")
	//var (
	//	userKeys     []*datastore.Key
	//	userEntities []*models.DebutsAppUserDataOBSOLETE
	//)
	//query := datastore.NewQuery(models.AppUserKind).Order("-DtCreated").Limit(20)
	//if userKeys, err = query.GetAll(ctx, &userEntities); err != nil {
	//	return
	//}
	//users = make([]models.AppUserOBSOLETE, len(userKeys))
	//for i, userEntity := range userEntities {
	//	users[i] = models.NewAppUserOBSOLETE(userKeys[i].IntID(), userEntity)
	//}
	//return
}

func (AdminDal) DeleteAll(_ context.Context, botCode, botChatID string) error {
	panic("not implemented")
}
