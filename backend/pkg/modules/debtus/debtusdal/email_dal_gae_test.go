package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/auth/models4auth"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

// InsertEmail uses an int64 incomplete key whose ID is auto-assigned by the
// datastore; dalgo2memory does not implement int64 ID generators, so that path
// cannot be exercised here.  UpdateEmail and GetEmailByID use a known int64 ID
// and work fine with the memory adapter.

func TestEmailDalGae_UpdateEmail(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	const id int64 = 55
	// Seed a record with a known ID
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4auth.NewEmail(id, &models4auth.EmailData{
			Status: "pending", Subject: "Sub", From: "a@b.c", To: "c@d.e",
		}).Record)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	err = db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		email := models4auth.NewEmail(id, &models4auth.EmailData{
			Status: "sent", Subject: "Sub", From: "a@b.c", To: "c@d.e",
		})
		return NewEmailDal().UpdateEmail(ctx, tx, email)
	})
	if err != nil {
		t.Fatalf("UpdateEmail() returned error: %v", err)
	}

	loaded := models4auth.NewEmail(id, nil)
	if err = db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if loaded.Data.Status != "sent" {
		t.Errorf("after UpdateEmail Status = %q, want sent", loaded.Data.Status)
	}
}

func TestEmailDalGae_GetEmailByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_email_when_exists", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		const id int64 = 77
		err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, models4auth.NewEmail(id, &models4auth.EmailData{
				Status: "sent", Subject: "s", From: "f@f.f", To: "t@t.t",
			}).Record)
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		email, err := NewEmailDal().GetEmailByID(ctx, db, id)
		if err != nil {
			t.Fatalf("GetEmailByID() returned error: %v", err)
		}
		if email.ID != id {
			t.Errorf("email.ID = %v, want %v", email.ID, id)
		}
		if email.Data.Status != "sent" {
			t.Errorf("email.Status = %q, want sent", email.Data.Status)
		}
	})

	t.Run("returns_not_found_for_missing_email", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		_, err := NewEmailDal().GetEmailByID(ctx, db, 999)
		if !dal.IsNotFound(err) {
			t.Errorf("expected not-found error, got: %v", err)
		}
	})
}
