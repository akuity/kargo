package projects

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

const (
	controllerServiceAccountLabelKey     = "app.kubernetes.io/component"
	controllerServiceAccountLabelValue   = "controller"
	controllerReadSecretsClusterRoleName = "kargo-controller-read-secrets"
)

type ReconcilerConfig struct {
	ManageControllerRoleBindings bool   `envconfig:"MANAGE_CONTROLLER_ROLE_BINDINGS" default:"true"`
	KargoNamespace               string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
	MaxConcurrentReconciles      int    `envconfig:"MAX_CONCURRENT_PROJECT_RECONCILES" default:"4"`
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

	ensureAPIAdminPermissionsFn func(context.Context, *kargoapi.Project) error

	ensureControllerPermissionsFn func(context.Context, *kargoapi.Project) error

	ensureDefaultUserRolesFn func(context.Context, *kargoapi.Project) error

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
	r.ensureAPIAdminPermissionsFn = r.ensureAPIAdminPermissions
	r.ensureControllerPermissionsFn = r.ensureControllerPermissions
	r.ensureDefaultUserRolesFn = r.ensureDefaultUserRoles
	r.createServiceAccountFn = r.client.Create
	r.createRoleFn = r.client.Create
	r.createRoleBindingFn = r.client.Create
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

	// Find the Project
	project, err := r.getProjectFn(ctx, r.client, req.NamespacedName.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if project == nil {
		// Ignore if not found. This can happen if the Project was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	if project.DeletionTimestamp != nil {
		logger.Debug("Project is being deleted; nothing to do")
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
			name: "migrate spec to ProjectConfig",
			reconcile: func() (kargoapi.ProjectStatus, error) {
				// TODO(hidde): Remove this migration code when the spec field is
				// removed from Project.
				migrated, err := r.migrateSpecToProjectConfig(ctx, project)
				if err != nil {
					return status, err
				}
				if migrated {
					logger.Debug("migrated Project spec to ProjectConfig")
					return status, nil
				}
				return status, nil
			},
		},
		{
			name: "syncing project resources",
			reconcile: func() (kargoapi.ProjectStatus, error) {
				newStatus, err := r.syncProject(ctx, project)
				if err != nil {
					return newStatus, err
				}
				return newStatus, nil
			},
		},
		{
			name: "collecting project stats",
			reconcile: func() (kargoapi.ProjectStatus, error) {
				newStatus, err := r.collectStats(ctx, project)
				if err != nil {
					return newStatus, err
				}
				return newStatus, nil
			},
		},
	}
	for _, subR := range subReconcilers {
		logger.Debug(subR.name)

		// Reconcile the Project with the sub-reconciler.
		var err error
		status, err = subR.reconcile()

		// If an error occurred during the sub-reconciler, then we should
		// return the error which will cause the Project to be requeued.
		if err != nil {
			return status, err
		}

		// Patch the status of the Project after each sub-reconciler to show
		// progress.
		if err = kubeclient.PatchStatus(ctx, r.client, project, func(st *kargoapi.ProjectStatus) {
			*st = status
		}); err != nil {
			logger.Error(err, fmt.Sprintf("failed to update Project status after %s", subR.name))
		}
	}

	return status, nil
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
					kargoapi.ProjectLabelKey,
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

	if err := r.ensureAPIAdminPermissionsFn(ctx, project); err != nil {
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "EnsuringAPIServerPermissionsFailed",
			Message:            "Failed to ensure project permissions for API server: " + err.Error(),
			ObservedGeneration: project.GetGeneration(),
		})
		return *status, fmt.Errorf("error ensuring API server permissions: %w", err)
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

	ns := &corev1.Namespace{}
	if err := r.getNamespaceFn(
		ctx,
		types.NamespacedName{Name: project.Name},
		ns,
	); err != nil && !kubeerr.IsNotFound(err) {
		return fmt.Errorf("error getting namespace %q: %w", project.Name, err)
	} else if err == nil {
		// We found an existing namespace with the same name as the Project. It's
		// only a problem if it is not labeled as a Project namespace.
		if ns.Labels[kargoapi.ProjectLabelKey] != kargoapi.LabelTrueValue {
			return fmt.Errorf(
				"failed to sync Project %q with namespace %q: %w",
				project.Name, project.Name, errProjectNamespaceExists,
			)
		}
		for _, ownerRef := range ns.OwnerReferences {
			if ownerRef.UID == project.UID {
				logger.Debug("namespace exists and is already owned by this Project")
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
		updated, err := r.ensureFinalizerFn(ctx, r.client, ns)
		if err != nil {
			return fmt.Errorf("error ensuring finalizer on namespace %q: %w", project.Name, err)
		}
		if updated {
			logger.Debug("added finalizer to namespace")
		}
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
				kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
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

func (r *reconciler) ensureAPIAdminPermissions(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	const roleBindingName = "kargo-project-admin"

	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project.Name,
		"name", project.Name,
		"namespace", project.Name,
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
	}
	if err := r.createRoleBindingFn(ctx, roleBinding); err != nil {
		if kubeerr.IsAlreadyExists(err) {
			logger.Debug("RoleBinding already exists in project namespace")
			return nil
		}
		return fmt.Errorf(
			"error creating RoleBinding %q in project namespace %q: %w",
			roleBinding.Name,
			project.Name,
			err,
		)
	}
	logger.Debug("granted API server and kargo-admin project admin permissions")

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

	// Create/update a RoleBinding for each ServiceAccount
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
			"serviceAccount.namespace", sa.Namespace,
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
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project.Name,
		"name", project.Name,
		"namespace", project.Name,
	)

	const adminRoleName = "kargo-admin"
	const viewerRoleName = "kargo-viewer"
	allRoles := []string{adminRoleName, viewerRoleName}

	for _, saName := range allRoles {
		saLogger := logger.WithValues("serviceAccount", saName)
		if err := r.createServiceAccountFn(
			ctx,
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      saName,
					Namespace: project.Name,
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			},
		); err != nil {
			if kubeerr.IsAlreadyExists(err) {
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
				{ // For managing project-level access and credentials
					APIGroups: []string{""},
					Resources: []string{"secrets", "serviceaccounts"},
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
					Verbs:     []string{"create", "delete", "get", "list", "watch"},
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
					Resources: []string{"events", "serviceaccounts"},
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
	}
	for _, role := range roles {
		roleLogger := logger.WithValues(
			"role", role.Name,
			"namespace", project.Name,
		)
		if err := r.createRoleFn(ctx, role); err != nil {
			if kubeerr.IsAlreadyExists(err) {
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
			"roleBinding", rbName,
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
			if kubeerr.IsAlreadyExists(err) {
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

// migrateSpecToProjectConfig migrates the Project's Spec to a dedicated
// ProjectConfig resource if necessary. It returns a boolean indicating whether
// the Project resource was updated.
func (r *reconciler) migrateSpecToProjectConfig(
	ctx context.Context,
	project *kargoapi.Project,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)

	if project.Spec == nil { // nolint:staticcheck
		return false, nil
	}

	if api.HasMigrationAnnotationValue(project, api.MigratedProjectSpecToProjectConfig) {
		return false, nil
	}

	if len(project.Spec.PromotionPolicies) != 0 { // nolint:staticcheck
		projectCfg := &kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project.Name,
				Namespace: project.Name,
			},
			Spec: kargoapi.ProjectConfigSpec{
				PromotionPolicies: project.Spec.PromotionPolicies, // nolint:staticcheck
			},
		}
		if err := r.client.Create(ctx, projectCfg); err != nil {
			// If the ProjectConfig already exists, we can ignore the error. This
			// could happen because the ProjectConfig was created by the user without
			// them removing the spec from the Project. It could also be the result of
			// a partial migration by a previous reconciliation attempt.
			if !kubeerr.IsAlreadyExists(err) {
				return false, fmt.Errorf(
					"error creating ProjectConfig in project namespace %q: %w",
					project.Name, err,
				)
			}
			logger.Debug("ProjectConfig already exists")
		} else {
			logger.Debug("migrated Project spec to ProjectConfig")
		}
	}

	// Mark the Project as migrated. This will prevent the migration code from
	// running again in the future.
	api.AddMigrationAnnotationValue(project, api.MigratedProjectSpecToProjectConfig)
	if err := r.client.Update(ctx, project); err != nil {
		return false, fmt.Errorf(
			"error updating Project %q to add migrated label: %w",
			project.Name, err,
		)
	}
	return true, nil
}

func getRoleBindingName(serviceAccountName string) string {
	return fmt.Sprintf("%s-read-secrets", serviceAccountName)
}
