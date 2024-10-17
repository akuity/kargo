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

const controllerReadSecretsClusterRoleName = "kargo-controller-read-secrets"

type ReconcilerConfig struct {
	KargoNamespace                     string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
	ControllerServiceAccountLabelKey   string `envconfig:"CONTROLLER_SERVICE_ACCOUNT_LABEL_KEY" default:"app.kubernetes.io/component"` // nolint: lll
	ControllerServiceAccountLabelValue string `envconfig:"CONTROLLER_SERVICE_ACCOUNT_LABEL_VALUE" default:"controller"`
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
}

// SetupReconcilerWithManager initializes a reconciler for ServiceAccount
// resources and registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager, cfg ReconcilerConfig) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.ServiceAccount{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're only interested in soft deletes
					return false
				},
			},
		).
		Complete(newReconciler(kargoMgr.GetClient(), cfg))
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	return &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"serviceAccount", req.NamespacedName.Name,
		"serviceAccount.namespace", req.NamespacedName.Namespace,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling ServiceAccount")

	sa := &corev1.ServiceAccount{}
	if err := r.client.Get(ctx, req.NamespacedName, sa); err != nil {
		if kubeerr.IsNotFound(err) {
			// Ignore if not found. This can happen if the ServiceAccount was deleted
			// after the current reconciliation request was issued.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w",
			req.NamespacedName.Name, req.NamespacedName.Namespace, err,
		)
	}

	if (sa.DeletionTimestamp != nil || !r.hasControllerLabel(sa)) &&
		controllerutil.ContainsFinalizer(sa, kargoapi.FinalizerName) {
		// Ensure non-existence of RoleBindings that grant this controller
		// ServiceAccount access to read Secrets in all Project namespaces.
		if err := r.removeControllerPermissions(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, err
		}
		if controllerutil.RemoveFinalizer(sa, kargoapi.FinalizerName) {
			if err := r.client.Update(ctx, sa); err != nil {
				return ctrl.Result{}, fmt.Errorf(
					"error removing finalizer from ServiceAccount %q in namespace %q: %w",
					sa.Name, sa.Namespace, err,
				)
			}
			logger.Debug("removed finalizer from ServiceAccount")
		}
		logger.Debug("done reconciling ServiceAccount")
		return ctrl.Result{}, nil
	}

	if sa.DeletionTimestamp == nil && r.hasControllerLabel(sa) {
		// Ensure the existence of RoleBindings that grant this controller
		// ServiceAccount access to read Secrets in all Project namespaces.
		if controllerutil.AddFinalizer(sa, kargoapi.FinalizerName) {
			if err := r.client.Update(ctx, sa); err != nil {
				return ctrl.Result{}, fmt.Errorf(
					"error adding finalizer to ServiceAccount %q in namespace %q: %w",
					sa.Name, sa.Namespace, err,
				)
			}
			logger.Debug("added finalizer to ServiceAccount")
		}
		if err := r.ensureControllerPermissions(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Debug("done reconciling ServiceAccount")
	return ctrl.Result{}, nil
}

// ensureControllerPermissions ensure the existence of RoleBindings that grant
// this referenced controller ServiceAccount access to read Secrets in all
// Project namespaces.
func (r *reconciler) ensureControllerPermissions(ctx context.Context, sa types.NamespacedName) error {
	roleBindingName := getRoleBindingName(sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"roleBinding", roleBindingName,
	)
	logger.Debug("ensuring necessary RoleBinding in all Project namespaces")

	projectList := &kargoapi.ProjectList{}
	if err := r.client.List(ctx, projectList); err != nil {
		return fmt.Errorf("error listing Projects: %w", err)
	}

	// Loop through each Project to create or update the corresponding
	// RoleBinding.
	for _, project := range projectList.Items {
		projectLogger := logger.WithValues("project.namespace", project.Name)
		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: project.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     controllerReadSecretsClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      sa.Name,
					Namespace: r.cfg.KargoNamespace,
				},
			},
		}
		if err := r.client.Create(ctx, roleBinding); err != nil {
			if !kubeerr.IsAlreadyExists(err) {
				return fmt.Errorf(
					"error creating RoleBinding %q for ServiceAccount %q in Project namespace %q: %w",
					roleBinding.Name, sa.Name, project.Name, err,
				)
			}
			if err = r.client.Update(ctx, roleBinding); err != nil {
				return fmt.Errorf(
					"error updating existing RoleBinding %q in Project namespace %q: %w",
					roleBinding.Name, project.Name, err,
				)
			}
			projectLogger.Debug("updated existing RoleBinding")
		} else {
			projectLogger.Debug("created RoleBinding")
		}
	}
	logger.Debug("necessary RoleBindings exist in all Project namespaces")
	return nil
}

// removeControllerPermissions ensure the non-existence of RoleBindings that
// would grant the referenced controller ServiceAccount access to read Secrets
// in all Project namespaces.
func (r *reconciler) removeControllerPermissions(ctx context.Context, sa types.NamespacedName) error {
	roleBindingName := getRoleBindingName(sa.Name)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"roleBinding", roleBindingName,
	)
	logger.Debug("ensuring non-existence of necessary RoleBinding in all Project namespaces")

	projectList := &kargoapi.ProjectList{}
	if err := r.client.List(ctx, projectList); err != nil {
		return fmt.Errorf("error listing Projects: %w", err)
	}

	for _, project := range projectList.Items {
		projectLogger := logger.WithValues("project.namespace", project.Name)
		if err := r.client.Delete(
			ctx,
			&rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleBindingName,
					Namespace: project.Name,
				},
			},
		); err != nil {
			if kubeerr.IsNotFound(err) {
				projectLogger.Debug("RoleBinding not found")
				continue
			}
			return fmt.Errorf(
				"error deleting RoleBinding %q in Project namespace %q: %w",
				roleBindingName, project.Name, err,
			)
		}
		projectLogger.Debug("deleted RoleBinding")
	}
	logger.Debug("Completed deletion of RoleBindings for all Projects")
	return nil
}

func (r *reconciler) hasControllerLabel(sa *corev1.ServiceAccount) bool {
	return sa.GetLabels()[r.cfg.ControllerServiceAccountLabelKey] == r.cfg.ControllerServiceAccountLabelValue
}

func getRoleBindingName(serviceAccountName string) string {
	return fmt.Sprintf("%s-read-secrets", serviceAccountName)
}
