package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/pkg/modules/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
)

func TestRewardDalGae_InsertReward(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	var rewardID string
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		reward, err := NewRewardDal().InsertReward(ctx, tx, &models4debtus.RewardDbo{})
		if err != nil {
			return err
		}
		if reward.ID == "" {
			t.Error("InsertReward() should assign a non-empty string ID")
		}
		rewardID = reward.ID
		return nil
	})
	if err != nil {
		t.Fatalf("InsertReward() returned error: %v", err)
	}

	// Verify the record was persisted
	loaded := models4debtus.NewReward(rewardID, nil)
	if err = db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("inserted reward not found by ID %q: %v", rewardID, err)
	}
}
