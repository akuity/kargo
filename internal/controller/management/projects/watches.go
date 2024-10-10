package projects

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// ServiceAccountEventHandler handles events related to ServiceAccounts.
type ServiceAccountEventHandler[T any] struct {
	kargoClient client.Client
	logger      *logging.Logger
}

// Create implements the Create event handler for ServiceAccounts.
// It enqueues a Project for reconciliation if the created ServiceAccount has the controller label.
func (h *ServiceAccountEventHandler[T]) Create(
	ctx context.Context,
	evt event.TypedCreateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	sa, ok := any(evt.Object).(*corev1.ServiceAccount)
	if !ok || sa == nil {
		h.logger.Error(nil, "Create event has no valid ServiceAccount object", "event", evt)
		return
	}

	if hasControllerLabel(sa) {
		h.enqueueProjectForReconciliation(ctx, sa, wq)
	}

}

// Update implements the Update event handler for ServiceAccounts.
func (h *ServiceAccountEventHandler[T]) Update(
	ctx context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {

	oldSA, okOld := any(evt.ObjectOld).(*corev1.ServiceAccount)
	if !okOld || oldSA == nil {
		h.logger.Error(
			nil, "Update event has no valid old ServiceAccount object",
			"event", evt,
		)
		return
	}

	newSA, okNew := any(evt.ObjectNew).(*corev1.ServiceAccount)
	if !okNew || newSA == nil {
		h.logger.Error(
			nil, "Update event has no valid new ServiceAccount object",
			"event", evt,
		)
		return
	}

	if hasControllerLabel(oldSA) && !hasControllerLabel(newSA) {
		// The label was removed or changed, hence, remove controller SA permissions.
		err := h.removeControllerPermissions(ctx, newSA)
		if err != nil {
			h.logger.Error(err, "Failed to remove RB for ServiceAccount", "serviceAccount", newSA.Name)
		} else {
			h.logger.Debug("Removed RB for ServiceAccount", "serviceAccount", newSA.Name)
		}
	} else if !hasControllerLabel(oldSA) && hasControllerLabel(newSA) {
		h.enqueueProjectForReconciliation(ctx, newSA, wq)
	}
}

// Delete implements the Delete event handler for ServiceAccounts.
// It removes the RoleBinding if the deleted ServiceAccount had the controller label.
func (h *ServiceAccountEventHandler[T]) Delete(
	ctx context.Context,
	evt event.TypedDeleteEvent[T],
	_ workqueue.TypedRateLimitingInterface[reconcile.Request],
) {

	sa, ok := any(evt.Object).(*corev1.ServiceAccount)
	if !ok || sa == nil {
		h.logger.Error(nil, "Delete event has no valid ServiceAccount object", "event", evt)
		return
	}

	if hasControllerLabel(sa) {
		err := h.removeControllerPermissions(ctx, sa)
		if err != nil {
			h.logger.Error(err, "Failed to remove RB for deleted ServiceAccount", "serviceAccount", sa.Name)
		} else {
			h.logger.Debug("Removed RoleBinding for deleted ServiceAccount", "serviceAccount", sa.Name)
		}
	}
}

func (h *ServiceAccountEventHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// enqueueProjectForReconciliation fetches the Project associated with the given ServiceAccount
// and enqueues it for reconciliation if it exists.
func (h *ServiceAccountEventHandler[T]) enqueueProjectForReconciliation(
	ctx context.Context,
	sa *corev1.ServiceAccount,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	project := &kargoapi.Project{}
	if err := h.kargoClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: sa.Namespace,
			Name:      sa.Namespace,
		},
		project,
	); err != nil {
		h.logger.Error(
			err,
			"Failed to find corresponding Project",
			"project", project.Name,
		)
		return
	}
	wq.Add(
		reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: project.Namespace,
				Name:      project.Name,
			},
		},
	)
	h.logger.Debug(
		"enqueued Project for reconciliation due to ServiceAccount creation",
		"namespace", project.Namespace,
		"project", project.Name,
	)
}

// hasControllerLabel checks if the given ServiceAccount has the "app.kubernetes.io/component"
// label set to "controller". Returns true if the label exists and is set to "controller", false otherwise.
func hasControllerLabel(sa *corev1.ServiceAccount) bool {
	if sa == nil {
		return false
	}
	labelValue, exists := sa.Labels["app.kubernetes.io/component"]
	return exists && labelValue == "controller"
}

// removeControllerPermissions removes the RoleBinding associated with the given ServiceAccount.
// It checks if the RoleBinding exists and deletes it if found.
func (h *ServiceAccountEventHandler[T]) removeControllerPermissions(
	ctx context.Context,
	sa *corev1.ServiceAccount,
) error {

	roleBindingName := fmt.Sprintf("%s-readonly-secrets", sa.Name)

	roleBinding := &rbacv1.RoleBinding{}
	err := h.kargoClient.Get(ctx, client.ObjectKey{Name: roleBindingName, Namespace: sa.Namespace}, roleBinding)
	if err != nil {
		if kubeerr.IsNotFound(err) {
			h.logger.Debug("RoleBinding not found, nothing to remove", "roleBinding", roleBindingName)
			return nil
		}
		return err
	}

	if err := h.kargoClient.Delete(ctx, roleBinding); err != nil {
		return err
	}

	h.logger.Debug("Deleted RoleBinding for ServiceAccount", "roleBinding", roleBindingName)
	return nil
}
