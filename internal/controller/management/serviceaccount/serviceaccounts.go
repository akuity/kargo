package upgrade

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

const (
	ComponentLabelKey = "app.kubernetes.io/component"
	ControllerLabelValue = "controller"
)

type ReconcilerConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" required:"true"`
}

// ServiceAccountReconciler Reconciles for ServiceAccounts
type ServiceAccountReconciler struct {
	cfg ReconcilerConfig
	client.Client
}

// SetupReconcilerWithManager initializes a reconciler
// and registers that reconciler with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.ServiceAccount{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return hasControllerLabel(e.Object.(*corev1.ServiceAccount))
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return hasControllerLabel(e.Object.(*corev1.ServiceAccount))
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldSA := e.ObjectOld.(*corev1.ServiceAccount)
				newSA := e.ObjectNew.(*corev1.ServiceAccount)
				return hasControllerLabel(oldSA) != hasControllerLabel(newSA)
			},
			GenericFunc: func(event.GenericEvent) bool {
				return false
			},
		}).
		Complete(&ServiceAccountReconciler{
			Client:         kargoMgr.GetClient(),
		})
}

// Reconcile handles the reconciliation logic for ServiceAccounts
func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceAccount", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("Reconciling ServiceAccounts")

	// Fetch the ServiceAccount
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, req.NamespacedName, sa); err != nil {
		if kubeerr.IsNotFound(err) {
			logger.Debug("ServiceAccount not found, deleting role bindings for all projects")
			// If not found it means the ServiceAccount with the 
			// controller label was deleted, hence remove Controller
			// Permissions for the ServiceAccount.
			return ctrl.Result{}, r.deleteRoleBindingsForAllProjects(ctx, sa)
		}
		logger.Error(err, "Failed to get ServiceAccount")
		return ctrl.Result{}, err
	}

	if controllerutil.AddFinalizer(sa, kargoapi.FinalizerName) {
		if err := r.Update(ctx, sa); err != nil {
			return ctrl.Result{}, err
		}
		logger.Debug("Added finalizer to ServiceAccount", "serviceaccount", sa.Name)
		return ctrl.Result{}, nil
	}

	// Handle deletion logic
	if sa.DeletionTimestamp != nil {
		logger.Debug("Deleting role bindings for ServiceAccount", "serviceaccount", sa.Name)
		if err := r.deleteRoleBindingsForAllProjects(ctx, sa); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(sa, kargoapi.FinalizerName)
		if err := r.Update(ctx, sa); err != nil {
			return ctrl.Result{}, err
		}
		logger.Debug("Removed finalizer from ServiceAccount", "serviceaccount", sa.Name)
		return ctrl.Result{}, nil
	}

	// Handle creation and update cases
	if !hasControllerLabel(sa) {
		logger.Debug("Deleting role bindings for ServiceAccount",
			"serviceaccount", sa.Name,
		)
		return ctrl.Result{}, r.deleteRoleBindingsForAllProjects(ctx, sa)
	}
	logger.Debug("Creating role bindings for ServiceAccount",
		"serviceaccount", sa.Name,
	)
	return ctrl.Result{}, r.createRoleBindingForAllProjects(ctx, sa)
}

// createRoleBindingForAllProjects creates RoleBindings for all projects
func (r *ServiceAccountReconciler) createRoleBindingForAllProjects(ctx context.Context, sa *corev1.ServiceAccount) error {
	
	roleBindingName := fmt.Sprintf("%s-readonly-secrets", sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceaccount", sa.Name,
		"roleBinding", roleBindingName,
	)
	logger.Info("Starting to create RoleBindings for all projects")

	// Get all projects (namespaces)
	projectList := &kargoapi.ProjectList{}
	if err := r.Client.List(ctx, projectList); err != nil {
		return fmt.Errorf("error listing projects: %w", err)
	}

	logger.Debug("Fetched project list", "projectCount", len(projectList.Items))

	// Iterate over each project and create RoleBinding
	for _, project := range projectList.Items {

		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: project.Name, // Project name is the namespace name
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

		// Create or update the RoleBinding		
		if err := r.Client.Get(ctx, client.ObjectKey{Name: roleBindingName, Namespace: project.Name}, roleBinding); err != nil {
			if kubeerr.IsNotFound(err) {
				// Create RoleBinding
				if err := r.Client.Create(ctx, roleBinding); err != nil {
					return fmt.Errorf("error creating RoleBinding %q for ServiceAccount %q in project namespace %q: %w",
						roleBinding.Name, sa.Name, project.Name, err)
				}
				logger.Info("Created RoleBinding for ServiceAccount", "roleBinding", roleBindingName, "namespace", project.Name)
			} else {
				return fmt.Errorf("error retrieving RoleBinding %q in namespace %q: %w", roleBindingName, project.Name, err)
			}
		} else {
			// Update RoleBinding
			if err := r.Client.Update(ctx, roleBinding); err != nil {
				return fmt.Errorf("error updating RoleBinding %q in project namespace %q: %w", roleBinding.Name, project.Name, err)
			}
			logger.Debug("Updated RoleBinding for ServiceAccount", "roleBinding", roleBindingName, "namespace", project.Name)
		}
	}
	logger.Info("Completed creating RoleBindings for all projects")
	return nil
}

// deleteRoleBindingsForAllProjects deletes RoleBindings for all projects
func (r *ServiceAccountReconciler) deleteRoleBindingsForAllProjects(ctx context.Context, sa *corev1.ServiceAccount) error {

	roleBindingName := fmt.Sprintf("%s-readonly-secrets", sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceaccount", sa.Name,
		"roleBinding", roleBindingName,
	)
	logger.Info("Starting to delete RoleBindings for all projects")

	projectList := &kargoapi.ProjectList{}
	if err := r.Client.List(ctx, projectList); err != nil {
		return fmt.Errorf("error listing projects: %w", err)
	}

	for _, project := range projectList.Items {
		
		roleBinding := &rbacv1.RoleBinding{}

		err := r.Client.Get(ctx, client.ObjectKey{Name: roleBindingName, Namespace: project.Namespace}, roleBinding)
		if err != nil {
			if kubeerr.IsNotFound(err) {
				logger.Debug("RoleBinding not found, nothing to remove", "roleBinding", roleBindingName, "namespace", project.Namespace)
				continue // Skip to the next project if not found
			}
			return fmt.Errorf("error retrieving RoleBinding %q in namespace %q: %w", roleBindingName, project.Namespace, err)
		}

		// Delete the RoleBinding if it exists
		if err := r.Client.Delete(ctx, roleBinding); err != nil {
			return fmt.Errorf("error deleting RoleBinding %q in namespace %q: %w", roleBindingName, project.Namespace, err)
		}

		logger.Debug("Deleted RoleBinding for ServiceAccount", "roleBinding", roleBindingName, "in namespace", project.Namespace)
	}
	logger.Info("Completed deletion of RoleBindings for all projects")
	return nil
}

func hasControllerLabel(sa *corev1.ServiceAccount) bool {
	return sa.GetLabels()[ComponentLabelKey] == ControllerLabelValue
}
