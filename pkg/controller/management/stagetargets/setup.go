package stagetargets

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

const (
	// pollInterval is how often the change feed asks the database which
	// Projects changed. It is one grouped query over Stages and Targets, and
	// bounds how long a new Target or a changed selector waits to be settled.
	pollInterval = 5 * time.Second
	// resyncInterval is how often every Project is settled regardless.
	resyncInterval = 10 * time.Minute
	resyncJitter   = 0.2
	numWorkers     = 4
)

// SetupWithManager registers the controller that keeps stage_targets in step
// with the manager. It runs only in the replica that holds the lease. The
// caller owns the database pool behind queries.
func SetupWithManager(ctx context.Context, mgr manager.Manager, queries *database.Queries) error {
	feed := newChangeFeed(queries)
	r := &stageTargetsReconciler{store: queries}
	c, err := reconciler.New[string]("stage-targets").
		Watch(list.New(feed.changed).Every(pollInterval)).
		Watch(list.New(feed.all).Every(resyncInterval).Jitter(resyncJitter)).
		Workers(numWorkers).
		Func(r.reconcile)
	if err != nil {
		return fmt.Errorf("error building stage targets controller: %w", err)
	}
	if err = mgr.Add(kube.Runnable(c)); err != nil {
		return fmt.Errorf("error adding stage targets controller to manager: %w", err)
	}
	logging.LoggerFromContext(ctx).Info("Initialized stage targets controller")
	return nil
}
