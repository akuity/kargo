package serviceaccounts

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	Scheme *runtime.Scheme
	Logger *logging.Logger
}

// SetupWithManager sets up the reconciler with the manager
func (r *ServiceAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ServiceAccount{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return hasOldAnnotations(e.ObjectNew.GetAnnotations())
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return hasOldAnnotations(e.Object.GetAnnotations())
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return hasOldAnnotations(e.Object.GetAnnotations())
			},
		}).
		Complete(r)
}

// Reconcile handles the reconciliation logic for migrating annotations
func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, req.NamespacedName, sa)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		r.Logger.Error(err, "Failed to get ServiceAccount")
		return ctrl.Result{}, err
	}

	annotations := sa.GetAnnotations()
	if annotations == nil {
		return ctrl.Result{}, nil
	}

	updated := false
	if val, ok := annotations[OldAnnotationKeySub]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("sub")] = val
		delete(annotations, OldAnnotationKeySub)
		updated = true
	}
	if val, ok := annotations[OldAnnotationKeyEmail]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("email")] = val
		delete(annotations, OldAnnotationKeyEmail)
		updated = true
	}
	if val, ok := annotations[OldAnnotationKeyGroups]; ok {
		annotations[rbacapi.AnnotationKeyOIDCClaim("groups")] = val
		delete(annotations, OldAnnotationKeyGroups)
		updated = true
	}

	if updated {
		sa.SetAnnotations(annotations)
		if err := r.Update(ctx, sa); err != nil {
			r.Logger.Error(err, "Failed to update ServiceAccount annotations")
			return ctrl.Result{}, err
		}
		r.Logger.Info("Successfully migrated ServiceAccount annotations", "ServiceAccount", req.NamespacedName)
	}

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

func SetupServiceAccountReconcilerWithManager(mgr ctrl.Manager) error {
	return (&ServiceAccountReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: logging.NewLogger(logging.InfoLevel),
	}).SetupWithManager(mgr)
}
