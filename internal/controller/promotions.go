package controller

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/logging"
)

// promotionReconciler reconciles Promotion resources.
type promotionReconciler struct {
	client client.Client
}

// SetupPromotionReconcilerWithManager initializes a reconciler for
// Promotion resources and registers it with the provided Manager.
func SetupPromotionReconcilerWithManager(
	ctx context.Context,
	config config.ControllerConfig,
	mgr manager.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Promotion{}).
		WithEventFilter(predicate.Funcs{}).
		Complete(
			newPromotionReconciler(config, mgr.GetClient()),
		)
}

func newPromotionReconciler(
	config config.ControllerConfig,
	client client.Client,
) *promotionReconciler {
	logger := log.New()
	logger.SetLevel(config.LogLevel)
	return &promotionReconciler{
		client: client,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (p *promotionReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{}

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"promotion": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Promotion")

	// Find the Promotion
	promo, err := p.getPromo(ctx, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if promo == nil {
		// Ignore if not found. This can happen if the Promotion was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	promo.Status = p.sync(ctx, promo)
	if promo.Status.Error != "" {
		logger.Error(promo.Status.Error)
	}
	p.updateStatus(ctx, promo)

	// TODO: Make RequeueAfter configurable (via API, probably)
	// TODO: Or consider using a progressive backoff here when there has been an
	// error.
	return ctrl.Result{RequeueAfter: time.Minute}, err
}

func (p *promotionReconciler) sync(
	ctx context.Context,
	promo *api.Promotion,
) api.PromotionStatus {
	status := *promo.Status.DeepCopy()

	// TODO: Implement sync

	return status
}

// getPromo returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (p *promotionReconciler) getPromo(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*api.Promotion, error) {
	promo := api.Promotion{}
	if err := p.client.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			logging.LoggerFromContext(ctx).WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"promotion": namespacedName.Name,
			}).Warn("Promotion not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Promotion %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &promo, nil
}

// updateStatus updates the status subresource of the provided Promotion.
func (p *promotionReconciler) updateStatus(
	ctx context.Context,
	promo *api.Promotion,
) {
	if err := p.client.Status().Update(ctx, promo); err != nil {
		logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"namespace": promo.Namespace,
			"promotion": promo.Name,
		}).Errorf("error updating Promotion status: %s", err)
	}
}
