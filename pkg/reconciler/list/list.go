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
	"math/rand/v2"
	"time"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
)

// Source pushes everything a list function returns when it starts and, if
// Every is set, again on an interval. Create one with New.
type Source[request any] struct {
	list   func(context.Context) ([]request, error)
	every  time.Duration
	jitter float64
	// random returns a value in [0, 1). Tests replace it.
	random func() float64
}

var _ reconciler.Source[int] = (*Source[int])(nil)

// New returns a source that pushes everything list returns when the
// controller starts. Chain Every to repeat it.
//
// The first list runs inside Start and fails the controller if it fails, so
// a controller never runs against a store it cannot read. Later lists are
// logged if they fail and tried again at the next interval.
func New[request any](list func(context.Context) ([]request, error)) *Source[request] {
	return &Source[request]{list: list, random: rand.Float64}
}

// Every repeats the list on the given interval after the initial one. The
// interval should be as long as correctness allows: load on the store is
// replicas × rows / interval, and event sources are the fast path.
func (s *Source[request]) Every(interval time.Duration) *Source[request] {
	s.every = interval
	return s
}

// Jitter lengthens each wait between lists by a random fraction of the
// interval, up to maxFactor. With Every(time.Minute) and Jitter(0.5), lists
// are 60 to 90 seconds apart. It spreads the load of replicas that started
// together, or of several controllers on one store, so that their lists stop
// landing at the same instant. The initial list is not delayed. maxFactor
// must not be negative; it has no effect without Every.
func (s *Source[request]) Jitter(maxFactor float64) *Source[request] {
	s.jitter = maxFactor
	return s
}

// Start implements reconciler.Source.
func (s *Source[request]) Start(ctx context.Context, out chan<- request) error {
	if s.list == nil {
		return errors.New("a list function is required")
	}
	if s.jitter < 0 {
		return fmt.Errorf("jitter must not be negative, got %v", s.jitter)
	}
	if err := s.push(ctx, out); err != nil {
		return fmt.Errorf("initial list failed: %w", err)
	}
	if s.every > 0 {
		go s.run(ctx, out)
	}
	return nil
}

// run lists on the interval until ctx is done. Each wait begins after the
// previous list finishes, so lists never overlap or pile up: a list that
// outlasts the interval delays the next one.
func (s *Source[request]) run(ctx context.Context, out chan<- request) {
	logger := logging.LoggerFromContext(ctx)
	timer := time.NewTimer(s.nextInterval())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if err := s.push(ctx, out); err != nil && ctx.Err() == nil {
			logger.Error(err, "error listing requests")
		}
		timer.Reset(s.nextInterval())
	}
}

// nextInterval is the interval plus a random share of it, up to the jitter
// factor.
func (s *Source[request]) nextInterval() time.Duration {
	if s.jitter <= 0 {
		return s.every
	}
	return s.every + time.Duration(s.random()*s.jitter*float64(s.every))
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
