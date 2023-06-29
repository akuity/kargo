package promotions

import (
	"context"

	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	authorizeFn func(
		ctx context.Context,
		promo *api.Promotion,
		action string,
	) error

	admissionRequestFromContextFn func(context.Context) (admission.Request, error)

	createSubjectAccessReviewFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	w.authorizeFn = w.authorize
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.createSubjectAccessReviewFn = w.client.Create
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Promotion{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) error {
	// nolint: forcetypeassert
	return w.authorizeFn(ctx, obj.(*api.Promotion), "create")
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	promo := newObj.(*api.Promotion) // nolint: forcetypeassert

	if err := w.authorizeFn(ctx, promo, "update"); err != nil {
		return err
	}

	// PromotionSpecs are meant to be immutable
	if *promo.Spec != *(oldObj.(*api.Promotion).Spec) { // nolint: forcetypeassert
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: api.GroupVersion.Group,
				Kind:  "Promotion",
			},
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec"),
					promo.Spec,
					"spec is immutable",
				),
			},
		)
	}
	return nil
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) error {
	logger := logging.LoggerFromContext(ctx)

	promo := obj.(*api.Promotion) // nolint: forcetypeassert

	// Special logic for delete only. Allow any delete by the Kubernetes namespace
	// controller. This prevents the webhook from stopping a namespace from being
	// cleaned up.
	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			schema.GroupResource{
				Group:    api.GroupVersion.Group,
				Resource: "Promotion",
			},
			promo.Name,
			errors.New(
				"error retrieving admission request from context; refusing to "+
					"delete Promotion",
			),
		)
	}
	if req.UserInfo.Username == "system:serviceaccount:kube-system:namespace-controller" {
		return nil
	}

	return w.authorizeFn(ctx, promo, "delete")
}

func (w *webhook) authorize(
	ctx context.Context,
	promo *api.Promotion,
	action string,
) error {
	logger := logging.LoggerFromContext(ctx)

	groupResource := schema.GroupResource{
		Group:    api.GroupVersion.Group,
		Resource: "Promotion",
	}

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"error retrieving admission request from context; refusing to "+
					"%s Promotion",
				action,
			),
		)
	}

	accessReview := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			User: req.UserInfo.Username,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     api.GroupVersion.Group,
				Resource:  "environments",
				Name:      promo.Spec.Environment,
				Verb:      "promote",
				Namespace: promo.Namespace,
			},
		},
	}
	if err := w.createSubjectAccessReviewFn(ctx, accessReview); err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"error creating SubjectAccessReview; refusing to %s Promotion",
				action,
			),
		)
	}

	if !accessReview.Status.Allowed {
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"subject %q is not permitted to %s Promotions for Environment %q",
				req.UserInfo.Username,
				action,
				promo.Spec.Environment,
			),
		)
	}

	return nil
}
