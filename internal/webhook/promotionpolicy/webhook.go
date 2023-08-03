package promotionpolicy

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/validation"
)

var (
	promotionPolicyGroupKind = schema.GroupKind{
		Group: v1alpha1.GroupVersion.Group,
		Kind:  "PromotionPolicy",
	}
	promotionPolicyGroupResource = schema.GroupResource{
		Group:    v1alpha1.GroupVersion.Group,
		Resource: "PromotionPolicy",
	}
)

type webhook struct {
	client client.Client
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.PromotionPolicy{}).
		WithValidator(&webhook{
			client: mgr.GetClient(),
		}).
		Complete()
}

func (w *webhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	policy := obj.(*v1alpha1.PromotionPolicy) // nolint: forcetypeassert
	if err := w.validateProject(ctx, policy); err != nil {
		return err
	}
	return w.validateStageUniqueness(ctx, policy)
}

func (w *webhook) ValidateUpdate(ctx context.Context, _ runtime.Object, newObj runtime.Object) error {
	policy := newObj.(*v1alpha1.PromotionPolicy) // nolint: forcetypeassert
	if err := w.validateProject(ctx, policy); err != nil {
		return err
	}
	return w.validateStageUniqueness(ctx, policy)
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) error {
	policy := obj.(*v1alpha1.PromotionPolicy) // nolint: forcetypeassert
	return w.validateProject(ctx, policy)
}

func (w *webhook) validateProject(ctx context.Context, policy *v1alpha1.PromotionPolicy) error {
	if err := validation.ValidateProject(ctx, w.client, policy.GetNamespace()); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return apierrors.NewNotFound(schema.GroupResource{
				Group:    corev1.SchemeGroupVersion.Group,
				Resource: "Namespace",
			}, policy.GetNamespace())
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return apierrors.NewInvalid(promotionPolicyGroupKind, policy.GetName(), field.ErrorList{fieldErr})
		}
		return apierrors.NewInternalError(err)
	}
	return nil
}

func (w *webhook) validateStageUniqueness(ctx context.Context, policy *v1alpha1.PromotionPolicy) error {
	var list v1alpha1.PromotionPolicyList
	if err := w.client.List(ctx, &list); err != nil {
		return apierrors.NewInternalError(errors.Wrap(err, "list promotion policies"))
	}
	for _, ep := range list.Items {
		if policy.Stage == ep.Stage {
			return apierrors.NewConflict(promotionPolicyGroupResource, policy.GetName(),
				fmt.Errorf("policy for stage %q is already exists: %s", policy.Stage, ep.GetName()))
		}
	}
	return nil
}
