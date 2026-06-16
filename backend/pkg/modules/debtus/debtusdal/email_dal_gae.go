package debtusdal

import (
	"context"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-core-modules/common4all"
)

type EmailDal struct {
}

func NewEmailDal() EmailDal {
	return EmailDal{}
}

var _ common4all.EmailDal = (*EmailDal)(nil)

func (EmailDal) InsertEmail(ctx context.Context, tx dal.ReadwriteTransaction, data *models4auth.EmailData) (email models4auth.Email, err error) {
	email = models4auth.NewEmail(0, data)
	if err = tx.Insert(ctx, email.Record); err != nil {
		return
	}
	email.ID = email.Record.Key().ID.(int64)
	email.Data = data
	return
}

func (EmailDal) UpdateEmail(ctx context.Context, tx dal.ReadwriteTransaction, email models4auth.Email) (err error) {
	return tx.Set(ctx, email.Record)
}

func (EmailDal) GetEmailByID(ctx context.Context, tx dal.ReadSession, id int64) (email models4auth.Email, err error) {
	email = models4auth.NewEmail(id, nil)
	return email, tx.Get(ctx, email.Record)
}
