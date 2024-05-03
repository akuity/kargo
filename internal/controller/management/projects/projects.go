package projects

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
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

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

type ReconcilerConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" required:"true"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

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

	syncProjectFn func(
		context.Context,
		*kargoapi.Project,
	) (kargoapi.ProjectStatus, error)

	ensureNamespaceFn func(
		context.Context,
		*kargoapi.Project,
	) (kargoapi.ProjectStatus, error)

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

	updateNamespaceFn func(
		context.Context,
		client.Object,
		...client.UpdateOption,
	) error

	ensureAPIAdminPermissionsFn func(context.Context, *kargoapi.Project) error

	ensureDefaultProjectRolesFn func(context.Context, *kargoapi.Project) error

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
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Project{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any deletes
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(kargoMgr.GetClient(), cfg))
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	r := &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
	r.getProjectFn = kargoapi.GetProject
	r.syncProjectFn = r.syncProject
	r.ensureNamespaceFn = r.ensureNamespace
	r.patchProjectStatusFn = r.patchProjectStatus
	r.getNamespaceFn = r.client.Get
	r.createNamespaceFn = r.client.Create
	r.updateNamespaceFn = r.client.Update
	r.ensureAPIAdminPermissionsFn = r.ensureAPIAdminPermissions
	r.ensureDefaultProjectRolesFn = r.ensureDefaultProjectRoles
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
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Project")

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

	if project.Status.Phase.IsTerminal() {
		logger.Debugf("Project is %s; nothing to do", project.Status.Phase)
		return ctrl.Result{}, nil
	}

	newStatus, err := r.syncProjectFn(ctx, project)
	if err != nil {
		newStatus.Message = err.Error()
		logger.Errorf("error syncing Project: %s", err)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		newStatus.Message = ""
	}

	patchErr := r.patchProjectStatusFn(ctx, project, newStatus)
	if patchErr != nil {
		logger.Errorf("error updating Project status: %s", patchErr)
	}

	// If we had no error, but couldn't patch, then we DO have an error. But we
	// do it this way so that a failure to patch is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = patchErr
	}
	logger.Debug("done reconciling Project")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return ctrl.Result{}, err
}

func (r *reconciler) syncProject(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	status, err := r.ensureNamespaceFn(ctx, project)
	if err != nil {
		return status, fmt.Errorf("error ensuring namespace: %w", err)
	}

	if err = r.ensureAPIAdminPermissionsFn(ctx, project); err != nil {
		return status, fmt.Errorf("error ensuring project admin permissions: %w", err)
	}

	if err = r.ensureDefaultProjectRolesFn(ctx, project); err != nil {
		return status, fmt.Errorf("error ensuring default project roles: %w", err)
	}

	status.Phase = kargoapi.ProjectPhaseReady
	return status, nil
}

func (r *reconciler) ensureNamespace(
	ctx context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	status := *project.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project": project.Name,
	})

	ownerRef := metav1.NewControllerRef(
		project,
		kargoapi.GroupVersion.WithKind("Project"),
	)
	ownerRef.BlockOwnerDeletion = ptr.To(false)

	ns := &corev1.Namespace{}
	err := r.getNamespaceFn(
		ctx,
		types.NamespacedName{Name: project.Name},
		ns,
	)
	if err == nil {
		// We found an existing namespace with the same name as the Project.
		for _, ownerRef := range ns.OwnerReferences {
			if ownerRef.UID == project.UID {
				logger.Debug("namespace exists and is owned by this Project")
				return status, nil
			}
		}
		if ns.Labels != nil &&
			ns.Labels[kargoapi.ProjectLabelKey] == kargoapi.LabelTrueValue &&
			len(ns.OwnerReferences) == 0 {
			logger.Debug(
				"namespace exists, but is not owned by this Project, but has the " +
					"project label; Project will adopt it",
			)
			ns.OwnerReferences = []metav1.OwnerReference{*ownerRef}
			controllerutil.AddFinalizer(ns, kargoapi.FinalizerName)
			if err = r.updateNamespaceFn(ctx, ns); err != nil {
				return status, fmt.Errorf("error updating namespace %q: %w", project.Name, err)
			}
			logger.Debug("updated namespace with Project as owner")
			return status, nil
		}
		status.Phase = kargoapi.ProjectPhaseInitializationFailed
		return status, fmt.Errorf(
			"failed to initialize Project %q because namespace %q already exists",
			project.Name,
			project.Name,
		)
	}
	if !kubeerr.IsNotFound(err) {
		return status, fmt.Errorf("error getting namespace %q: %w", project.Name, err)
	}

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
		return status, fmt.Errorf("error creating namespace %q: %w", project.Name, err)
	}
	logger.Debug("created namespace")

	return status, nil
}

func (r *reconciler) ensureAPIAdminPermissions(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	const roleBindingName = "kargo-project-admin"

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project":     project.Name,
		"name":        project.Name,
		"namespace":   project.Name,
		"roleBinding": roleBindingName,
	})

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

func (r *reconciler) ensureDefaultProjectRoles(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project":   project.Name,
		"name":      project.Name,
		"namespace": project.Name,
	})

	const adminRoleName = "kargo-admin"
	const viewerRoleName = "kargo-viewer"
	allRoles := []string{adminRoleName, viewerRoleName}

	for _, saName := range allRoles {
		saLogger := logger.WithField("serviceAccount", saName)
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
					Resources: []string{"freights", "stages", "warehouses"},
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
					Resources: []string{"freights", "promotions", "stages", "warehouses"},
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
		roleLogger := logger.WithField("role", role.Name)
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
		roleLogger.Debugf(
			"created Role %q in project namespace %q", role.Name, project.Name,
		)
	}

	for _, rbName := range allRoles {
		rbLogger := logger.WithField("roleBinding", rbName)
		if err := r.createRoleBindingFn(
			ctx,
			&rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rbName,
					Namespace: project.Name,
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
		rbLogger.Debugf(
			"created RoleBinding %q in project namespace %q", rbName, project.Name,
		)
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
