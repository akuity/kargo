package kube

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Option configures how a controller runs under a manager.
type Option func(*runnable)

// WithoutLeaderElection runs the controller in every replica rather than
// only in the one that holds the lease. Its Reconciler must then be
// idempotent, since every replica also resyncs, and its event sources
// should share events across replicas; see nats.Source.QueueGroup.
func WithoutLeaderElection() Option {
	return func(r *runnable) { r.leaderElection = false }
}

// Runnable adapts a controller, or anything else with a Start, for a
// controller-runtime manager:
//
//	mgr.Add(kube.Runnable(c))
//
// The manager starts it after winning leader election unless
// WithoutLeaderElection is given. Leader election is the manager's concept,
// which is why it is decided here and not on the controller.
func Runnable(c interface{ Start(context.Context) error }, opts ...Option) manager.Runnable {
	r := &runnable{start: c.Start, leaderElection: true}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type runnable struct {
	start          func(context.Context) error
	leaderElection bool
}

var _ manager.LeaderElectionRunnable = (*runnable)(nil)

// Start implements manager.Runnable.
func (r *runnable) Start(ctx context.Context) error { return r.start(ctx) }

// NeedLeaderElection implements manager.LeaderElectionRunnable.
func (r *runnable) NeedLeaderElection() bool { return r.leaderElection }
