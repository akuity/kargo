package upgrade

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// Old annotation keys
const (
	OldAnnotationKeySub    = "rbac.kargo.akuity.io/sub"
	OldAnnotationKeyEmail  = "rbac.kargo.akuity.io/email"
	OldAnnotationKeyGroups = "rbac.kargo.akuity.io/groups"
)

// ServiceAccountReconciler Reconciles for ServiceAccounts that need annotation migration
type ServiceAccountReconciler struct {
	client.Client
}

// SetupServiceAccountReconcilerWithManager initializes a reconciler
// that upgrades annotations on ServiceAccount resources for v0.9.x
// compatibility and registers that reconciler with the provided Manager.
func SetupServiceAccountReconcilerWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ServiceAccount{}).
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
		Complete(&ServiceAccountReconciler{
			Client: mgr.GetClient(),
		})
}

// Reconcile handles the reconciliation logic for migrating annotations
func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sa := &corev1.ServiceAccount{}
	logger := logging.LoggerFromContext(ctx)
	err := r.Get(ctx, req.NamespacedName, sa)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get ServiceAccount: %w", err)
	}

	annotations := sa.Annotations

	if val, ok := annotations[OldAnnotationKeySub]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("sub")] = val
		delete(annotations, OldAnnotationKeySub)
	}
	if val, ok := annotations[OldAnnotationKeyEmail]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("email")] = val
		delete(annotations, OldAnnotationKeyEmail)
	}
	if val, ok := annotations[OldAnnotationKeyGroups]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("groups")] = val
		delete(annotations, OldAnnotationKeyGroups)
	}

	sa.SetAnnotations(annotations)
	if err := r.Update(ctx, sa); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update ServiceAccount annotations: %w", err)
	}
	logger.Info("Successfully migrated ServiceAccount annotations", "ServiceAccount", req.NamespacedName)

	return ctrl.Result{}, nil
}

// Helper function to check if any of the old annotations are present
func hasOldAnnotations(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	_, hasSub := annotations[OldAnnotationKeySub]
	_, hasEmail := annotations[OldAnnotationKeyEmail]
	_, hasGroups := annotations[OldAnnotationKeyGroups]
	return hasSub || hasEmail || hasGroups
}
