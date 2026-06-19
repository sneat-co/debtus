package debtusdal

import (
	"context"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/sneat-co/sneat-core-modules/contactus/briefs4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactus/dbo4contactus"
	"github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/strongo/strongoapp/person"
)

func TestRewardDal_InsertReward(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	var reward models4debtus.Reward
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) (err error) {
		reward, err = NewRewardDal().InsertReward(ctx, tx, &models4debtus.RewardDbo{})
		return
	})
	if err != nil {
		t.Fatalf("InsertReward() returned error: %v", err)
	}
	if reward.ID == "" {
		t.Error("InsertReward() should assign a non-empty random string ID")
	}
}

func TestContactDal_GetContactIDsByTitle_matches(t *testing.T) {
	ctx := context.Background()
	const spaceID coretypes.SpaceID = "space1"

	seedSpace := func(t *testing.T, db dal.DB) {
		t.Helper()
		space := dal4contactus.NewContactusSpaceEntryWithData(spaceID, &dbo4contactus.ContactusSpaceDbo{})
		space.Data.Contacts = map[string]*briefs4contactus.ContactBrief{
			"c1": {Names: &person.NameFields{FullName: "Alice Smith"}},
			"c2": {Names: &person.NameFields{FullName: "bob jones"}},
		}
		seedRecord(t, ctx, db, space.Record)
	}

	t.Run("case_sensitive_match", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedSpace(t, db)
		ids, err := NewContactDal().GetContactIDsByTitle(ctx, db, spaceID, "u1", "Alice Smith", true)
		if err != nil {
			t.Fatalf("GetContactIDsByTitle() returned error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "c1" {
			t.Errorf("GetContactIDsByTitle() = %v, want [c1]", ids)
		}
	})

	t.Run("case_sensitive_no_match", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedSpace(t, db)
		ids, err := NewContactDal().GetContactIDsByTitle(ctx, db, spaceID, "u1", "alice smith", true)
		if err != nil {
			t.Fatalf("GetContactIDsByTitle() returned error: %v", err)
		}
		if len(ids) != 0 {
			t.Errorf("GetContactIDsByTitle() = %v, want empty", ids)
		}
	})

	t.Run("case_insensitive_match", func(t *testing.T) {
		db := sneattesting.SetupMemoryDB(t)
		seedSpace(t, db)
		ids, err := NewContactDal().GetContactIDsByTitle(ctx, db, spaceID, "u1", "BOB JONES", false)
		if err != nil {
			t.Fatalf("GetContactIDsByTitle() returned error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "c2" {
			t.Errorf("GetContactIDsByTitle() = %v, want [c2]", ids)
		}
	})
}
