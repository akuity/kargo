package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/kubernetes"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	controllerServiceAccountLabelKey     = "app.kubernetes.io/component"
	controllerServiceAccountLabelValue   = "controller"
	controllerReadSecretsClusterRoleName = "kargo-controller-read-secrets"
	// nolint: gosec
	projectSecretsReaderClusterRoleName = "kargo-project-secrets-reader"
)

type ReconcilerConfig struct {
	ManageControllerRoleBindings bool   `envconfig:"MANAGE_CONTROLLER_ROLE_BINDINGS" default:"true"`
	KargoNamespace               string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
	MaxConcurrentReconciles      int    `envconfig:"MAX_CONCURRENT_PROJECT_RECONCILES" default:"4"`

	ManageExtendedPermissions bool `envconfig:"MANAGE_EXTENDED_PERMISSIONS" default:"false"`

	ManageOrchestrator             bool   `envconfig:"MANAGE_ORCHESTRATOR" default:"false"`
	OrchestratorServiceAccountName string `envconfig:"ORCHESTRATOR_SERVICE_ACCOUNT_NAME" default:""`
	OrchestratorClusterRoleName    string `envconfig:"ORCHESTRATOR_CLUSTER_ROLE_NAME" default:""`
	TokenManagerClusterRoleName    string `envconfig:"TOKEN_MANAGER_CLUSTER_ROLE_NAME" default:""`

	ControlPlaneServiceAccountName string `envconfig:"CONTROL_PLANE_SERVICE_ACCOUNT_NAME" default:""`
	ControlPlaneClusterRoleName    string `envconfig:"CONTROL_PLANE_CLUSTER_ROLE_NAME" default:""`

	ManagerServiceAccountName string `envconfig:"MANAGER_SERVICE_ACCOUNT_NAME" default:""`
	ManagerClusterRoleName    string `envconfig:"MANAGER_CLUSTER_ROLE_NAME" default:""`
	ManagedResourceNamespace  string `envconfig:"MANAGED_RESOURCE_NAMESPACE" default:""`

	ArgoCDServiceAccountName string `envconfig:"ARGOCD_SERVICE_ACCOUNT_NAME" default:""`
	ArgoCDRoleName           string `envconfig:"ARGOCD_ROLE_NAME" default:""`
	ArgoCDClusterRoleName    string `envconfig:"ARGOCD_CLUSTER_ROLE_NAME" default:""`
	ArgoCDNamespace          string `envconfig:"ARGOCD_NAMESPACE" default:"argocd"`
	ArgoCDWatchNamespaceOnly bool   `envconfig:"ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY" default:"false"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

var errProjectNamespaceExists = errors.New("namespace already exists and is not labeled as a Project namespace")

// reconciler reconciles Project resources.
type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client

	// The following behaviors are overridable for testing purposes:

	getProjectFn func(
		context.Context,
		client.Client,
		string,
	) (*kargoapi.Project, error)

	reconcileFn func(
		context.Context,
		*kargoapi.Project,
	) (kargoapi.ProjectStatus, error)

	ensureNamespaceFn func(context.Context, *kargoapi.Project) error

	patchProjectStatusFn func(
		context.Context,
		*kargoapi.Project,
		kargoapi.ProjectStatus,
	) error

	getNamespaceFn func(
		context.Context,
		types.NamespacedName,
		client.Object,
		...client.GetOption,
	) error

	createNamespaceFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	patchOwnerReferencesFn func(
		context.Context,
		client.Client,
		client.Object,
	) error

	ensureFinalizerFn func(
		context.Context,
		client.Client,
		client.Object,
	) (bool, error)

	removeFinalizerFn func(
		context.Context,
		client.Client,
		client.Object,
	) error

	ensureSystemPermissionsFn func(context.Context, *kargoapi.Project) error

	ensureControllerPermissionsFn func(context.Context, *kargoapi.Project) error

	ensureDefaultUserRolesFn func(context.Context, *kargoapi.Project) error

	ensureExtendedPermissionsFn func(context.Context, *kargoapi.Project) error

	createServiceAccountFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	createRoleFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	createRoleBindingFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	createClusterRoleFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	createClusterRoleBindingFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	deleteClusterRoleFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error

	deleteClusterRoleBindingFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Project resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Project{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any deletes
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Build(newReconciler(kargoMgr.GetClient(), cfg))
	if err != nil {
		return fmt.Errorf("error creating Project reconciler: %w", err)
	}

	// Watch for Warehouses for which the health condition has changed.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Warehouse{},
			&projectWarehouseHealthEnqueuer[*kargoapi.Warehouse]{},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Warehouses: %w", err)
	}

	// Watch for Stages for which the health condition has changed.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Stage{},
			&projectStageHealthEnqueuer[*kargoapi.Stage]{},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Stages: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized Project reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)

	return err
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	r := &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
	r.getProjectFn = api.GetProject
	r.reconcileFn = r.reconcile
	r.ensureNamespaceFn = r.ensureNamespace
	r.patchProjectStatusFn = r.patchProjectStatus
	r.getNamespaceFn = r.client.Get
	r.createNamespaceFn = r.client.Create
	r.patchOwnerReferencesFn = api.PatchOwnerReferences
	r.ensureFinalizerFn = api.EnsureFinalizer
	r.removeFinalizerFn = api.RemoveFinalizer
	r.ensureSystemPermissionsFn = r.ensureSystemPermissions
	r.ensureControllerPermissionsFn = r.ensureControllerPermissions
	r.ensureDefaultUserRolesFn = r.ensureDefaultUserRoles
	r.ensureExtendedPermissionsFn = r.ensureExtendedPermissions
	r.createServiceAccountFn = r.client.Create
	r.createRoleFn = r.client.Create
	r.createRoleBindingFn = r.client.Create
	r.createClusterRoleFn = r.client.Create
	r.createClusterRoleBindingFn = r.client.Create
	r.deleteClusterRoleFn = r.client.Delete
	r.deleteClusterRoleBindingFn = r.client.Delete
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", req.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Find the Project
	project, err := r.getProjectFn(ctx, r.client, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if project == nil {
		// Ignore if not found. This can happen if the Project was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	// ensure the finalizer is present on the Project
	updated, err := r.ensureFinalizerFn(ctx, r.client, project)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer on Project %q: %w", project.Name, err)
	}
	if updated {
		logger.Debug("added finalizer to Project")
	}

	if project.DeletionTimestamp != nil {
		if err = r.cleanupProject(ctx, project); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	logger.Debug("reconciling Project")
	newStatus, reconcileErr := r.reconcileFn(ctx, project)
	logger.Debug("done reconciling Project")

	// Patch the status of the Project.
	if err := kubeclient.PatchStatus(ctx, r.client, project, func(status *kargoapi.ProjectStatus) {
		*status = newStatus
	}); err != nil {
		// Prioritize the reconcile error if it exists.
		if reconcileErr != nil {
			logger.Error(err, "failed to update Project status after reconciliation error")
			return ctrl.Result{}, reconcileErr
		}
		return ctrl.Result{}, fmt.Errorf("failed to update Project status: %w", err)
	}

	// Return the reconcile error if it exists.
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}
	// Otherwise, requeue after a delay.
	// TODO: Make the requeue delay configurable.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *reconciler) reconcile(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	status := *project.Status.DeepCopy()

	subReconcilers := []struct {
		name      string
		reconcile func() (kargoapi.ProjectStatus, error)
	}{
		{
			name: "syncing project resources",
			reconcile: func() (kargoapi.ProjectStatus, error) {
				return r.syncProject(ctx, project)
			},
		},
		{
			name: "collecting project stats",
			reconcile: func() (kargoapi.ProjectStatus, error) {
				return r.collectStats(ctx, project)
			},
		},
	}
	for _, subR := range subReconcilers {
		logger.Debug(subR.name)

		// Reconcile the Project with the sub-reconciler.
		var err error
		status, err = subR.reconcile()
		// If an error occurred during the sub-reconciler, then we should return the
		// error which will cause the Project to be requeued.
		if err != nil {
			return status, err
		}

		// Patch the status of the Project after each sub-reconciler to show
		// progress.
		if err = kubeclient.PatchStatus(
			ctx,
			r.client,
			project,
			func(st *kargoapi.ProjectStatus) { *st = status },
		); err != nil {
			logger.Error(
				err,
				fmt.Sprintf("failed to update Project status after %s", subR.name),
			)
		}
	}

	return status, nil
}

// cleanupProject handles the deletion and cleanup of a Project's associated
// resources.
func (r *reconciler) cleanupProject(ctx context.Context, project *kargoapi.Project) error {
	logger := logging.LoggerFromContext(ctx)

	// Delete cluster-scoped resources that are specific to the Project
	crbs := map[string]bool{
		kubernetes.ShortenResourceName(fmt.Sprintf("kargo-project-admin-%s", project.Name)): true,
	}
	if r.cfg.ManageExtendedPermissions && r.cfg.ArgoCDClusterRoleName != "" {
		crbs[kubernetes.ShortenResourceName(fmt.Sprintf("%s-%s", r.cfg.ArgoCDClusterRoleName, project.Name))] = false
	}
	for crbName, hasCR := range crbs {
		if err := r.deleteClusterRoleBindingFn(
			ctx,
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: crbName}},
		); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("error deleting ClusterRoleBinding %q: %w", crbName, err)
		}

		if !hasCR {
			// No ClusterRole to delete for this ClusterRoleBinding.
			continue
		}

		crName := crbName
		if err := r.deleteClusterRoleFn(
			ctx,
			&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: crName}},
		); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("error deleting ClusterRole %q: %w", crName, err)
		}
	}

	// Get namespace for the Project
	ns := &corev1.Namespace{}
	err := r.getNamespaceFn(ctx, types.NamespacedName{Name: project.Name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Namespace already deleted or never existed, just remove finalizer
			// from project
			logger.Debug("namespace not found, removing project finalizer")
			if err = r.removeFinalizerFn(ctx, r.client, project); err != nil {
				return fmt.Errorf("failed to remove finalizer from project %q: %w", project.Name, err)
			}
			return nil
		}
		return fmt.Errorf("error getting namespace %q: %w", project.Name, err)
	}

	if shouldKeepNamespace(project, ns) {
		logger.Debug("keeping namespace due to keep-namespace annotation")

		// Remove only this Project's OwnerReference from the Namespace
		var newOwnerRefs []metav1.OwnerReference
		for _, ref := range ns.OwnerReferences {
			if ref.UID != project.UID {
				newOwnerRefs = append(newOwnerRefs, ref)
			}
		}

		// Only update owner references if we actually found and removed one
		if len(newOwnerRefs) < len(ns.OwnerReferences) {
			ns.OwnerReferences = newOwnerRefs
			if err = r.patchOwnerReferencesFn(ctx, r.client, ns); err != nil {
				return fmt.Errorf("failed to patch namespace %q owner references: %w", ns.Name, err)
			}
			logger.Debug("removed project owner reference from namespace")
		}
	}

	// Remove finalizer from namespace
	if err = r.removeFinalizerFn(ctx, r.client, ns); err != nil {
		return fmt.Errorf("failed to remove finalizer from namespace %q: %w", ns.Name, err)
	}

	// Remove finalizer from Project
	if err = r.removeFinalizerFn(ctx, r.client, project); err != nil {
		return fmt.Errorf("failed to remove finalizer from project %q: %w", project.Name, err)
	}
	return nil
}

// syncProject ensures the existence of the Project's namespace and any
// resources that are required for the Project to function properly. It
// returns an updated ProjectStatus.
func (r *reconciler) syncProject(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	status := project.Status.DeepCopy()

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "Syncing",
		Message:            "Ensuring project namespace and permissions",
		ObservedGeneration: project.GetGeneration(),
	})
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "Syncing",
		Message:            "Ensuring project namespace and permissions",
		ObservedGeneration: project.GetGeneration(),
	})

	if err := r.ensureNamespaceFn(ctx, project); err != nil {
		if errors.Is(err, errProjectNamespaceExists) {
			// Stalled is a special condition because this won't be resolved without
			// user intervention.
			conditions.Set(status, &metav1.Condition{
				Type:   kargoapi.ConditionTypeStalled,
				Status: metav1.ConditionTrue,
				Reason: "ExistingNamespaceMissingLabel",
				Message: fmt.Sprintf(
					"Namespace %q already exists but is not labeled as a Project namespace using label %q",
					project.Name,
					kargoapi.LabelKeyProject,
				),
				ObservedGeneration: project.GetGeneration(),
			})
		} else {
			conditions.Delete(status, kargoapi.ConditionTypeStalled)
		}
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "EnsuringNamespaceFailed",
			Message:            "Failed to ensure existence of project namespace: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return *status, fmt.Errorf("error ensuring namespace: %w", err)
	}

	if err := r.ensureSystemPermissionsFn(ctx, project); err != nil {
		conditions.Set(status, &metav1.Condition{
			Type:   kargoapi.ConditionTypeReady,
			Status: metav1.ConditionFalse,
			Reason: "EnsuringSystemPermissionsFailed",
			Message: "Failed to ensure Project permissions for system " +
				"ServiceAccounts: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return *status, fmt.Errorf(
			"error ensuring Project permissions for system ServiceAccounts: %w",
			err,
		)
	}

	if r.cfg.ManageControllerRoleBindings {
		if err := r.ensureControllerPermissionsFn(ctx, project); err != nil {
			conditions.Set(status, &metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "EnsuringControllerPermissionsFailed",
				Message:            "Failed to ensure project permissions for controller: " + err.Error(),
				ObservedGeneration: project.GetGeneration(),
			})
			return *status, fmt.Errorf("error ensuring controller permissions: %w", err)
		}
	}

	if err := r.ensureDefaultUserRolesFn(ctx, project); err != nil {
		conditions.Set(status, &metav1.Condition{
			Type:   kargoapi.ConditionTypeReady,
			Status: metav1.ConditionFalse,
			Reason: "EnsuringDefaultUserRoles",
			Message: "Failed to ensure existence of default project " +
				"ServiceAccount, Roles, and RoleBindings: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return *status, fmt.Errorf("error ensuring default project roles: %w", err)
	}

	if r.cfg.ManageExtendedPermissions {
		if err := r.ensureExtendedPermissionsFn(ctx, project); err != nil {
			conditions.Set(status, &metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "EnsuringExtendedPermissionsFailed",
				Message:            "Failed to ensure existence of extended permissions: " + err.Error(),
				ObservedGeneration: project.GetGeneration(),
			})
			return *status, fmt.Errorf("error ensuring extended permissions: %w", err)
		}
	}

	conditions.Delete(status, kargoapi.ConditionTypeReconciling)
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "Project is synced and ready for use",
		ObservedGeneration: project.GetGeneration(),
	})
	return *status, nil
}

func (r *reconciler) ensureNamespace(ctx context.Context, project *kargoapi.Project) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project.Name)

	ownerRef := metav1.NewControllerRef(
		project,
		kargoapi.GroupVersion.WithKind("Project"),
	)
	ownerRef.BlockOwnerDeletion = ptr.To(false)
	ownerRef.Controller = nil

	ns := &corev1.Namespace{}
	if err := r.getNamespaceFn(
		ctx,
		types.NamespacedName{Name: project.Name},
		ns,
	); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("error getting namespace %q: %w", project.Name, err)
	} else if err == nil {
		// We found an existing namespace with the same name as the Project. It's
		// only a problem if it is not labeled as a Project namespace.
		if ns.Labels[kargoapi.LabelKeyProject] != kargoapi.LabelValueTrue {
			return fmt.Errorf(
				"failed to sync Project %q with namespace %q: %w",
				project.Name, project.Name, errProjectNamespaceExists,
			)
		}

		// always ensure finalizer is present on the namespace
		updated, err := r.ensureFinalizerFn(ctx, r.client, ns)
		if err != nil {
			return fmt.Errorf("error ensuring finalizer on namespace %q: %w", project.Name, err)
		}
		if updated {
			logger.Debug("added finalizer to namespace")
		}

		for i, ownerRef := range ns.OwnerReferences {
			if ownerRef.UID == project.UID {
				logger.Debug("namespace exists and is already owned by this Project")
				if ownerRef.Controller != nil {
					logger.Debug("owner reference requires update")
					ns.OwnerReferences[i].Controller = nil // Update in place
					if err = r.patchOwnerReferencesFn(ctx, r.client, ns); err != nil {
						return fmt.Errorf(
							"error patching namespace %q owner references: %w",
							project.Name, err,
						)
					}
					logger.Debug("updated owner reference")
				}
				return nil
			}
		}

		// If we get to here, the Project is not already an owner of the existing
		// namespace.
		logger.Debug(
			"namespace exists, is not owned by this Project, but has the " +
				"project label; Project will adopt it",
		)

		// Note: We allow multiple owners of a namespace due to the not entirely
		// uncommon scenario where an organization has its own controller that
		// creates and initializes namespaces to ensure compliance with
		// internal policies. Such a controller might already own the namespace.
		ns.OwnerReferences = append(ns.OwnerReferences, *ownerRef)
		if err = r.patchOwnerReferencesFn(ctx, r.client, ns); err != nil {
			return fmt.Errorf(
				"error patching namespace %q with project %q as owner: %w",
				project.Name,
				project.Name,
				err,
			)
		}
		logger.Debug("patched namespace with Project as owner")

		return nil
	}

	// If we get to here, we had a not found error and we can proceed with
	// creating the namespace.

	logger.Debug("namespace does not exist yet; creating namespace")

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
			},
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
	}
	// Project namespaces are owned by a Project. Deleting a Project automatically
	// deletes the namespace. But we also want this to work in the other
	// direction, where that behavior is not automatic. We add a finalizer to the
	// namespace and use our own namespace reconciler to clear it after deleting
	// the Project.
	controllerutil.AddFinalizer(ns, kargoapi.FinalizerName)
	if err := r.createNamespaceFn(ctx, ns); err != nil {
		return fmt.Errorf("error creating namespace %q: %w", project.Name, err)
	}
	logger.Debug("created namespace")

	return nil
}

// ensureSystemPermissions ensures that system-level ServiceAccounts, including
// that for the Kargo admin, have all necessary permissions to operate on the
// specified Project. This excludes permissions for the controllers, which have
// their own dedicated method for ensuring their permissions.
func (r *reconciler) ensureSystemPermissions(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project.Name,
		"name", project.Name,
		"namespace", project.Name,
	)

	roleBindings := []rbacv1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kargo-project-admin",
				Namespace: project.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "kargo-project-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "kargo-api",
					Namespace: r.cfg.KargoNamespace,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "kargo-admin",
					Namespace: r.cfg.KargoNamespace,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kargo-project-secrets-reader",
				Namespace: project.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     projectSecretsReaderClusterRoleName,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "kargo-external-webhooks-server",
				Namespace: r.cfg.KargoNamespace,
			}},
		},
	}
	for _, roleBinding := range roleBindings {
		rbLogger := logger.WithValues("roleBinding", roleBinding.Name)
		if err := r.createRoleBindingFn(ctx, &roleBinding); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf(
					"error creating RoleBinding %q in Project namespace %q: %w",
					roleBinding.Name, project.Name, err,
				)
			}
			if err = r.client.Update(ctx, &roleBinding); err != nil {
				return fmt.Errorf(
					"error updating existing RoleBinding %q in Project namespace %q: %w",
					roleBinding.Name, project.Name, err,
				)
			}
			rbLogger.Debug("updated RoleBinding")
			continue
		}
		rbLogger.Debug("created RoleBinding in Project namespace")
	}

	return nil
}

func (r *reconciler) ensureControllerPermissions(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project.Name,
		"namespace", project.Name,
	)

	// Get all ServiceAccounts labeled as controller ServiceAccounts
	controllerSAs := &corev1.ServiceAccountList{}
	if err := r.client.List(
		ctx, controllerSAs,
		client.InNamespace(r.cfg.KargoNamespace),
		client.MatchingLabels{
			controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
		},
	); err != nil {
		return fmt.Errorf("error listing controller ServiceAccounts: %w", err)
	}

	// Create/update RoleBindings for each ServiceAccount
	for _, controllerSA := range controllerSAs.Items {
		sa := &controllerSA
		if controllerutil.AddFinalizer(sa, kargoapi.FinalizerName) {
			if err := r.client.Update(ctx, sa); err != nil {
				return fmt.Errorf(
					"error adding finalizer to controller ServiceAccount %q in namespace %q: %w",
					sa.Name, sa.Namespace, err,
				)
			}
		}

		roleBindingName := getRoleBindingName(sa.Name)
		saLogger := logger.WithValues(
			"serviceAccount", sa.Name,
			corev1.ServiceAccountNameKey, sa.Namespace,
			"roleBinding", roleBindingName,
		)

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
					Namespace: sa.Namespace,
				},
			},
		}

		if err := r.client.Create(ctx, roleBinding); err != nil {
			if !apierrors.IsAlreadyExists(err) {
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
			saLogger.Debug("updated RoleBinding")
			continue
		}
		saLogger.Debug("created RoleBinding")
	}

	return nil
}

func (r *reconciler) ensureDefaultUserRoles(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project.Name)

	const adminRoleName = "kargo-admin"
	const viewerRoleName = "kargo-viewer"
	const promoterRoleName = "kargo-promoter"
	allRoles := []string{adminRoleName, viewerRoleName, promoterRoleName}
	saAnnotations := map[string]string{
		rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
	}
	if creator, ok := project.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		if parts := strings.SplitN(creator, ":", 2); len(parts) == 2 {
			saAnnotations[rbacapi.AnnotationKeyOIDCClaims] = fmt.Sprintf("{%q:[%q]}", parts[0], parts[1])
		}
	}
	for _, saName := range allRoles {
		saLogger := logger.WithValues(
			"name", saName,
			"namespace", project.Name,
		)
		if err := r.createServiceAccountFn(
			ctx,
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        saName,
					Namespace:   project.Name,
					Annotations: saAnnotations,
				},
			},
		); err != nil {
			if apierrors.IsAlreadyExists(err) {
				saLogger.Debug("ServiceAccount already exists in project namespace")
				continue
			}
			return fmt.Errorf(
				"error creating ServiceAccount %q in project namespace %q: %w",
				saName,
				project.Name,
				err,
			)
		}
	}

	roles := []*rbacv1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      adminRoleName,
				Namespace: project.Name,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			},
			Rules: []rbacv1.PolicyRule{
				{ // For viewing events; no need to create, edit, or delete them
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{ // For managing project-level access, credentials, and other config
					APIGroups: []string{""},
					Resources: []string{"configmaps", "secrets", "serviceaccounts"},
					Verbs:     []string{"*"},
				},
				{ // For managing project-level access
					APIGroups: []string{rbacv1.SchemeGroupVersion.Group},
					Resources: []string{"rolebindings", "roles"},
					Verbs:     []string{"*"},
				},
				{ // Full access to all mutable Kargo resource types
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"freights", "stages", "warehouses", "projectconfigs"},
					Verbs:     []string{"*"},
				},
				{ // Promote permission on all stages
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"promote"},
				},
				{ // Nearly full access to all Promotions, but they are immutable
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"promotions"},
					Verbs:     []string{"create", "delete", "get", "list", "watch", "patch"},
				},
				{ // Manual approvals involve patching Freight status
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"freights/status"},
					Verbs:     []string{"patch"},
				},
				{
					// View and delete AnalysisRuns
					APIGroups: []string{rolloutsapi.GroupVersion.Group},
					Resources: []string{"analysisruns"},
					Verbs:     []string{"delete", "get", "list", "watch"},
				},
				{ // Full access to AnalysisTemplates
					APIGroups: []string{rolloutsapi.GroupVersion.Group},
					Resources: []string{"analysistemplates"},
					Verbs:     []string{"*"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      viewerRoleName,
				Namespace: project.Name,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "events", "serviceaccounts"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{rbacv1.SchemeGroupVersion.Group},
					Resources: []string{"rolebindings", "roles"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"freights", "promotions", "stages", "warehouses", "projectconfigs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{rolloutsapi.GroupVersion.Group},
					Resources: []string{"analysisruns", "analysistemplates"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      promoterRoleName,
				Namespace: project.Name,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			},
			Rules: []rbacv1.PolicyRule{
				{ // For viewing configmaps, events, and serviceaccounts
					APIGroups: []string{""},
					Resources: []string{"configmaps", "events", "serviceaccounts"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{ // For viewing project-level access
					APIGroups: []string{rbacv1.SchemeGroupVersion.Group},
					Resources: []string{"rolebindings", "roles"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{ // View access to Kargo resources
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"freights", "stages", "warehouses", "projectconfigs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{ // Promote permission on all stages
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"promote"},
				},
				{ // Can create and view promotions
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"promotions"},
					Verbs:     []string{"create", "get", "list", "watch"},
				},
				{ // Manual approvals involve patching Freight status
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"freights/status"},
					Verbs:     []string{"patch"},
				},
				{ // View AnalysisRuns and AnalysisTemplates
					APIGroups: []string{rolloutsapi.GroupVersion.Group},
					Resources: []string{"analysisruns", "analysistemplates"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
	for _, role := range roles {
		roleLogger := logger.WithValues(
			"name", role.Name,
			"namespace", project.Name,
		)
		if err := r.createRoleFn(ctx, role); err != nil {
			if apierrors.IsAlreadyExists(err) {
				roleLogger.Debug("Role already exists in project namespace")
				continue
			}
			return fmt.Errorf(
				"error creating Role %q in project namespace %q: %w",
				role.Name, project.Name, err,
			)
		}
		roleLogger.Debug("created Role in project namespace")
	}

	for _, rbName := range allRoles {
		rbLogger := logger.WithValues(
			"name", rbName,
			"namespace", project.Name,
		)
		if err := r.createRoleBindingFn(
			ctx,
			&rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rbName,
					Namespace: project.Name,
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "Role",
					Name:     rbName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      rbName,
						Namespace: project.Name,
					},
				},
			},
		); err != nil {
			if apierrors.IsAlreadyExists(err) {
				rbLogger.Debug("RoleBinding already exists in project namespace")
				continue
			}
			return fmt.Errorf(
				"error creating RoleBinding %q in project namespace %q: %w",
				rbName, project.Name, err,
			)
		}
		rbLogger.Debug("created RoleBinding in project namespace")
	}

	// This ClusterRole allows those bound to it to update and delete one specific
	// Project. This is necessary since Projects are cluster-scoped, meaning this
	// permission cannot be defined anywhere except at the cluster-level.
	crName := kubernetes.ShortenResourceName(
		fmt.Sprintf("kargo-project-admin-%s", project.Name),
	)
	crLogger := logger.WithValues("name", crName)
	if err := r.createClusterRoleFn(
		ctx,
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: crName},
			Rules: []rbacv1.PolicyRule{{
				APIGroups:     []string{kargoapi.GroupVersion.Group},
				Resources:     []string{"projects"},
				ResourceNames: []string{project.Name},
				Verbs:         []string{"delete", "update"},
			}},
		},
	); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("error creating ClusterRole %q: %w", crName, err)
		}
		crLogger.Debug("ClusterRole already exists")
	}
	crLogger.Debug("created ClusterRole")

	crbName := crName
	crbLogger := logger.WithValues("name", crbName)
	logger.WithValues()
	if err := r.createClusterRoleBindingFn(
		ctx,
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: crbName},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     crName,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      adminRoleName,
				Namespace: project.Name,
			}},
		},
	); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("error creating ClusterRoleBinding %q: %w", crName, err)
		}
		crbLogger.Debug("ClusterRoleBinding already exists")
	}
	crbLogger.Debug("created ClusterRoleBinding")

	return nil
}

func (r *reconciler) ensureExtendedPermissions(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project.Name)

	var serviceAccounts []string

	// If a control plane ServiceAccount is configured, then add it to the list
	// of ServiceAccounts to create and bind extended permissions to.
	if r.cfg.ControlPlaneServiceAccountName != "" {
		serviceAccounts = append(serviceAccounts, r.cfg.ControlPlaneServiceAccountName)
	}

	// If a dedicated namespace for managed resources is configured, then we do
	// not have to create the orchestrator ServiceAccount in every Project
	// namespace.
	if r.cfg.ManageOrchestrator && r.cfg.ManagedResourceNamespace == "" && r.cfg.OrchestratorServiceAccountName != "" {
		serviceAccounts = append([]string{
			r.cfg.OrchestratorServiceAccountName,
		}, serviceAccounts...)
	}

	// If an Argo CD ServiceAccount is configured, then add it to the list of
	// ServiceAccounts to create and bind extended permissions to.
	//
	// Note: We only do this if Argo CD is not restricted to watch its own
	// namespace only. If it is, then the Argo CD ServiceAccount does not need
	// to exist in every Project namespace, and we do not need to create a
	// ClusterRoleBinding for it.
	if r.cfg.ArgoCDServiceAccountName != "" {
		serviceAccounts = append(serviceAccounts, r.cfg.ArgoCDServiceAccountName)
	}

	saAnnotations := map[string]string{
		rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
	}

	for _, saName := range serviceAccounts {
		saLogger := logger.WithValues(
			"name", saName,
			"namespace", project.Name,
		)

		if err := r.createServiceAccountFn(
			ctx,
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        saName,
					Namespace:   project.Name,
					Annotations: saAnnotations,
				},
			},
		); err != nil {
			if apierrors.IsAlreadyExists(err) {
				saLogger.Debug("ServiceAccount already exists in project namespace")
				continue
			}
			return fmt.Errorf(
				"error creating ServiceAccount %q in project namespace %q: %w",
				saName,
				project.Name,
				err,
			)
		}
	}

	var clusterRoleBindings []*rbacv1.ClusterRoleBinding

	if r.cfg.ArgoCDServiceAccountName != "" && !r.cfg.ArgoCDWatchNamespaceOnly {
		// ClusterRoleBinding for the Argo CD ServiceAccount to access and
		// operate on Argo CD Application resources in any namespace.
		clusterRoleBindings = append(clusterRoleBindings, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubernetes.ShortenResourceName(fmt.Sprintf("%s-%s", r.cfg.ArgoCDClusterRoleName, project.Name)),
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     r.cfg.ArgoCDClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.ArgoCDServiceAccountName,
					Namespace: project.Name,
				},
			},
		})
	}

	for _, crb := range clusterRoleBindings {
		crbLogger := logger.WithValues("name", crb.Name, "project", project.Name)
		if err := r.createClusterRoleBindingFn(ctx, crb); err != nil {
			if apierrors.IsAlreadyExists(err) {
				crbLogger.Debug("ClusterRoleBinding already exists for Project")
				continue
			}

			return fmt.Errorf(
				"error creating ClusterRoleBinding %q for Project %q: %w",
				crb.Name, project.Name, err,
			)
		}
	}

	rbAnnotations := map[string]string{
		rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
	}

	var roleBindings []*rbacv1.RoleBinding

	if r.cfg.ControlPlaneServiceAccountName != "" && r.cfg.ControlPlaneClusterRoleName != "" {
		roleBindings = append(roleBindings, &rbacv1.RoleBinding{
			// RoleBinding for the control plane ServiceAccount to access and
			// operate on a minimal set of Kargo resources in the Project
			// namespace.
			ObjectMeta: metav1.ObjectMeta{
				Name:        r.cfg.ControlPlaneServiceAccountName,
				Namespace:   project.Name,
				Annotations: rbAnnotations,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     r.cfg.ControlPlaneClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.ControlPlaneClusterRoleName,
					Namespace: project.Name,
				},
			},
		})
	}

	if r.cfg.ManagerServiceAccountName != "" && r.cfg.ManagedResourceNamespace == "" {
		// RoleBinding for the manager ServiceAccount to manage resources in
		// the Project namespace, if no dedicated managed resources namespace
		// is configured.
		roleBindings = append(roleBindings, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        r.cfg.ManagerClusterRoleName,
				Namespace:   project.Name,
				Annotations: rbAnnotations,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     r.cfg.ManagerClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.ManagerServiceAccountName,
					Namespace: r.cfg.KargoNamespace,
				},
			},
		})
	}

	if r.cfg.OrchestratorServiceAccountName != "" {
		// RoleBinding for the orchestrator ServiceAccount to access and operate
		// on resources in the Project namespace.
		roleBindings = append(roleBindings, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        r.cfg.OrchestratorServiceAccountName,
				Namespace:   project.Name,
				Annotations: rbAnnotations,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     r.cfg.OrchestratorClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.OrchestratorServiceAccountName,
					Namespace: project.Name,
				},
			},
		})

		if r.cfg.TokenManagerClusterRoleName != "" {
			// RoleBinding for the orchestrator ServiceAccount to create and
			// manage ServiceAccount tokens in the Project namespace.
			roleBindings = append(roleBindings, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:        r.cfg.TokenManagerClusterRoleName,
					Namespace:   project.Name,
					Annotations: rbAnnotations,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     r.cfg.TokenManagerClusterRoleName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      r.cfg.OrchestratorClusterRoleName,
						Namespace: project.Name,
					},
				},
			})
		}

		// RoleBinding for the orchestrator ServiceAccount to access Secrets
		// in the Project namespace.
		roleBindings = append(roleBindings, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubernetes.ShortenResourceName(
					fmt.Sprintf("%s-secrets-reader", r.cfg.OrchestratorServiceAccountName),
				),
				Namespace:   project.Name,
				Annotations: rbAnnotations,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     projectSecretsReaderClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.OrchestratorServiceAccountName,
					Namespace: project.Name,
				},
			},
		})
	}

	if r.cfg.ArgoCDServiceAccountName != "" && r.cfg.ArgoCDNamespace != "" && r.cfg.ArgoCDWatchNamespaceOnly {
		// RoleBinding for the Argo CD ServiceAccount to access and operate
		// on Applications in the Argo CD namespace.
		roleBindings = append(roleBindings, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubernetes.ShortenResourceName(
					fmt.Sprintf("%s-%s", r.cfg.ArgoCDRoleName, project.Name),
				),
				Namespace:   r.cfg.ArgoCDNamespace,
				Annotations: rbAnnotations,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     r.cfg.ArgoCDRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      r.cfg.ArgoCDServiceAccountName,
					Namespace: project.Name,
				},
			},
		})
	}

	for _, rb := range roleBindings {
		rbLogger := logger.WithValues(
			"name", rb.Name,
			"namespace", project.Name,
		)

		if err := r.createRoleBindingFn(ctx, rb); err != nil {
			if apierrors.IsAlreadyExists(err) {
				rbLogger.Debug("RoleBinding already exists in project namespace")
				continue
			}
			return fmt.Errorf(
				"error creating RoleBinding %q in project namespace %q: %w",
				rb.Name, project.Name, err,
			)
		}
	}

	return nil
}

func (r *reconciler) patchProjectStatus(
	ctx context.Context,
	project *kargoapi.Project,
	status kargoapi.ProjectStatus,
) error {
	return kubeclient.PatchStatus(
		ctx,
		r.client,
		project,
		func(s *kargoapi.ProjectStatus) {
			*s = status
		},
	)
}

// shouldKeepNamespace determines if a Namespace should be kept during Project
// cleanup based on the keep-namespace annotation on either the Project or
// Namespace.
func shouldKeepNamespace(project *kargoapi.Project, ns *corev1.Namespace) bool {
	return project.Annotations[kargoapi.AnnotationKeyKeepNamespace] == kargoapi.AnnotationValueTrue ||
		ns.Annotations[kargoapi.AnnotationKeyKeepNamespace] == kargoapi.AnnotationValueTrue
}

func getRoleBindingName(serviceAccountName string) string {
	return fmt.Sprintf("%s-read-secrets", serviceAccountName)
}
