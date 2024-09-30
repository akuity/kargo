package upgrade

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// Old annotation key and new label key
const (
	oldAnnotationKey = "rbac.kargo.akuity.io/claim"
	newLabelKey      = "rbac.akuity.io/new-label"
)

// AppUpgradeReconciler Reconciles for App resources that need annotation migration
type AppUpgradeReconciler struct {
	client.Client
}

// SetupAppUpgradeReconcilerWithManager initializes a reconciler
func SetupAppUpgradeReconcilerWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Application{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return hasOldAnnotations(e.ObjectNew.GetAnnotations())
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return hasOldAnnotations(e.Object.GetAnnotations())
			},
			DeleteFunc: func(event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(event.GenericEvent) bool {
				return false
			},
		}).
		Complete(&AppUpgradeReconciler{
			Client: mgr.GetClient(),
		})
}

// Reconcile handles the reconciliation logic for upgrading applications
func (r *AppUpgradeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	app := &v1alpha1.Application{}
	logger := logging.LoggerFromContext(ctx)
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get App: %w", err)
	}

	if val, ok := app.Annotations[oldAnnotationKey]; ok {
		app.Labels[newLabelKey] = val 
		delete(app.Annotations, oldAnnotationKey)
	}

	if err := r.Update(ctx, app); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update App annotations and labels: %w", err)
	}
	logger.Info(
		"Successfully migrated annotation to label",
		"app", app.Name, "namespace", app.Namespace,
	)

	return ctrl.Result{}, nil
}

// Helper function to check if any of the old annotations are present
func hasOldAnnotations(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	_, hasOldAnnotation := annotations[oldAnnotationKey]
	return hasOldAnnotation
}
