package facade4splitusbot

import (
	"context"

	"github.com/sneat-co/sneat-core-modules/userus/const4userus"
	"github.com/strongo/delaying"
)

var DelayerUpdateUserWithGroups delaying.Delayer

func delayUpdateUserWithGroups(ctx context.Context, userID string, groupIDs2add, groupIDs2remove []string) (err error) { // TODO: make args meaningful
	args := []any{userID, groupIDs2add, groupIDs2remove}
	params := delaying.With(const4userus.QueueUsers, "update-user-with-groups", 0)
	return DelayerUpdateUserWithGroups.EnqueueWorkMulti(ctx, params, args)
}
