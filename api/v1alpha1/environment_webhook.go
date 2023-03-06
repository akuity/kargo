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

// Default implements webhook.Defaulter so a webhook will be registered for the
// type
func (e *Environment) Default() {
	// Default namespace for Environments we subscribe to
	for i := range e.Spec.Subscriptions.UpstreamEnvs {
		if e.Spec.Subscriptions.UpstreamEnvs[i].Namespace == "" {
			e.Spec.Subscriptions.UpstreamEnvs[i].Namespace = e.Namespace
		}
	}

	// Default namespace for Argo CD Applications we update
	for i := range e.Spec.PromotionMechanisms.ArgoCDAppUpdates {
		if e.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace == "" {
			e.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace = e.Namespace
		}
	}

	// Default namespace for Argo CD Applications we check health of
	for i := range e.Spec.HealthChecks.ArgoCDAppChecks {
		if e.Spec.HealthChecks.ArgoCDAppChecks[i].AppNamespace == "" {
			e.Spec.HealthChecks.ArgoCDAppChecks[i].AppNamespace = e.Namespace
		}
	}
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
