package models4splitus

import (
	"testing"

	"github.com/sneat-co/sneat-go/pkg/modules/splitus/briefs4splitus"
)

func TestNewGroupEntry(t *testing.T) {
	t.Run("nil_data", func(t *testing.T) {
		e := NewGroupEntry("g1", nil)
		if e.Data == nil {
			t.Fatal("expected non-nil Data for nil input")
		}
		if e.ID != "g1" {
			t.Errorf("expected ID=g1, got %v", e.ID)
		}
	})

	t.Run("non_nil_data", func(t *testing.T) {
		d := &GroupDbo{CreatorUserID: "u1", Name: "Test"}
		e := NewGroupEntry("g2", d)
		if e.Data != d {
			t.Error("expected Data to be the provided pointer")
		}
	})
}

func TestNewGroupKey(t *testing.T) {
	t.Run("empty_id_generates_random", func(t *testing.T) {
		k := NewGroupKey("")
		if k == nil {
			t.Fatal("expected non-nil key")
		}
		if k.ID == nil {
			t.Error("expected non-nil key ID")
		}
	})

	t.Run("non_empty_id", func(t *testing.T) {
		k := NewGroupKey("g42")
		if k == nil {
			t.Fatal("expected non-nil key")
		}
	})
}

func TestGroupDbo_GetTelegramGroups(t *testing.T) {
	t.Run("empty_json_returns_nil", func(t *testing.T) {
		g := &GroupDbo{}
		groups, err := g.GetTelegramGroups()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if groups != nil {
			t.Errorf("expected nil groups for empty JSON, got %v", groups)
		}
	})

	t.Run("valid_json", func(t *testing.T) {
		g := &GroupDbo{}
		input := []briefs4splitus.GroupTgChatJson{{ChatID: 123, Title: "chat"}}
		g.SetTelegramGroups(input)

		got, err := g.GetTelegramGroups()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].ChatID != 123 {
			t.Errorf("unexpected groups: %+v", got)
		}
	})

	t.Run("cached_returns_without_unmarshal", func(t *testing.T) {
		g := &GroupDbo{}
		input := []briefs4splitus.GroupTgChatJson{{ChatID: 456}}
		g.SetTelegramGroups(input)

		// first call populates cache
		_, _ = g.GetTelegramGroups()
		// second call hits cached path
		got, err := g.GetTelegramGroups()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].ChatID != 456 {
			t.Errorf("unexpected cached groups: %+v", got)
		}
	})

	t.Run("count_mismatch_error", func(t *testing.T) {
		g := &GroupDbo{
			TelegramGroupsJson:  `[{"chatID":1}]`,
			TelegramGroupsCount: 99,
		}
		_, err := g.GetTelegramGroups()
		if err == nil {
			t.Error("expected error for count mismatch")
		}
	})
}

func TestGroupDbo_SetTelegramGroups(t *testing.T) {
	g := &GroupDbo{}
	groups := []briefs4splitus.GroupTgChatJson{{ChatID: 1, Title: "A"}, {ChatID: 2, Title: "B"}}
	changed := g.SetTelegramGroups(groups)
	if !changed {
		t.Error("expected changed=true on first set")
	}
	if g.TelegramGroupsCount != 2 {
		t.Errorf("expected count=2, got %d", g.TelegramGroupsCount)
	}
	if g.TelegramGroupsJson == "" {
		t.Error("expected non-empty TelegramGroupsJson")
	}

	// Setting same data should not change
	changed = g.SetTelegramGroups(groups)
	if changed {
		t.Error("expected changed=false when setting same data")
	}
}

func TestGroupDbo_Validate(t *testing.T) {
	t.Run("missing_creator", func(t *testing.T) {
		g := &GroupDbo{Name: "Test"}
		if err := g.Validate(); err == nil {
			t.Error("expected error for missing CreatorUserID")
		}
	})

	t.Run("blank_name", func(t *testing.T) {
		g := &GroupDbo{CreatorUserID: "u1", Name: "   "}
		if err := g.Validate(); err == nil {
			t.Error("expected error for blank Name")
		}
	})

	t.Run("valid", func(t *testing.T) {
		g := &GroupDbo{CreatorUserID: "u1", Name: "Test Group"}
		if err := g.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
