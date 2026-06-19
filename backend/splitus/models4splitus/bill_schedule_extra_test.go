package models4splitus

import (
	"testing"
	"time"
)

func TestNewBillScheduleKey(t *testing.T) {
	k := NewBillScheduleKey(7)
	if k == nil {
		t.Fatal("expected non-nil key")
	}
	if k.Collection() != BillScheduleKind {
		t.Errorf("expected kind=%v, got %v", BillScheduleKind, k.Collection())
	}
}

func TestNewBillScheduleIncompleteKey(t *testing.T) {
	k := NewBillScheduleIncompleteKey()
	if k == nil {
		t.Fatal("expected non-nil incomplete key")
	}
}

func TestBillScheduleEntity_Validate(t *testing.T) {
	t.Run("missing_split_mode_error", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		bc.SplitMode = ""
		entity := &BillScheduleEntity{BillCommon: bc}
		if err := entity.Validate(); err == nil {
			t.Error("expected error for missing SplitMode")
		}
	})

	t.Run("valid", func(t *testing.T) {
		bc := newValidBillCommon()
		setCreatedAt(&bc)
		entity := &BillScheduleEntity{BillCommon: bc}
		if err := entity.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestBillSchedule_Kind(t *testing.T) {
	bs := BillSchedule{}
	if bs.Kind() != BillKind {
		t.Errorf("expected Kind=%v, got %v", BillKind, bs.Kind())
	}
}

func TestBillSchedule_Entity(t *testing.T) {
	t.Run("nil_data_allocates", func(t *testing.T) {
		bs := &BillSchedule{}
		e := bs.Entity()
		if e == nil {
			t.Fatal("expected non-nil entity")
		}
		if bs.Data == nil {
			t.Error("expected Data to be set")
		}
	})

	t.Run("non_nil_data_returns_same", func(t *testing.T) {
		entity := &BillScheduleEntity{BillCommon: BillCommon{CreatorUserID: "u1"}}
		bs := &BillSchedule{Data: entity}
		e := bs.Entity()
		if e != entity {
			t.Error("expected same pointer returned")
		}
	})
}

func TestBillSchedule_SetEntity(t *testing.T) {
	bs := &BillSchedule{}
	entity := &BillScheduleEntity{BillCommon: BillCommon{
		CreatorUserID: "u1",
		SplitMode:     SplitModeEqually,
		Status:        BillStatusDraft,
	}}
	entity.CreatedAt = time.Now()
	bs.SetEntity(entity)
	if bs.Data != entity {
		t.Error("expected Data to be set to the provided entity")
	}
}
