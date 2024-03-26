package upgrade

import (
	"context"

	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// freightReconciler reconciles Freight resources to upgrade them from
// v0.4.0-compatible to v0.5.0-compatible.
type freightReconciler struct {
	client client.Client
}

// SetupFreightReconcilerWithManager initializes a freightReconciler and
// registers it with the provided Manager.
func SetupFreightReconcilerWithManager(mgr manager.Manager) error {
	notV050CompatiblePredicate, err := getNotV050CompatiblePredicate()
	if err != nil {
		return err
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Freight{}).
		WithEventFilter(ignoreDeletesPredicate()).
		WithEventFilter(notV050CompatiblePredicate).
		Build(&freightReconciler{
			client: mgr.GetClient(),
		})
	return err
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (f *freightReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"freight":   req.NamespacedName.Name,
	})
	logger.Debug("reconciling Freight")

	// Find the Freight
	freight, err := kargoapi.GetFreight(ctx, f.client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if freight == nil {
		// Ignore if not found. This can happen if the Freight was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil // Do not requeue
	}

	// None of these things should really happen
	if len(freight.OwnerReferences) == 0 {
		logger.Warning("skipping Freight with no OwnerReferences")
		return ctrl.Result{
			Requeue: false,
		}, nil
	}
	if len(freight.OwnerReferences) > 1 {
		logger.Warning("skipping Freight with multiple OwnerReferences")
		return ctrl.Result{
			Requeue: false,
		}, nil
	}
	if freight.OwnerReferences[0].APIVersion != kargoapi.GroupVersion.String() {
		logger.Warning("skipping Freight with non-Kargo OwnerReference")
		return ctrl.Result{
			Requeue: false,
		}, nil
	}
	if freight.OwnerReferences[0].Kind != "Warehouse" {
		logger.Warning("skipping Freight with non-Warehouse OwnerReference")
		return ctrl.Result{
			Requeue: false,
		}, nil
	}

	// Update the Freight to be v0.5.0-compatible
	freight.Warehouse = freight.OwnerReferences[0].Name
	freight.OwnerReferences = nil
	if err := f.client.Update(ctx, freight); err != nil {
		return ctrl.Result{}, err
	}

	// If we get to here, patch the Stage with the v0.5.0 compatibility label
	// so that we won't ever have to reconcile it again.
	if err := patchLabel(
		ctx,
		f.client,
		freight,
		v050CompatibilityLabelKey,
		kargoapi.LabelTrueValue,
	); err != nil {
		return ctrl.Result{}, err
	}

	logger.Debug("updated Freight for v0.5.0 compatibility")

	return ctrl.Result{
		Requeue: false,
	}, nil
}
