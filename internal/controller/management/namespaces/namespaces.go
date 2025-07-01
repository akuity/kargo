package namespaces

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/logging"
)

type ReconcilerConfig struct {
	MaxConcurrentReconciles int `envconfig:"MAX_CONCURRENT_NAMESPACE_RECONCILES" default:"4"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

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

	removeFinalizerFn func(
		context.Context,
		client.Client,
		client.Object,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Namespace resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.Namespace{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any ACTUAL deletes. (We do care about
					// updates where DeletionTimestamp is non-nil, but that's not a
					// delete event.)
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Complete(newReconciler(kargoMgr.GetClient()))

	if err == nil {
		logging.LoggerFromContext(ctx).Info(
			"Initialized Namespace reconciler",
			"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
		)
	}

	return err
}

func newReconciler(kubeClient client.Client) *reconciler {
	r := &reconciler{
		client: kubeClient,
	}
	r.getNamespaceFn = r.client.Get
	r.deleteProjectFn = r.client.Delete
	r.removeFinalizerFn = api.RemoveFinalizer
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Namespace")

	// Find the Namespace
	ns := &corev1.Namespace{}
	if err := r.getNamespaceFn(ctx, req.NamespacedName, ns); err != nil {
		// Ignore if not found. This can happen if the Namespace was deleted after
		// the current reconciliation request was issued.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// We're only interested in deletes
	if ns.DeletionTimestamp == nil {
		return ctrl.Result{}, nil
	}
	logger.Debug("Namespace is being deleted")

	if !controllerutil.ContainsFinalizer(ns, kargoapi.FinalizerName) {
		return ctrl.Result{}, nil
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
		return ctrl.Result{}, fmt.Errorf("error deleting Project %q: %w", ns.Name, err)
	}
	if err := r.removeFinalizerFn(ctx, r.client, ns); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}
	logger.Debug("done reconciling Namespace")
	return ctrl.Result{}, nil
}
