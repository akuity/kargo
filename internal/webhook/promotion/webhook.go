package promotion

import (
	"context"

	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/validation"
	"github.com/akuity/kargo/internal/logging"
)

var (
	promotionGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "Promotion",
	}
	promotionGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "Promotion",
	}
)

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	authorizeFn func(
		ctx context.Context,
		promo *kargoapi.Promotion,
		action string,
	) error

	admissionRequestFromContextFn func(context.Context) (admission.Request, error)

	createSubjectAccessReviewFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	validateProjectFn func(context.Context, *kargoapi.Promotion) error
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	w.authorizeFn = w.authorize
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.createSubjectAccessReviewFn = w.client.Create
	w.validateProjectFn = w.validateProject
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Promotion{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) error {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err := w.validateProjectFn(ctx, promo); err != nil {
		return err
	}
	return w.authorizeFn(ctx, promo, "create")
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	promo := newObj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err := w.validateProjectFn(ctx, promo); err != nil {
		return err
	}
	if err := w.authorizeFn(ctx, promo, "update"); err != nil {
		return err
	}

	// PromotionSpecs are meant to be immutable
	if *promo.Spec != *(oldObj.(*kargoapi.Promotion).Spec) { // nolint: forcetypeassert
		return apierrors.NewInvalid(
			promotionGroupKind,
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
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err := w.validateProjectFn(ctx, promo); err != nil {
		return err
	}
	return w.authorizeFn(ctx, promo, "delete")
}

func (w *webhook) authorize(
	ctx context.Context,
	promo *kargoapi.Promotion,
	action string,
) error {
	logger := logging.LoggerFromContext(ctx)

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			promotionGroupResource,
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
			User:   req.UserInfo.Username,
			Groups: req.UserInfo.Groups,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     kargoapi.GroupVersion.Group,
				Resource:  "stages",
				Name:      promo.Spec.Stage,
				Verb:      "promote",
				Namespace: promo.Namespace,
			},
		},
	}
	if err := w.createSubjectAccessReviewFn(ctx, accessReview); err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			errors.Errorf(
				"error creating SubjectAccessReview; refusing to %s Promotion",
				action,
			),
		)
	}

	if !accessReview.Status.Allowed {
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			errors.Errorf(
				"subject %q is not permitted to %s Promotions for Stage %q",
				req.UserInfo.Username,
				action,
				promo.Spec.Stage,
			),
		)
	}

	return nil
}

func (w *webhook) validateProject(ctx context.Context, promo *kargoapi.Promotion) error {
	if err := validation.ValidateProject(ctx, w.client, promo.GetNamespace()); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return apierrors.NewNotFound(schema.GroupResource{
				Group:    corev1.SchemeGroupVersion.Group,
				Resource: "Namespace",
			}, promo.GetNamespace())
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return apierrors.NewInvalid(promotionGroupKind, promo.GetName(), field.ErrorList{fieldErr})
		}
		return apierrors.NewInternalError(err)
	}
	return nil
}
