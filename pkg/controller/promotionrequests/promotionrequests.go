// Package promotionrequests contains the PromotionRequest reconciler.
package promotionrequests

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
	enterpriseOnlyReason  = "EnterpriseOnlyFeature"
	enterpriseOnlyMessage = "PromotionRequests are a Kargo Enterprise-only feature"
)

// ReconcilerConfig represents configuration for the PromotionRequest reconciler.
type ReconcilerConfig struct {
	IsDefaultController     bool   `envconfig:"IS_DEFAULT_CONTROLLER"`
	ShardName               string `envconfig:"SHARD_NAME"`
	MaxConcurrentReconciles int    `envconfig:"MAX_CONCURRENT_PROMOTION_REQUEST_RECONCILES" default:"4"`
}

// Name returns the name of the PromotionRequest controller, qualified by shard name
// when this controller is responsible for a specific shard.
func (c ReconcilerConfig) Name() string {
	const name = "promotion-request-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

// shardPredicate returns a predicate that narrows this reconciler's watch to
// the PromotionRequests its shard is responsible for. A PromotionRequest is labeled
// with the shard of the Stage that created it, so an unlabeled PromotionRequest
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

// SetupReconcilerWithManager initializes the PromotionRequest reconciler and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.PromotionRequest{}).
		// Without this, every controller in a sharded installation would
		// reconcile every PromotionRequest.
		WithEventFilter(cfg.shardPredicate()).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Named(cfg.Name()).
		Complete(&reconciler{client: mgr.GetClient()}); err != nil {
		return fmt.Errorf("error building PromotionRequest reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized PromotionRequest reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
		"shard", cfg.ShardName,
		"isDefaultController", cfg.IsDefaultController,
	)
	return nil
}

// reconciler reports on every PromotionRequest it observes that fanning Freight
// out to Targets is a Kargo Enterprise-only feature, so that a user whose Stage
// has stopped promoting can find out why from the object it produced.
//
// There is deliberately no hook to swap this behavior out. A reconciler that
// fans a PromotionRequest out into Promotions must also watch the child
// Promotions that drive it forward, and a watch is not something a swappable
// Reconcile body can contribute; such a reconciler is its own controller.
type reconciler struct {
	client client.Client
}

func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	promotionRequest := &kargoapi.PromotionRequest{}
	if err := r.client.Get(ctx, req.NamespacedName, promotionRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := kubeclient.PatchStatus(ctx, r.client, promotionRequest, func(status *kargoapi.PromotionRequestStatus) {
		status.ObservedGeneration = promotionRequest.Generation
		status.Phase = kargoapi.PromotionRequestPhaseErrored
		if status.FinishedAt == nil {
			now := metav1.Now()
			status.FinishedAt = &now
		}
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             enterpriseOnlyReason,
			Message:            enterpriseOnlyMessage,
			ObservedGeneration: promotionRequest.Generation,
		})
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating PromotionRequest status: %w", err)
	}

	logging.LoggerFromContext(ctx).Debug(
		"reported PromotionRequest as an enterprise-only feature",
	)
	return ctrl.Result{}, nil
}
