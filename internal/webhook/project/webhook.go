package project

import (
	"context"

	"github.com/containerd/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

var projectGroupResource = schema.GroupResource{
	Group:    kargoapi.GroupVersion.Group,
	Resource: "projects",
}

type webhook struct {
	// The following behaviors are overridable for testing purposes:

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
		getNamespaceFn:    kubeClient.Get,
		createNamespaceFn: kubeClient.Create,
	}
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(
			errors.Wrap(err, "error getting admission request from context"),
		)
	}
	if req.DryRun == nil || !*req.DryRun {
		project := obj.(*kargoapi.Project) // nolint: forcetypeassert

		logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"project": project.Name,
			"name":    project.Name,
		})

		// We handle creation of a Project's associated namespace synchronously in
		// this webhook so that the namespace is guaranteed to exist before
		// other resources (appearing below the Project) in a manifest will not fail
		// to create due to the namespace not existing yet.

		// There is no guarantee that just because this is a create request that
		// there wasn't a previous attempt to create this Project that failed. If
		// that is the case, the namespace may already exist.
		ns := &corev1.Namespace{}
		if err = w.getNamespaceFn(
			ctx,
			types.NamespacedName{Name: project.Name},
			ns,
		); err == nil {
			// We found an existing namespace with the same name as the Project. If it's
			// owned by this Project then it was created on a previous attempt to
			// reconcile this Project, but otherwise, this is a problem.
			for _, ownerRef := range ns.OwnerReferences {
				if ownerRef.UID == project.UID {
					logger.Debug("namespace exists and is owned by this Project")
					return nil, nil
				}
			}
			return nil, apierrors.NewConflict(
				projectGroupResource,
				project.Name,
				errors.Errorf(
					"failed to initialize Project %q because namespace %q already exists",
					project.Name,
					project.Name,
				),
			)
		}
		if !apierrors.IsNotFound(err) {
			return nil, apierrors.NewInternalError(
				errors.Wrapf(err, "error getting namespace %q", project.Name),
			)
		}

		logger.Debug("namespace does not exist yet; creating namespace")

		ownerRef := metav1.NewControllerRef(
			project,
			kargoapi.GroupVersion.WithKind("Project"),
		)
		ownerRef.BlockOwnerDeletion = ptr.To(false)
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: project.Name,
				Labels: map[string]string{
					kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
				},
				OwnerReferences: []metav1.OwnerReference{*ownerRef},
			},
		}
		// Project namespaces are owned by a Project. Deleting a Project
		// automatically deletes the namespace. But we also want this to work in the
		// other direction, where that behavior is not automatic. We add a finalizer
		// to the namespace and use our own namespace reconciler to clear it after
		// deleting the Project.
		controllerutil.AddFinalizer(ns, kargoapi.FinalizerName)
		if err := w.createNamespaceFn(ctx, ns); err != nil {
			return nil, apierrors.NewInternalError(
				errors.Wrapf(err, "error creating namespace %q", project.Name),
			)
		}
		logger.Debug("created namespace")
	}
	return nil, nil
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
