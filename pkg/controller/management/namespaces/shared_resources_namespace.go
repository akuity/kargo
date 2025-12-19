package namespaces

import (
	"context"
	"fmt"
	"slices"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/logging"
)

type SharedResourcesReconcilerConfig struct {
	GlobalCredentialsNamespaces []string `envconfig:"GLOBAL_CREDENTIALS_NAMESPACES" default:""`
	SharedResourcesNamespace    string   `envconfig:"SHARED_RESOURCES_NAMESPACE" default:"kargo-shared-resources"`
	MaxConcurrentReconciles     int      `envconfig:"MAX_CONCURRENT_NAMESPACE_RECONCILES" default:"4"`
}

func SharedResourcesReconcilerConfigFromEnv() SharedResourcesReconcilerConfig {
	cfg := SharedResourcesReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Namespace resources.
type sharedResourcesReconciler struct {
	client client.Client
	config SharedResourcesReconcilerConfig
}

// SetupSharedResourcesReconcilerWithManager initializes a reconciler for shared resources
// and registers it with the provided Manager.
func SetupSharedResourcesReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg SharedResourcesReconcilerConfig,
) error {
	err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.Namespace{}).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Complete(newSharedResourcesReconciler(kargoMgr.GetClient(), cfg))

	if err == nil {
		logging.LoggerFromContext(ctx).Info(
			"Initialized Namespace reconciler",
			"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
		)
	}

	return err
}

func newSharedResourcesReconciler(
	kubeClient client.Client,
	cfg SharedResourcesReconcilerConfig,
) *sharedResourcesReconciler {
	return &sharedResourcesReconciler{
		client: kubeClient,
		config: cfg,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *sharedResourcesReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", req.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Namespace")

	if req.Name == r.config.SharedResourcesNamespace {
		logger.Debug("Namespace is the shared resources namespace; skipping migration")
		return ctrl.Result{}, nil
	}

	if !slices.Contains(r.config.GlobalCredentialsNamespaces, req.Name) {
		logger.Debug("Namespace is not a global credentials namespace; skipping migration")
		return ctrl.Result{}, nil
	}

	ns := new(corev1.Namespace)
	if err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, ns); err != nil {
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

	// Migrate resources from GlobalCredentialsNamespaces to SharedResourcesNamespace
	if err := r.migrateResources(ctx, ns.Name); err != nil {
		return ctrl.Result{}, fmt.Errorf("error migrating resources: %w", err)
	}

	// Ignore not found errors to keep this idempotent.
	// if err := client.IgnoreNotFound(
	// 	r.deleteProjectFn(
	// 		ctx,
	// 		&kargoapi.Project{
	// 			ObjectMeta: metav1.ObjectMeta{
	// 				Name: ns.Name,
	// 			},
	// 		},
	// 	),
	// ); err != nil {
	// 	return ctrl.Result{}, fmt.Errorf("error deleting Project %q: %w", ns.Name, err)
	// }
	// if err := r.removeFinalizerFn(ctx, r.client, ns); err != nil {
	// 	return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	// }
	logger.Debug("done reconciling Namespace")
	return ctrl.Result{}, nil
}

func (r *sharedResourcesReconciler) migrateResources(ctx context.Context, namespace string) error {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("migrating resources", "namespace", namespace)

	// List all resources in the GlobalCredentialsNamespaces
	for _, globalNamespace := range r.config.GlobalCredentialsNamespaces {
		resourceList := &corev1.SecretList{}
		if err := r.client.List(ctx, resourceList, client.InNamespace(globalNamespace)); err != nil {
			return fmt.Errorf("error listing resources in namespace %q: %w", globalNamespace, err)
		}

		// Migrate each resource to the SharedResourcesNamespace
		for _, resource := range resourceList.Items {
			logger.Debug("migrating resource", "resource", resource.Name, "from", globalNamespace, "to", r.config.SharedResourcesNamespace)

			// Update the resource's namespace
			resource.Namespace = r.config.SharedResourcesNamespace
			resource.ResourceVersion = "" // Clear the resource version to allow re-creation

			// Create the resource in the SharedResourcesNamespace
			if err := r.client.Create(ctx, &resource); err != nil {
				return fmt.Errorf("error migrating resource %q: %w", resource.Name, err)
			}
		}
	}

	logger.Debug("finished migrating resources", "namespace", namespace)
	return nil
}
