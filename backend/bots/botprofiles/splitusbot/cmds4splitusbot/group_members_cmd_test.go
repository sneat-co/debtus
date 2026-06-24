package cmds4splitusbot

import (
	"context"
	"strings"
	"testing"

	"github.com/sneat-co/contactus-ext/backend/contactusmodels/briefs4contactus"
	"github.com/sneat-co/contactus-ext/backend/contactusmodels/const4contactus"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func newTestTranslator(ctx context.Context) i18n.SingleLocaleTranslator {
	return i18n.NewSingleMapTranslator(
		i18n.LocalesByCode5[i18n.LocaleCodeEnUK],
		i18n.NewMapTranslator(ctx, i18n.LocaleCodeEnUK, trans.TRANS),
	)
}

func TestGroupMembersCard(t *testing.T) {
	ctx := context.Background()
	translator := newTestTranslator(ctx)

	t.Run("no_members", func(t *testing.T) {
		contactusSpace := dal4contactus.NewContactusSpaceEntry("space1")
		text, err := groupMembersCard(ctx, translator, contactusSpace, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if text == "" {
			t.Error("expected non-empty card text")
		}
	})

	t.Run("with_members", func(t *testing.T) {
		contactusSpace := dal4contactus.NewContactusSpaceEntry("space1")
		brief := &briefs4contactus.ContactBrief{
			Type:  briefs4contactus.ContactTypePerson,
			Title: "First Member",
		}
		brief.Roles = []string{const4contactus.SpaceMemberRoleMember}
		contactusSpace.Data.AddContact("c1", brief)
		text, err := groupMembersCard(ctx, translator, contactusSpace, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(text, "First Member") {
			t.Errorf("expected member name in card text, got: %q", text)
		}
	})
}
