package facade4debtus

import (
	"context"
	"testing"

	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/delaying"
)

func initDelayers4Test(t *testing.T) {
	t.Helper()
	delaying.Init(delaying.VoidWithLog)
	if delayerUpdateUserHasDueTransfers == nil {
		InitDelays4debtus(delaying.MustRegisterFunc)
	}
}

func TestInitDelays4debtus(t *testing.T) {
	registered := make(map[string]any)
	InitDelays4debtus(func(key string, i any) delaying.Delayer {
		registered[key] = i
		return delaying.VoidWithLog(key, i)
	})
	for _, key := range []string{"delayedUpdateUserHasDueTransfers", "delayedUpdateSpaceHasDueTransfers"} {
		if registered[key] == nil {
			t.Errorf("delayed func %q not registered", key)
		}
	}
}

func TestDelayUpdateHasDueTransfers(t *testing.T) {
	initDelayers4Test(t)
	ctx := context.Background()

	if err := DelayUpdateHasDueTransfers(ctx, "", testSpaceID); err == nil {
		t.Error("expected error for empty userID")
	}
	if err := DelayUpdateHasDueTransfers(ctx, "u1", ""); err == nil {
		t.Error("expected error for empty spaceID")
	}
	if err := DelayUpdateHasDueTransfers(ctx, "u1", testSpaceID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelayedUpdateUserHasDueTransfers(t *testing.T) {
	ctx := context.Background()

	t.Run("empty_user_id_is_logged_and_ignored", func(t *testing.T) {
		sneattesting.SetupMemoryDB(t)
		if err := delayedUpdateUserHasDueTransfers(ctx, "", string(testSpaceID)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("already_has_due_transfers", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		debtusUser := models4debtus.NewDebtusUserEntry("u1")
		debtusUser.Data.HasDueTransfers = true
		seedRecords(t, db, debtusUser.Record)
		if err := delayedUpdateUserHasDueTransfers(ctx, "u1", string(testSpaceID)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no_due_transfers_found", func(t *testing.T) {
		// The transfers query legitimately returns no rows on an empty DB, so
		// the worker should finish without error and without updating records.
		sneattesting.SetupMemoryDB(t)
		if err := delayedUpdateUserHasDueTransfers(ctx, "u1", string(testSpaceID)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
