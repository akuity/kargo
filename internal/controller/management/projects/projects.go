package projects

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

// reconciler reconciles Project resources.
type reconciler struct {
	kargoClient client.Client
}

// SetupReconcilerWithManager initializes a reconciler for Project resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Project{}).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(kargoMgr.GetClient()))
}

func newReconciler(kargoClient client.Client) *reconciler {
	return &reconciler{
		kargoClient: kargoClient,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	context.Context,
	ctrl.Request,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
