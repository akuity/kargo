package controller

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// rateLimiterConfig holds work queue rate limiter settings. These mirror the
// controller-runtime defaults but can be overridden via environment variables
// to tune reconciler backoff without a code change.
type rateLimiterConfig struct {
	// BackoffBaseDelay is the initial delay for the per-item exponential
	// failure rate limiter.
	BackoffBaseDelay time.Duration `envconfig:"RECONCILER_BACKOFF_BASE_DELAY" default:"5ms"`
	// BackoffMaxDelay caps the per-item exponential failure rate limiter.
	// The controller-runtime default is 1000s; a burst of reconciliation
	// errors (e.g. "Stage health evaluated to Unknown" during an active
	// promotion) can drive the backoff up to that cap and stall the stage
	// for 15+ minutes. Lower this value to recover faster.
	BackoffMaxDelay time.Duration `envconfig:"RECONCILER_BACKOFF_MAX_DELAY" default:"1000s"`
}

func CommonOptions(maxConcurrentReconciles int) controller.Options {
	var cfg rateLimiterConfig
	envconfig.MustProcess("", &cfg)
	return controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
		RecoverPanic:            ptr.To(true),
		RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
			workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
				cfg.BackoffBaseDelay,
				cfg.BackoffMaxDelay,
			),
			&workqueue.TypedBucketRateLimiter[reconcile.Request]{
				Limiter: rate.NewLimiter(rate.Limit(10), 100),
			},
		),
	}
}
