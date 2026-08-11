// Package promotionsets contains the Community Edition PromotionSet reconciler.
package promotionsets

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	unsupportedReason  = "UnsupportedInCommunityEdition"
	unsupportedMessage = "PromotionSets are not supported in Community Edition"
)

// ReconcilerConfig represents configuration for the PromotionSet reconciler.
type ReconcilerConfig struct {
	IsDefaultController     bool   `envconfig:"IS_DEFAULT_CONTROLLER"`
	ShardName               string `envconfig:"SHARD_NAME"`
	MaxConcurrentReconciles int    `envconfig:"MAX_CONCURRENT_PROMOTION_SET_RECONCILES" default:"4"`
}

// Name returns the name of the PromotionSet controller, qualified by shard name
// when this controller is responsible for a specific shard.
func (c ReconcilerConfig) Name() string {
	const name = "promotion-set-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

// shardPredicate returns a predicate that narrows this reconciler's watch to
// the PromotionSets its shard is responsible for. A PromotionSet is labeled
// with the shard of the Stage that created it, so an unlabeled PromotionSet
// belongs to the default controller.
func (c ReconcilerConfig) shardPredicate() controller.ResponsibleFor[client.Object] {
	return controller.ResponsibleFor[client.Object]{
		IsDefaultController: c.IsDefaultController,
		ShardName:           c.ShardName,
	}
}

// ReconcilerConfigFromEnv returns a ReconcilerConfig populated from
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// SetupReconcilerWithManager initializes the Community Edition PromotionSet
// reconciler and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	cfg ReconcilerConfig,
	reconcileFn ReconcileFn,
) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.PromotionSet{}).
		// Without this, every controller in a sharded installation would
		// reconcile every PromotionSet.
		WithEventFilter(cfg.shardPredicate()).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Named(cfg.Name()).
		Complete(&reconciler{
			client:      mgr.GetClient(),
			reconcileFn: reconcileFn,
		}); err != nil {
		return fmt.Errorf("error building PromotionSet reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized PromotionSet reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
		"shard", cfg.ShardName,
		"isDefaultController", cfg.IsDefaultController,
	)
	return nil
}

// ReconcileFn reconciles a PromotionSet after it has been loaded.
type ReconcileFn func(
	ctx context.Context,
	kubeClient client.Client,
	promotionSet *kargoapi.PromotionSet,
) (ctrl.Result, error)

// reconciler delegates PromotionSet reconciliation to its configured function.
type reconciler struct {
	client      client.Client
	reconcileFn ReconcileFn
}

func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	promotionSet := &kargoapi.PromotionSet{}
	if err := r.client.Get(ctx, req.NamespacedName, promotionSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileFn(ctx, r.client, promotionSet)
}

// DefaultReconcile reports that PromotionSets are unsupported in Community
// Edition.
func DefaultReconcile(
	ctx context.Context,
	kubeClient client.Client,
	promotionSet *kargoapi.PromotionSet,
) (ctrl.Result, error) {
	if err := kubeclient.PatchStatus(ctx, kubeClient, promotionSet, func(status *kargoapi.PromotionSetStatus) {
		status.ObservedGeneration = promotionSet.Generation
		status.Phase = kargoapi.PromotionSetPhaseErrored
		if status.FinishedAt == nil {
			now := metav1.Now()
			status.FinishedAt = &now
		}
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             unsupportedReason,
			Message:            unsupportedMessage,
			ObservedGeneration: promotionSet.Generation,
		})
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating PromotionSet status: %w", err)
	}

	logging.LoggerFromContext(ctx).Debug("reported unsupported PromotionSet")
	return ctrl.Result{}, nil
}
