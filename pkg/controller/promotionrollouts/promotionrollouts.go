// Package promotionrollouts contains the PromotionRollout controller.
//
// PromotionRollouts live in the database, so the controller is built on
// pkg/reconciler rather than controller-runtime. In Kargo OSS a rollout is
// never fanned out, so there is nothing to do for any one rollout but report
// that: on an interval, the controller ends every rollout that has not
// started as Errored, in one statement, with the explanation that fanning
// Freight out to Targets is a Kargo Enterprise-only feature, so that a user
// whose Stage has stopped promoting can find out why from the rollout the
// Stage produced.
//
// There is deliberately no hook to swap this behavior out. A reconciler that
// fans a rollout out into Promotions must also watch the child Promotions
// that drive it forward, and a watch is not something a swappable reconcile
// body can contribute; such a reconciler is its own controller, built from
// the same store and the same engine, and runs instead of this one.
package promotionrollouts

import (
	"context"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.opentelemetry.io/otel"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

const (
	controllerName = "promotion-rollouts"

	enterpriseOnlyReason  = "EnterpriseOnlyFeature"
	enterpriseOnlyMessage = "PromotionRollouts are a Kargo Enterprise-only feature"

	// reconcileTimeout bounds one sweep, so a blocked statement or an
	// unreachable database fails it for retry with backoff instead of
	// holding the worker indefinitely.
	reconcileTimeout = 5 * time.Second
	// resyncJitter spreads the sweeps of replicas that started together.
	resyncJitter = 0.2
)

// tracer is the instrumentation scope under which this package's spans are
// recorded.
var tracer = otel.Tracer("github.com/akuity/kargo/pkg/controller/promotionrollouts")

// ReconcilerConfig represents configuration for the PromotionRollout
// reconciler.
type ReconcilerConfig struct {
	// ResyncInterval is how often rollouts that have not started are ended.
	// There is no change feed, so this is also how soon a new rollout is.
	ResyncInterval time.Duration `envconfig:"PROMOTION_ROLLOUT_RESYNC_INTERVAL" default:"1m"`
}

// ReconcilerConfigFromEnv returns a ReconcilerConfig populated from
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Store is the slice of promotionrolloutstore.Store the controller uses.
type Store interface {
	ErrorPending(ctx context.Context, reason, message string) (int64, error)
}

// sweep is the controller's only request. Every push asks for the same
// thing -- end whatever has not started -- so the queue collapses pushes
// that arrive while one sweep runs into the next.
type sweep struct{}

// sweeps is the list function behind the controller's source: one sweep,
// whenever it is asked.
func sweeps(context.Context) ([]sweep, error) {
	return []sweep{{}}, nil
}

// SetupWithManager builds the PromotionRollout controller and adds it to
// the manager, which starts it once it holds the leader lease. The caller
// owns the database behind the store.
func SetupWithManager(
	ctx context.Context,
	mgr manager.Manager,
	cfg ReconcilerConfig,
	store Store,
) error {
	c, err := reconciler.New[sweep](controllerName).
		Watch(list.New(sweeps).Every(cfg.ResyncInterval).Jitter(resyncJitter)).
		Func(newReconciler(store))
	if err != nil {
		return fmt.Errorf("error building PromotionRollout reconciler: %w", err)
	}
	if err = mgr.Add(kube.Runnable(c)); err != nil {
		return fmt.Errorf("error adding PromotionRollout reconciler to manager: %w", err)
	}
	logging.LoggerFromContext(ctx).Info(
		"Initialized PromotionRollout reconciler",
		"resyncInterval", cfg.ResyncInterval,
	)
	return nil
}

// newReconciler returns the reconcile function: one sweep, which ends every
// rollout that has not started as an Enterprise-only feature. A rollout
// that has ended is left alone, so a sweep can run any number of times.
func newReconciler(store Store) reconciler.Func[sweep] {
	return func(ctx context.Context, _ sweep) (reconciler.Result, error) {
		ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
		defer cancel()
		ctx, span := tracer.Start(ctx, "Sweep PromotionRollouts")
		defer span.End()

		ended, err := store.ErrorPending(ctx, enterpriseOnlyReason, enterpriseOnlyMessage)
		if err != nil {
			return reconciler.Result{}, err
		}
		if ended > 0 {
			logging.LoggerFromContext(ctx).Debug(
				"reported PromotionRollouts as an enterprise-only feature",
				"count", ended,
			)
		}
		return reconciler.Result{}, nil
	}
}
