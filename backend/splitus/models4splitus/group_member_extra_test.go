package models4splitus

import "testing"

func TestNewGroupMember(t *testing.T) {
	t.Run("nil_data", func(t *testing.T) {
		gm := NewGroupMember(1, nil)
		if gm.Data == nil {
			t.Fatal("expected non-nil Data for nil input")
		}
	})

	t.Run("non_nil_data", func(t *testing.T) {
		d := &GroupMemberData{Name: "Alice"}
		gm := NewGroupMember(1, d)
		if gm.Data != d {
			t.Error("expected Data to be the provided pointer")
		}
	})
}

func TestNewGroupMemberKey(t *testing.T) {
	t.Run("zero_id_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for id=0")
			}
		}()
		NewGroupMemberKey(0)
	})

	t.Run("non_zero_id", func(t *testing.T) {
		k := NewGroupMemberKey(42)
		if k == nil {
			t.Fatal("expected non-nil key")
		}
	})
}

func TestNewGroupMemberIncompleteKey(t *testing.T) {
	k := NewGroupMemberIncompleteKey()
	if k == nil {
		t.Fatal("expected non-nil incomplete key")
	}
}
