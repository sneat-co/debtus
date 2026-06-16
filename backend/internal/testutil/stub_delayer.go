// Package testutil provides shared test helpers for the sneat-go module.
package testutil

import (
	"context"
	"sync"

	"github.com/strongo/delaying"
)

// StubDelayer is a test double for delaying.Delayer that records calls and
// optionally returns a configured error.
type StubDelayer struct {
	Err   error
	mu    sync.Mutex
	calls [][]any
}

var _ delaying.Delayer = (*StubDelayer)(nil)

// ID returns a fixed stub identifier.
func (s *StubDelayer) ID() string { return "stub" }

// Implementation returns the stub itself.
func (s *StubDelayer) Implementation() any { return s }

// EnqueueWork records the call and returns Err (if set).
func (s *StubDelayer) EnqueueWork(_ context.Context, _ delaying.Params, args ...any) error {
	s.mu.Lock()
	s.calls = append(s.calls, args)
	s.mu.Unlock()
	return s.Err
}

// EnqueueWorkMulti records each call set and returns Err (if set).
func (s *StubDelayer) EnqueueWorkMulti(_ context.Context, _ delaying.Params, args ...[]any) error {
	s.mu.Lock()
	s.calls = append(s.calls, args...)
	s.mu.Unlock()
	return s.Err
}

// Calls returns a snapshot of all recorded call argument slices.
func (s *StubDelayer) Calls() [][]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]any, len(s.calls))
	copy(out, s.calls)
	return out
}
