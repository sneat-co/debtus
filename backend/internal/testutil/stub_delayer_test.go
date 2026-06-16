package testutil

import (
	"context"
	"errors"
	"testing"
)

func TestStubDelayer(t *testing.T) {
	t.Run("ID and Implementation", func(t *testing.T) {
		s := &StubDelayer{}
		if got := s.ID(); got != "stub" {
			t.Errorf("ID: got %q want %q", got, "stub")
		}
		if got := s.Implementation(); got != s {
			t.Errorf("Implementation: got %v want %v", got, s)
		}
	})

	t.Run("EnqueueWork records and returns nil", func(t *testing.T) {
		s := &StubDelayer{}
		if err := s.EnqueueWork(context.Background(), nil, "a", "b"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		calls := s.Calls()
		if len(calls) != 1 || len(calls[0]) != 2 {
			t.Fatalf("expected 1 call with 2 args, got %v", calls)
		}
	})

	t.Run("EnqueueWork returns Err", func(t *testing.T) {
		errExp := errors.New("boom")
		s := &StubDelayer{Err: errExp}
		if err := s.EnqueueWork(context.Background(), nil); err != errExp {
			t.Errorf("got %v want %v", err, errExp)
		}
	})

	t.Run("EnqueueWorkMulti records and returns nil", func(t *testing.T) {
		s := &StubDelayer{}
		if err := s.EnqueueWorkMulti(context.Background(), nil, []any{"a"}, []any{"b", "c"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := s.Calls(); len(calls) != 2 {
			t.Fatalf("expected 2 calls, got %d", len(calls))
		}
	})

	t.Run("EnqueueWorkMulti returns Err", func(t *testing.T) {
		errExp := errors.New("multi")
		s := &StubDelayer{Err: errExp}
		if err := s.EnqueueWorkMulti(context.Background(), nil); err != errExp {
			t.Errorf("got %v want %v", err, errExp)
		}
	})
}
