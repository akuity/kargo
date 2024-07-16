package upgrade

import (
	"context"
	"errors"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// freightReconciler reconciles Freight resources to upgrade them from
// v0.7.x to v0.8.x.
type freightReconciler struct {
	client client.Client
}

// SetupFreightReconcilerWithManager initializes a freightReconciler and
// registers it with the provided Manager.
func SetupFreightReconcilerWithManager(mgr manager.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Freight{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
			},
		).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			freight, ok := object.(*kargoapi.Freight)
			if !ok {
				return false
			}

			// Ignore Freight that are already v0.8 compatible.
			if _, ok = freight.Labels[kargoapi.V08CompatibilityLabelKey]; ok {
				return false
			}

			// Ignore Freight that have an origin set.
			if freight.Origin.Kind != "" && freight.Origin.Name != "" {
				return false
			}

			// Ignore Freight that are missing the Warehouse field.
			return freight.Warehouse != "" // nolint: staticcheck
		})).
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
	logger := logging.LoggerFromContext(ctx).WithValues(
		"namespace", req.NamespacedName.Namespace,
		"freight", req.NamespacedName.Name,
	)
	logger.Debug("reconciling Freight")

	// Check if the freights.kargo.akuity.io CRD has the origin field.
	// If it does not, we need to wait for the CRD to be updated.
	var freightCRD extv1.CustomResourceDefinition
	if err := f.client.Get(
		ctx,
		types.NamespacedName{
			Name: "freights.kargo.akuity.io",
		},
		&freightCRD,
	); err != nil {
		return ctrl.Result{}, err
	}
	if _, hasOriginField := freightCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.
		Properties["origin"]; !hasOriginField {
		return ctrl.Result{}, errors.New(
			"freights.kargo.akuity.io does not have an origin field: waiting for update",
		)
	}

	// Find the Freight.
	freight, err := kargoapi.GetFreight(ctx, f.client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if freight == nil {
		// Ignore if not found. This can happen if the Freight was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil // Do not requeue
	}

	// Check if the Freight is already v0.8.x compatible.
	if freight.Origin.Kind != "" && freight.Origin.Name != "" {
		logger.Debug("Freight is already v0.8 compatible")
		return ctrl.Result{}, nil
	}

	// If the Warehouse field is not set, we can't migrate the Freight.
	if freight.Warehouse == "" { // nolint: staticcheck
		logger.Debug("Freight is missing Warehouse")
		return ctrl.Result{}, nil
	}

	// Migrate the Warehouse field to the Origin field.
	freight.Origin.Kind = kargoapi.FreightOriginKindWarehouse
	freight.Origin.Name = freight.Warehouse // nolint: staticcheck
	freight.Warehouse = ""                  // nolint: staticcheck

	if err = f.client.Update(ctx, freight); err != nil {
		return ctrl.Result{}, err
	}

	// If the update was successful, add the v0.8 compatibility label.
	if err = kargoapi.AddV08CompatibilityLabel(ctx, f.client, freight); err != nil {
		return ctrl.Result{}, err
	}

	logger.Debug("updated Freight for v0.8 compatibility")
	return ctrl.Result{}, nil
}
