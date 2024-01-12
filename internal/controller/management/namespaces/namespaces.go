package namespaces

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Namespace resources.
type reconciler struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	getNamespaceFn func(
		context.Context,
		types.NamespacedName,
		client.Object,
		...client.GetOption,
	) error

	deleteProjectFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error

	updateNamespaceFn func(
		context.Context,
		client.Object,
		...client.UpdateOption,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Namespace resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.Namespace{}).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(kargoMgr.GetClient()))
}

func newReconciler(kubeClient client.Client) *reconciler {
	r := &reconciler{
		client: kubeClient,
	}
	r.getNamespaceFn = r.client.Get
	r.deleteProjectFn = r.client.Delete
	r.updateNamespaceFn = r.client.Update
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only prevents requeueing
		// a Namespace if THIS reconciliation succeeds.
		RequeueAfter: 0,
	}

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Namespace")

	// Find the Namespace
	ns := &corev1.Namespace{}
	if err := r.getNamespaceFn(ctx, req.NamespacedName, ns); err != nil {
		if apierrors.IsNotFound(err) {
			// Ignore if not found. This can happen if the Namespace was deleted after
			// the current reconciliation request was issued.
			return result, nil
		}
		return result, err
	}

	// We're only interested in deletes
	if ns.DeletionTimestamp == nil {
		return result, nil
	}
	logger.Debug("Namespace is being deleted")

	if !controllerutil.ContainsFinalizer(ns, kargoapi.FinalizerName) {
		return result, nil
	}
	logger.Debug("Namespace needs finalizing")

	// Ignore not found errors to keep this idempotent.
	if err := client.IgnoreNotFound(
		r.deleteProjectFn(
			ctx,
			&kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns.Name,
				},
			},
		),
	); err != nil {
		return result, errors.Wrapf(err, "error deleting Project %q", ns.Name)
	}
	if controllerutil.RemoveFinalizer(ns, kargoapi.FinalizerName) {
		if err := r.updateNamespaceFn(ctx, ns); err != nil {
			return result, errors.Wrap(err, "error removing finalizer")
		}
	}
	logger.Debug("done reconciling Namespace")
	return result, nil
}
