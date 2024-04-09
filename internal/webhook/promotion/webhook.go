package promotion

import (
	"context"
	"fmt"

	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/record"
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

	recorder record.EventRecorder

	// The following behaviors are overridable for testing purposes:

	getFreightFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Freight, error)

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

	isRequestFromKargoControlplaneFn libWebhook.IsRequestFromKargoControlplaneFn
}

func SetupWebhookWithManager(
	cfg libWebhook.Config,
	mgr ctrl.Manager,
) error {
	w := newWebhook(
		cfg,
		mgr.GetClient(),
		mgr.GetEventRecorderFor("promotion-webhook"),
	)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Promotion{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

func newWebhook(
	cfg libWebhook.Config,
	kubeClient client.Client,
	recorder record.EventRecorder,
) *webhook {
	w := &webhook{
		client:   kubeClient,
		recorder: recorder,
	}
	w.getFreightFn = kargoapi.GetFreight
	w.getStageFn = kargoapi.GetStage
	w.validateProjectFn = libWebhook.ValidateProject
	w.authorizeFn = w.authorize
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.createSubjectAccessReviewFn = w.client.Create
	w.isRequestFromKargoControlplaneFn =
		libWebhook.IsRequestFromKargoControlplane(cfg.ControlplaneUserRegex)
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
		return fmt.Errorf(
			"error finding Stage %q in namespace %q: %w",
			promo.Spec.Stage,
			promo.Namespace,
			err,
		)
	}
	if stage == nil {
		return fmt.Errorf(
			"could not find Stage %q in namespace %q",
			promo.Spec.Stage,
			promo.Namespace,
		)
	}

	// Make sure the Promotion has the same shard as the Stage
	if stage.Spec.Shard != "" {
		if promo.Labels == nil {
			promo.Labels = make(map[string]string, 1)
		}
		promo.Labels[kargoapi.ShardLabelKey] = stage.Spec.Shard
	} else {
		delete(promo.Labels, kargoapi.ShardLabelKey)
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

	if err := w.authorizeFn(ctx, promo, "create"); err != nil {
		return nil, err
	}

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("get admission request from context: %w", err)
	}

	// Record Promotion created event if the request doesn't come from Kargo controlplane
	if !w.isRequestFromKargoControlplaneFn(req) {
		freight, err := w.getFreightFn(ctx, w.client, types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Freight,
		})
		if err != nil {
			return nil, fmt.Errorf("get freight: %w", err)
		}
		w.recordPromotionCreatedEvent(req, promo, freight)
	}
	return nil, nil
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
			fmt.Errorf(
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
			fmt.Errorf(
				"error creating SubjectAccessReview; refusing to %s Promotion",
				action,
			),
		)
	}

	if !accessReview.Status.Allowed {
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			fmt.Errorf(
				"subject %q is not permitted to %s Promotions for Stage %q",
				req.UserInfo.Username,
				action,
				promo.Spec.Stage,
			),
		)
	}

	return nil
}

func (w *webhook) recordPromotionCreatedEvent(
	req admission.Request,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) {
	actor := kargoapi.FormatEventKubernetesUserActor(req.UserInfo)
	w.recorder.AnnotatedEventf(
		p,
		kargoapi.NewPromotionCreatedEventAnnotations(actor, p, f),
		corev1.EventTypeNormal,
		kargoapi.EventReasonPromotionCreated,
		"Promotion created for Stage %q by %q",
		p.Spec.Stage,
		actor,
	)
}
