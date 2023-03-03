package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (e *Environment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(e).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateCreate() error {
	// TODO: Add validation
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateUpdate(old runtime.Object) error {
	// TODO: Add validation
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateDelete() error {
	// Nothing to validate upon delete
	return nil
}
