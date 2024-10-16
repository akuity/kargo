package serviceaccounts

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

const (
	componentLabelKey    = "app.kubernetes.io/component"
	controllerLabelValue = "controller"
)

type ReconcilerConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" required:"true"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles ServiceAccount resources
type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client

	getServiceAccountFn func(
		context.Context,
		types.NamespacedName,
		client.Object,
		...client.GetOption,
	) error

	updateServiceAccountFn func(
		context.Context,
		client.Object,
		...client.UpdateOption,
	) error

	createRoleBindingFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	updateRoleBindingFn func(
		context.Context,
		client.Object,
		...client.UpdateOption,
	) error

	deleteRoleBindingFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error

	listProjectFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	ensureControllerPermissionsFn func(context.Context, types.NamespacedName) error
	removeControllerPermissionsFn func(context.Context, types.NamespacedName) error
}

// SetupReconcilerWithManager initializes a reconciler for ServiceAccount
// resources and registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager, cfg ReconcilerConfig) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.ServiceAccount{}).
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(event.CreateEvent) bool {
					// Allow creation events to be handled for all ServiceAccounts
					// This ensures any labeled or unlabeled SA gets proper
					// reconciliation, including on restarts
					return true
				},
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldSA, _ := e.ObjectOld.(*corev1.ServiceAccount)
					newSA, _ := e.ObjectNew.(*corev1.ServiceAccount)
					return hasControllerLabel(oldSA) != hasControllerLabel(newSA) ||
						hasControllerLabel(newSA) && newSA.DeletionTimestamp != nil
				},
				GenericFunc: func(event.GenericEvent) bool {
					return false
				},
			},
		).
		Complete(newReconciler(kargoMgr.GetClient(), cfg))
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	r := &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
	r.getServiceAccountFn = r.client.Get
	r.updateServiceAccountFn = r.client.Update
	r.createRoleBindingFn = r.client.Create
	r.updateRoleBindingFn = r.client.Update
	r.deleteRoleBindingFn = r.client.Delete
	r.listProjectFn = r.client.List
	r.ensureControllerPermissionsFn = r.ensureControllerPermissions
	r.removeControllerPermissionsFn = r.removeControllerPermissions
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceAccount", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling ServiceAccount")

	sa := &corev1.ServiceAccount{}
	if err := r.getServiceAccountFn(ctx, req.NamespacedName, sa); err != nil {
		if kubeerr.IsNotFound(err) {
			logger.Debug("ServiceAccount not found, deleting RoleBindings for all Projects")
			// If not found it means the ServiceAccount with the
			// controller label was deleted, hence remove Controller
			// Permissions for the ServiceAccount.
			return ctrl.Result{}, r.removeControllerPermissionsFn(ctx, req.NamespacedName)
		}
		logger.Error(err, "Failed to get ServiceAccount")
		return ctrl.Result{}, err
	}

	// Handle ServiceAccount deletion or updates where the
	// controller label has been removed from the ServiceAccount.
	// This indicates that the ServiceAccount is no longer managed by the controller,
	// and we need to clean up any associated RoleBindings.
	if sa.DeletionTimestamp != nil || !hasControllerLabel(sa) {
		logger.Debug("Deleting RoleBindings for ServiceAccount", "serviceaccount", sa.Name)
		if err := r.removeControllerPermissionsFn(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(sa, kargoapi.FinalizerName)
		if err := r.updateServiceAccountFn(ctx, sa); err != nil {
			return ctrl.Result{}, err
		}
		logger.Debug("Removed finalizer from ServiceAccount", "serviceaccount", sa.Name)
		return ctrl.Result{}, nil
	}

	// If we get to here, we had a not found error and we can proceed with
	// creating the RoleBindings.

	// Add a finalizer to the ServiceAccount to prevent premature deletion
	// before RoleBindings are removed. The AddFinalizer function:
	// - Returns false if the finalizer is already present, avoiding redundant logs.
	// - Adds the finalizer if not present, ensuring proper cleanup.
	if controllerutil.AddFinalizer(sa, kargoapi.FinalizerName) {
		if err := r.updateServiceAccountFn(ctx, sa); err != nil {
			return ctrl.Result{}, err
		}
		logger.Debug("Added finalizer to ServiceAccount", "serviceaccount", sa.Name)
		return ctrl.Result{}, nil
	}

	logger.Debug("Creating RoleBindings for ServiceAccount",
		"serviceaccount", sa.Name,
	)
	return ctrl.Result{}, r.ensureControllerPermissionsFn(ctx, req.NamespacedName)
}

// ensureControllerPermissions grants the controller ServiceAccount necessary access to all Projects
// by creating RoleBindings in each Project's namespace. This function ensures that the
// controller has the necessary permissions to manage resources across all Projects.
func (r *reconciler) ensureControllerPermissions(ctx context.Context, sa types.NamespacedName) error {

	roleBindingName := fmt.Sprintf("%s-readonly-secrets", sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceaccount", sa.Name,
		"roleBinding", roleBindingName,
	)
	logger.Debug("starting to create RoleBindings for all Projects")

	projectList := &kargoapi.ProjectList{}
	if err := r.listProjectFn(ctx, projectList); err != nil {
		return fmt.Errorf("error listing Projects: %w", err)
	}

	// Loop through each Project to create or update the corresponding RoleBinding.
	for _, project := range projectList.Items {

		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: project.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "kargo-controller-secrets-readonly",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      sa.Name,
					Namespace: r.cfg.KargoNamespace,
				},
			},
		}

		// Try to create RoleBinding directly, update if it already exists.
		if err := r.createRoleBindingFn(ctx, roleBinding); err != nil {
			if !kubeerr.IsAlreadyExists(err) {
				return fmt.Errorf("error creating RoleBinding %q for ServiceAccount %q in Project namespace %q: %w",
					roleBinding.Name, sa.Name, project.Name, err)
			}
			if updateErr := r.updateRoleBindingFn(ctx, roleBinding); updateErr != nil {
				return fmt.Errorf("error updating existing RoleBinding %q in Project namespace %q: %w",
					roleBinding.Name, project.Name, updateErr)
			}
			logger.Debug("Updated existing RoleBinding for ServiceAccount", "roleBinding",
				roleBindingName, "namespace", project.Name)
		} else {
			logger.Debug("Created RoleBinding for ServiceAccount", "roleBinding", roleBindingName, "namespace", project.Name)
		}
	}
	logger.Debug("Completed creating RoleBindings for all Projects")
	return nil
}

// removeControllerPermissions removes RoleBindings for the specified ServiceAccount
// from all Projects. This function is called when the ServiceAccount is no longer
// managed by the controller, ensuring that permissions are cleaned up to prevent
// unauthorized access to resources.
func (r *reconciler) removeControllerPermissions(ctx context.Context, sa types.NamespacedName) error {

	roleBindingName := fmt.Sprintf("%s-readonly-secrets", sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceaccount", sa.Name,
		"roleBinding", roleBindingName,
	)
	logger.Debug("Starting to delete RoleBindings for all Projects")

	projectList := &kargoapi.ProjectList{}
	if err := r.listProjectFn(ctx, projectList); err != nil {
		return fmt.Errorf("error listing Projects: %w", err)
	}

	for _, project := range projectList.Items {

		projectLogger := logger.WithValues("namespace", project.Name, "roleBinding", roleBindingName)

		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: project.Name,
			},
		}

		if err := r.deleteRoleBindingFn(ctx, roleBinding); err != nil {
			if kubeerr.IsNotFound(err) {
				projectLogger.Debug("RoleBinding not found")
				continue // RoleBinding does not exist; moving to the next Project.
			}

			// Return the error to trigger a requeue and stop further cleanup until the issue is resolved.
			return fmt.Errorf("error deleting RoleBinding %q in Project namespace %q: %w",
				roleBindingName, project.Name, err)
		}

		projectLogger.Debug("Deleted RoleBinding")
	}
	logger.Debug("Completed deletion of RoleBindings for all Projects")
	return nil
}

func hasControllerLabel(sa *corev1.ServiceAccount) bool {
	return sa.GetLabels()[componentLabelKey] == controllerLabelValue
}
