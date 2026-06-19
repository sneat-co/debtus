package debtusdal

import (
	"context"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-bots/pkg/sneattesting"
	"github.com/strongo/strongoapp"
)

func TestCreatePersonalInvite(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	invite, err := NewInviteDal().CreatePersonalInvite(ec, "u1", models4debtus.InviteByEmail, "Friend@Example.com", "telegram", "DebtusBot", "")
	if err != nil {
		t.Fatalf("CreatePersonalInvite() returned error: %v", err)
	}
	if invite.ID == "" || invite.ID == AUTO_GENERATE_INVITE_CODE {
		t.Fatalf("expected auto-generated invite code, got %q", invite.ID)
	}
	if len(invite.ID) != INVITE_CODE_LENGTH {
		t.Errorf("invite code length = %d, want %d", len(invite.ID), INVITE_CODE_LENGTH)
	}
	if id, _ := invite.Record.Key().ID.(string); id != invite.ID {
		t.Errorf("record key ID %q does not match invite.ID %q", id, invite.ID)
	}

	stored := models4debtus.NewInvite(invite.ID, nil)
	if err = db.Get(ctx, stored.Record); err != nil {
		t.Fatalf("stored invite not found by code %q: %v", invite.ID, err)
	}
	if stored.Data.ToEmail != "friend@example.com" {
		t.Errorf("stored invite ToEmail = %q, want lower-cased friend@example.com", stored.Data.ToEmail)
	}
	if stored.Data.ToEmailOriginal != "Friend@Example.com" {
		t.Errorf("stored invite ToEmailOriginal = %q", stored.Data.ToEmailOriginal)
	}
	if stored.Data.CreatedByUserID != "u1" {
		t.Errorf("stored invite CreatedByUserID = %q, want u1", stored.Data.CreatedByUserID)
	}
}

func TestCreateMassInvite(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)

	invite, err := NewInviteDal().CreateMassInvite(ec, "u1", "PARTY", 100, "telegram")
	if err != nil {
		t.Fatalf("CreateMassInvite() returned error: %v", err)
	}
	if invite.ID != "PARTY" {
		t.Errorf("invite.ID = %q, want PARTY", invite.ID)
	}
	stored := models4debtus.NewInvite("PARTY", nil)
	if err = db.Get(ctx, stored.Record); err != nil {
		t.Fatalf("stored invite not found: %v", err)
	}
	if stored.Data.MaxClaimsCount != 100 {
		t.Errorf("stored invite MaxClaimsCount = %d, want 100", stored.Data.MaxClaimsCount)
	}
}

func TestCreateInvite_inputValidation(t *testing.T) {
	ctx := context.Background()
	sneattesting.SetupMemoryDB(t)
	ec := strongoapp.NewExecutionContext(ctx)
	inviteDal := NewInviteDal()

	for name, tc := range map[string]struct {
		inviteBy models4debtus.InviteBy
		address  string
		related  string
		wantErr  string
	}{
		"missing_email":  {models4debtus.InviteByEmail, "", "", "email address is not supplied"},
		"invalid_email":  {models4debtus.InviteByEmail, "not-an-email", "", "invalid email address"},
		"invalid_phone":  {models4debtus.InviteBySms, "not-a-number", "", "invalid syntax"},
		"invalid_relate": {models4debtus.InviteByEmail, "a@b.c", "no-equals-sign", "invalid format for related"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := inviteDal.CreatePersonalInvite(ec, "u1", tc.inviteBy, tc.address, "telegram", "DebtusBot", tc.related)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestNewInvite_recordDataRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)
	err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, models4debtus.NewInvite("CODE1", &models4debtus.InviteData{CreatedByUserID: "u1"}).Record)
	})
	if err != nil {
		t.Fatalf("failed to store invite: %v", err)
	}
	loaded := models4debtus.NewInvite("CODE1", nil)
	if err = db.Get(ctx, loaded.Record); err != nil {
		t.Fatalf("failed to load invite: %v", err)
	}
	if loaded.Data.CreatedByUserID != "u1" {
		t.Errorf("loaded.Data.CreatedByUserID = %q, want u1", loaded.Data.CreatedByUserID)
	}
}

func TestUpdateUserContactDetails(t *testing.T) {
	newUser := func() dbo4userus.UserEntry {
		return dbo4userus.NewUserEntry("u1")
	}

	t.Run("email_sets_primary_and_verified", func(t *testing.T) {
		user := newUser()
		changed := updateUserContactDetails(user, models4debtus.InviteData{
			Channel: string(models4debtus.InviteByEmail),
			ToEmail: "friend@example.com", ToEmailOriginal: "Friend@Example.com",
		})
		if !changed {
			t.Error("expected changed=true")
		}
		if user.Data.Email != "friend@example.com" || !user.Data.EmailVerified {
			t.Errorf("user email = %q verified=%v", user.Data.Email, user.Data.EmailVerified)
		}
		props := user.Data.Emails["friend@example.com"]
		if props == nil || !props.Verified || props.Original != "Friend@Example.com" {
			t.Errorf("email props = %+v", props)
		}
	})

	t.Run("email_noop_when_already_verified", func(t *testing.T) {
		user := newUser()
		inviteData := models4debtus.InviteData{Channel: string(models4debtus.InviteByEmail), ToEmail: "friend@example.com"}
		_ = updateUserContactDetails(user, inviteData)
		if changed := updateUserContactDetails(user, inviteData); changed {
			t.Error("expected changed=false on second application")
		}
	})

	t.Run("email_empty_is_noop", func(t *testing.T) {
		user := newUser()
		if changed := updateUserContactDetails(user, models4debtus.InviteData{Channel: string(models4debtus.InviteByEmail)}); changed {
			t.Error("expected changed=false for empty email")
		}
	})

	t.Run("email_marks_existing_unverified_as_verified", func(t *testing.T) {
		user := newUser()
		_ = updateUserContactDetails(user, models4debtus.InviteData{Channel: string(models4debtus.InviteByEmail), ToEmail: "friend@example.com"})
		user.Data.Emails["friend@example.com"].Verified = false
		user.Data.EmailVerified = true
		if changed := updateUserContactDetails(user, models4debtus.InviteData{Channel: string(models4debtus.InviteByEmail), ToEmail: "friend@example.com"}); !changed {
			t.Error("expected changed=true when props were unverified")
		}
		if !user.Data.Emails["friend@example.com"].Verified {
			t.Error("expected props re-verified")
		}
	})

	t.Run("sms_adds_verified_phone", func(t *testing.T) {
		user := newUser()
		changed := updateUserContactDetails(user, models4debtus.InviteData{
			Channel: string(models4debtus.InviteBySms), ToPhoneNumber: 353857000000,
		})
		if !changed {
			t.Error("expected changed=true")
		}
		props := user.Data.Phones["353857000000"]
		if props == nil || !props.Verified {
			t.Errorf("phone props = %+v", props)
		}
	})

	t.Run("sms_noop_when_phone_known_and_verified", func(t *testing.T) {
		user := newUser()
		inviteData := models4debtus.InviteData{Channel: string(models4debtus.InviteBySms), ToPhoneNumber: 353857000000}
		_ = updateUserContactDetails(user, inviteData)
		if changed := updateUserContactDetails(user, inviteData); changed {
			t.Error("expected changed=false on second application")
		}
	})

	t.Run("sms_zero_phone_is_noop", func(t *testing.T) {
		user := newUser()
		if changed := updateUserContactDetails(user, models4debtus.InviteData{Channel: string(models4debtus.InviteBySms)}); changed {
			t.Error("expected changed=false for zero phone")
		}
	})

	t.Run("sms_marks_existing_unverified_as_verified", func(t *testing.T) {
		user := newUser()
		inviteData := models4debtus.InviteData{Channel: string(models4debtus.InviteBySms), ToPhoneNumber: 353857000000}
		_ = updateUserContactDetails(user, inviteData)
		user.Data.Phones["353857000000"].Verified = false
		if changed := updateUserContactDetails(user, inviteData); !changed {
			t.Error("expected changed=true when props were unverified")
		}
	})

	t.Run("unknown_channel_is_noop", func(t *testing.T) {
		user := newUser()
		if changed := updateUserContactDetails(user, models4debtus.InviteData{Channel: "carrier-pigeon"}); changed {
			t.Error("expected changed=false for unknown channel")
		}
	})
}
