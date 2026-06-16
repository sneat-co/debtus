package delay4reminders

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sneat-co/sneat-go/pkg/modules/debtus/debtusdal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/delayer4debtus"
	"github.com/strongo/delaying"
	"github.com/strongo/i18n"
)

func TestDelaySetReminderIsSent(t *testing.T) {
	var err error

	if err = DelaySetReminderIsSent(context.TODO(), "", time.Now(), 1, "", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("Should fail as reminder is 0")
	}
	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Now(), 0, "", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("Should fail as no message id supplied")
	}
	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Now(), 1, "not empty", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("Should fail as both int and string message ids supplied")
	}
	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Time{}, 1, "not empty", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("Should fail as both int and string message ids supplied")
	}
	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Time{}, 1, "", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("Should fail as both sentAt is zero")
	}

	//countOfCallsToDelay := 0
	//apphostgae.CallDelayFunc = func(ctx context.Context, queueName, subPath string, f *delay.Function, args ...interface{}) error {
	//	countOfCallsToDelay += 1
	//	return nil
	//}
	delayer4debtus.SetReminderIsSent = delaying.NewDelayer("test", debtusdal.DelayedSetReminderIsSent,
		func(c context.Context, params delaying.Params, args ...interface{}) error {
			return nil
		},
		func(c context.Context, params delaying.Params, args ...[]interface{}) error {
			return nil
		},
	)

	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Now(), 1, "", i18n.LocaleCodeEnUS, ""); err != nil {
		t.Error(fmt.Errorf("should NOT fail: %w", err).Error())
	}

	// Cover the EnqueueWork error path with a delayer that always fails.
	wantEnqueueErr := errors.New("enqueue error")
	delayer4debtus.SetReminderIsSent = delaying.NewDelayer("test-err", debtusdal.DelayedSetReminderIsSent,
		func(c context.Context, params delaying.Params, args ...interface{}) error {
			return wantEnqueueErr
		},
		func(c context.Context, params delaying.Params, args ...[]interface{}) error {
			return nil
		},
	)
	if err = DelaySetReminderIsSent(context.TODO(), "1", time.Now(), 1, "", i18n.LocaleCodeEnUS, ""); err == nil {
		t.Error("should fail when EnqueueWork returns an error")
	} else if !errors.Is(err, wantEnqueueErr) {
		t.Errorf("expected wrapped enqueue error, got: %v", err)
	}

	//if countOfCallsToDelay != 1 {
	//	t.Errorf("Expeted to get 1 call to delay, got: %v", countOfCallsToDelay)
	//}
}
