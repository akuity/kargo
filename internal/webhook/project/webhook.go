package project

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var projectGroupResource = schema.GroupResource{
	Group:    kargoapi.GroupVersion.Group,
	Resource: "projects",
}

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	getNamespaceFn func(
		context.Context,
		types.NamespacedName,
		client.Object,
		...client.GetOption,
	) error
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Project{}).
		WithValidator(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	return &webhook{
		client:         kubeClient,
		getNamespaceFn: kubeClient.Get,
	}
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	project := obj.(*kargoapi.Project) // nolint: forcetypeassert
	// Validate that a namespace matching the project name doesn't already exist.
	namespace := &corev1.Namespace{}
	if err := w.getNamespaceFn(
		ctx,
		client.ObjectKey{Name: project.Name},
		namespace,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// This is good. The namespace doesn't already exist.
			return nil, nil
		}
		return nil, apierrors.NewInternalError(
			errors.Wrapf(err, "error getting namespace %q", project.Name),
		)
	}
	return nil, apierrors.NewConflict(
		projectGroupResource,
		project.Name,
		errors.Errorf(
			"cannot create Project %q because namespace %q already exists",
			project.Name,
			project.Name,
		),
	)
}

func (w *webhook) ValidateUpdate(
	context.Context,
	runtime.Object,
	runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}
