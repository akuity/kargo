package promotion

import (
	"context"

	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	libWebhook "github.com/akuity/kargo/internal/webhook"
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

	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)

	validateProjectFn func(
		context.Context,
		client.Client,
		schema.GroupKind,
		client.Object,
	) error

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
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Promotion{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	w := &webhook{
		client: kubeClient,
	}
	w.getStageFn = kargoapi.GetStage
	w.validateProjectFn = libWebhook.ValidateProject
	w.authorizeFn = w.authorize
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.createSubjectAccessReviewFn = w.client.Create
	return w
}

func (w *webhook) Default(ctx context.Context, obj runtime.Object) error {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	stage, err := w.getStageFn(
		ctx,
		w.client,
		types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Stage %q in namespace %q",
			promo.Spec.Stage,
			promo.Namespace,
		)
	}
	if stage == nil {
		return errors.Errorf(
			"could not find Stage %q in namespace %q",
			promo.Spec.Stage,
			promo.Namespace,
		)
	}
	ownerRef :=
		metav1.NewControllerRef(stage, kargoapi.GroupVersion.WithKind("Stage"))
	promo.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	return nil
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err :=
		w.validateProjectFn(ctx, w.client, promotionGroupKind, promo); err != nil {
		return nil, err
	}
	return nil, w.authorizeFn(ctx, promo, "create")
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	promo := newObj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err := w.authorizeFn(ctx, promo, "update"); err != nil {
		return nil, err
	}

	// PromotionSpecs are meant to be immutable
	if *promo.Spec != *(oldObj.(*kargoapi.Promotion).Spec) { // nolint: forcetypeassert
		return nil, apierrors.NewInvalid(
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
	return nil, nil
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	return nil, w.authorizeFn(ctx, promo, "delete")
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
