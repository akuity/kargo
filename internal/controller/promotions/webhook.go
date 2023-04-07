package promotions

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/akuity/kargo/api/v1alpha1"
)

type webhook struct{}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Promotion{}).
		WithValidator(&webhook{}).
		Complete()
}

func (w *webhook) ValidateCreate(context.Context, runtime.Object) error {
	// Nothing to validate upon create
	return nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	p := newObj.(*api.Promotion)
	// PromotionSpecs are meant to be immutable
	if *p.Spec != *(oldObj.(*api.Promotion).Spec) {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: api.GroupVersion.Group,
				Kind:  "Promotion",
			},
			p.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec"),
					p.Spec,
					"spec is immutable",
				),
			},
		)
	}
	return nil
}

func (w *webhook) ValidateDelete(context.Context, runtime.Object) error {
	// Nothing to validate upon delete
	return nil
}
