package upgrade

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// v0.4-compatible to v0.5-compatible.
type freightReconciler struct {
	client client.Client
}

// SetupFreightReconcilerWithManager initializes a freightReconciler and
// registers it with the provided Manager.
func SetupFreightReconcilerWithManager(mgr manager.Manager) error {
	notV050CompatiblePredicate, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{ // All Freight that do not have the v0.5 compatibility label
					Key:      kargoapi.V05CompatibilityLabelKey,
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Freight{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
			},
		).
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

	// Copy alias label into new alias field
	if alias, ok := freight.Labels[kargoapi.AliasLabelKey]; freight.Alias == "" && ok {
		if err = f.client.Patch(
			ctx,
			freight,
			client.RawPatch(
				types.MergePatchType,
				[]byte(fmt.Sprintf(`{"alias":"%s"}`, alias)),
			),
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Rescind formal ownership by the Warehouse and move the Warehouse name to
	// the new warehouse field
	if len(freight.OwnerReferences) == 1 &&
		freight.OwnerReferences[0].APIVersion == kargoapi.GroupVersion.String() &&
		freight.OwnerReferences[0].Kind == "Warehouse" {
		if err = f.client.Patch(
			ctx,
			freight,
			client.RawPatch(
				types.MergePatchType,
				[]byte(
					fmt.Sprintf(
						`{"warehouse":"%s","metadata":{"ownerReferences":null}}`,
						freight.OwnerReferences[0].Name,
					),
				),
			),
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	// If we get to here, patch the Stage with the v0.5 compatibility label
	// so that we won't ever have to reconcile it again.
	if err := kargoapi.AddV05CompatibilityLabel(ctx, f.client, freight); err != nil {
		return ctrl.Result{}, err
	}

	logger.Debug("updated Freight for v0.5 compatibility")

	return ctrl.Result{
		Requeue: false,
	}, nil
}
