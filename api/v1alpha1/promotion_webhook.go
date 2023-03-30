package v1alpha1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (p *Promotion) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(p).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered
// for the type
func (p *Promotion) ValidateCreate() error {
	// Nothing to validate upon create
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered
// for the type
func (p *Promotion) ValidateUpdate(old runtime.Object) error {
	// PromotionSpecs are meant to be immutable
	if *p.Spec != *(old.(*Promotion).Spec) {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
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

// ValidateDelete implements webhook.Validator so a webhook will be registered
// for the type
func (p *Promotion) ValidateDelete() error {
	// Nothing to validate upon delete
	return nil
}
