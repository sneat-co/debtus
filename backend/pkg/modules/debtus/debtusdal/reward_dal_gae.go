package debtusdal

import (
	"context"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/dal4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/models4debtus"
)

func NewRewardDal() rewardDal {
	return rewardDal{}
}

type rewardDal struct {
}

var _ dal4debtus.RewardDal = (*rewardDal)(nil)

func (rewardDal) InsertReward(ctx context.Context, tx dal.ReadwriteTransaction, rewardEntity *models4debtus.RewardDbo) (reward models4debtus.Reward, err error) {
	reward = models4debtus.NewRewardWithIncompleteKey(rewardEntity)
	if err = dal4debtus.InsertWithRandomStringID(ctx, tx, reward.Record); err != nil {
		return
	}
	reward.ID = reward.Record.Key().ID.(string)
	return
}
