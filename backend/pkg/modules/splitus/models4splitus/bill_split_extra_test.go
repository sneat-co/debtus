package models4splitus

import "testing"

func TestSplit_Kind(t *testing.T) {
	bs := Split{}
	if bs.Kind() != SplitsCollection {
		t.Errorf("expected Kind=%v, got %v", SplitsCollection, bs.Kind())
	}
}

func TestSplit_Entity(t *testing.T) {
	se := &SplitEntity{}
	bs := Split{SplitEntity: se}
	if bs.Entity() != se {
		t.Error("expected Entity() to return SplitEntity")
	}
}

func TestSplit_NewEntity(t *testing.T) {
	bs := Split{}
	e := bs.NewEntity()
	if e == nil {
		t.Fatal("expected non-nil new entity")
	}
	if _, ok := e.(*SplitEntity); !ok {
		t.Error("expected *SplitEntity from NewEntity()")
	}
}

func TestSplit_SetEntity(t *testing.T) {
	bs := &Split{}
	se := &SplitEntity{}
	bs.SetEntity(se)
	if bs.SplitEntity != se {
		t.Error("expected SplitEntity to be set")
	}

	bs.SetEntity(nil)
	if bs.SplitEntity != nil {
		t.Error("expected SplitEntity to be nil after SetEntity(nil)")
	}
}
