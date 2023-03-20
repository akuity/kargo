package controller

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/config"
)

// promotionReconciler reconciles Promotion resources.
type promotionReconciler struct {
	client client.Client
	logger *log.Logger
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
		logger: logger,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (p *promotionReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only prevents requeueing
		// a Promotion if THIS reconciliation succeeds.
		RequeueAfter: 0,
	}

	logger := p.logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"promotion": req.NamespacedName.Name,
	})

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

	promo.Status, err = p.sync(ctx, promo)
	if err != nil {
		promo.Status.Error = err.Error()
		logger.Error(err)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		promo.Status.Error = ""
	}

	updateErr := p.client.Status().Update(ctx, promo)
	if updateErr != nil {
		logger.Errorf("error updating Promotion status: %s", updateErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

func (p *promotionReconciler) sync(
	ctx context.Context,
	promo *api.Promotion,
) (api.PromotionStatus, error) {
	status := *promo.Status.DeepCopy()

	// TODO: Implement sync

	return status, nil
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
			p.logger.WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"name":      namespacedName.Name,
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
