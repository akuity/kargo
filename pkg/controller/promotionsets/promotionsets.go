// Package promotionsets contains the no-op PromotionSet reconciler.
package promotionsets

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/logging"
)

// ReconcilerConfig represents configuration for the PromotionSet reconciler.
type ReconcilerConfig struct {
	MaxConcurrentReconciles int `envconfig:"MAX_CONCURRENT_PROMOTION_SET_RECONCILES" default:"4"`
}

// ReconcilerConfigFromEnv returns a ReconcilerConfig populated from
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// SetupReconcilerWithManager initializes the no-op PromotionSet reconciler and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.PromotionSet{}).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Named("promotion-set-controller").
		Complete(&reconciler{client: mgr.GetClient()}); err != nil {
		return fmt.Errorf("error building PromotionSet reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized PromotionSet reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)
	return nil
}

// reconciler observes PromotionSets without modifying them. PromotionSet
// behavior is supplied by Kargo Enterprise.
type reconciler struct {
	client client.Client
}

func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	promotionSet := &kargoapi.PromotionSet{}
	if err := r.client.Get(ctx, req.NamespacedName, promotionSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logging.LoggerFromContext(ctx).Debug(
		"observed PromotionSet",
		"namespace", req.Namespace,
		"name", req.Name,
	)
	return ctrl.Result{}, nil
}
