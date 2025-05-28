package project

import (
	"context"
	"fmt"
	"reflect"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/logging"
)

var (
	projectGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "Project",
	}
	projectGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "projects",
	}
)

type WebhookConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" required:"true"`
}

func WebhookConfigFromEnv() WebhookConfig {
	cfg := WebhookConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type webhook struct {
	cfg    WebhookConfig
	client client.Client
}

func SetupWebhookWithManager(mgr ctrl.Manager, cfg WebhookConfig) error {
	w := &webhook{
		cfg:    cfg,
		client: mgr.GetClient(),
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Project{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	project := obj.(*kargoapi.Project) // nolint: forcetypeassert

	// TODO(hidde): Remove this when the deprecated Spec field is removed.
	var warnings admission.Warnings
	if project.Spec != nil { // nolint: staticcheck
		warnings = append(warnings, fmt.Sprintf(
			"ProjectSpec is deprecated: create a ProjectConfig named %q with the config instead",
			project.Name,
		))
	}

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(
			fmt.Errorf("error getting admission request from context: %w", err),
		)
	}

	if req.DryRun != nil && *req.DryRun {
		return warnings, nil
	}

	// We synchronously ensure the existence of a namespace with the same name as
	// the Project because resources following the Project in a manifest are
	// likely to be scoped to that namespace.
	if err = w.ensureNamespace(ctx, project); err != nil {
		return nil, err
	}

	// Ensure the Kargo API server and kargo-admin ServiceAccount receive
	// permissions to manage ServiceAccounts, Roles, RoleBindings, and Secrets in
	// the Project namespace just in time. This prevents us from having to give
	// the Kargo API server carte blanche access these resources throughout the
	// cluster. We do this synchronously because resources of these types are
	// likely to follow the Project in a manifest.
	return warnings, w.ensureProjectAdminPermissions(ctx, project)
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	oldProject := oldObj.(*kargoapi.Project) // nolint: forcetypeassert
	newProject := newObj.(*kargoapi.Project) // nolint: forcetypeassert

	specPath := field.NewPath("spec")

	// TODO(hidde): Remove this when the deprecated Spec field is removed.
	var warnings admission.Warnings
	if newProject.Spec != nil { // nolint: staticcheck
		if api.HasMigrationAnnotationValue(newProject, api.MigratedProjectSpecToProjectConfig) {
			if !reflect.DeepEqual(oldProject.Spec, newProject.Spec) { // nolint: staticcheck
				return nil, apierrors.NewInvalid(
					projectGroupKind,
					newProject.Name,
					field.ErrorList{
						field.Forbidden(
							specPath,
							fmt.Sprintf(
								"deprecated field: create a ProjectConfig named %q with the config instead",
								newProject.Name,
							),
						),
					},
				)
			}
		}

		// Warn the user that the ProjectSpec is deprecated.
		warnings = append(warnings, fmt.Sprintf(
			"ProjectSpec is deprecated: create a ProjectConfig named %q with the config instead", newProject.Name,
		))
	}

	return warnings, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}

// ensureNamespace is used to ensure the existence of a namespace with the same
// name as the Project. If the namespace does not exist, it is created. If the
// namespace exists, it is checked for any ownership conflicts with the Project
// and will return an error if any are found.
func (w *webhook) ensureNamespace(
	ctx context.Context,
	project *kargoapi.Project,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project.Name)

	ns := &corev1.Namespace{}
	if err := w.client.Get(
		ctx,
		types.NamespacedName{Name: project.Name},
		ns,
	); err != nil && !apierrors.IsNotFound(err) {
		return apierrors.NewInternalError(
			fmt.Errorf("error getting namespace %q: %w", project.Name, err),
		)
	} else if err == nil {
		// We found an existing namespace with the same name as the Project. It's
		// only a problem if it is not labeled as a Project namespace.
		if ns.Labels[kargoapi.ProjectLabelKey] != kargoapi.LabelTrueValue {
			return apierrors.NewConflict(
				projectGroupResource,
				project.Name,
				fmt.Errorf(
					"failed to initialize Project %q because namespace %q already "+
						"exists and is not labeled as a Project namespace",
					project.Name,
					project.Name,
				),
			)
		}
		logger.Debug("namespace exists but no conflict was found")
		return nil
	}

	// If we get to here, we had a not found error, and we can proceed with
	// creating the namespace.

	logger.Debug("namespace does not exist; creating it")

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name,
			Labels: map[string]string{
				kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
			},
			// Note: We no longer use an owner reference here. If we did, and too
			// much time were to pass between namespace creation and the completion of
			// this webhook, Kubernetes would notice the absence of the owner, mistake
			// the namespace for an orphan, and delete it. We do still want the
			// Project to own the namespace, but we rely on the Project reconciler in
			// the management controller to establish that relationship
			// asynchronously.
		},
	}
	// Project namespaces are owned by a Project. Deleting a Project
	// automatically deletes the namespace. But we also want this to work in the
	// other direction, where that behavior is not automatic. We add a finalizer
	// to the namespace and use our own namespace reconciler to clear it after
	// deleting the Project.
	controllerutil.AddFinalizer(ns, kargoapi.FinalizerName)
	if err := w.client.Create(ctx, ns); err != nil {
		return apierrors.NewInternalError(
			fmt.Errorf("error creating namespace %q: %w", project.Name, err),
		)
	}
	logger.Debug("created namespace")

	return nil
}

func (w *webhook) ensureProjectAdminPermissions(
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
				Namespace: w.cfg.KargoNamespace,
			},
			{
				Kind:      "ServiceAccount",
				Name:      "kargo-admin",
				Namespace: w.cfg.KargoNamespace,
			},
		},
	}
	if err := w.client.Create(ctx, roleBinding); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debug("role binding already exists in project namespace")
			return nil
		}
		return apierrors.NewInternalError(
			fmt.Errorf(
				"error creating role binding %q in project namespace %q: %w",
				roleBinding.Name,
				project.Name,
				err,
			),
		)
	}
	logger.Debug("granted API server and kargo-admin project admin permissions")

	return nil
}
