// Package list is the simplest reconciler.Source: a function that returns
// every request that may need work, called when the controller starts and
// then on an interval.
//
// It is how a controller over database rows relists at boot and resyncs
// afterwards. The store supplies the function and decides what it lists and
// how; this package only calls it and pushes the results.
package list

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
)

// Source pushes everything a list function returns when it starts and, if
// Every is set, again on an interval. Create one with New.
type Source[request any] struct {
	list  func(context.Context) ([]request, error)
	every time.Duration
}

var _ reconciler.Source[int] = (*Source[int])(nil)

// New returns a source that pushes everything list returns when the
// controller starts. Chain Every to repeat it.
//
// The first list runs inside Start and fails the controller if it fails, so
// a controller never runs against a store it cannot read. Later lists are
// logged if they fail and tried again at the next interval.
func New[request any](list func(context.Context) ([]request, error)) *Source[request] {
	return &Source[request]{list: list}
}

// Every repeats the list on the given interval after the initial one. The
// interval should be as long as correctness allows: load on the store is
// replicas × rows / interval, and event sources are the fast path.
func (s *Source[request]) Every(interval time.Duration) *Source[request] {
	s.every = interval
	return s
}

// Start implements reconciler.Source.
func (s *Source[request]) Start(ctx context.Context, out chan<- request) error {
	if s.list == nil {
		return errors.New("a list function is required")
	}
	if err := s.push(ctx, out); err != nil {
		return fmt.Errorf("initial list failed: %w", err)
	}
	if s.every > 0 {
		go s.run(ctx, out)
	}
	return nil
}

// run lists every interval until ctx is done. Ticks never pile up: a list
// that outlasts an interval delays the next one rather than overlapping it.
func (s *Source[request]) run(ctx context.Context, out chan<- request) {
	logger := logging.LoggerFromContext(ctx)
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if err := s.push(ctx, out); err != nil && ctx.Err() == nil {
			logger.Error(err, "error listing requests")
		}
	}
}

// push lists once and pushes every result.
func (s *Source[request]) push(ctx context.Context, out chan<- request) error {
	reqs, err := s.list(ctx)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		select {
		case out <- req:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
