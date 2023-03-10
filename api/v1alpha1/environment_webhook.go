package v1alpha1

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
	// Note that defaults are applied BEFORE validation, so we do not have the
	// luxury of assuming certain required fields must be non-nil.
	if e.Spec != nil {

		if e.Spec.Subscriptions != nil {
			// Default namespace for Environments we subscribe to
			for i := range e.Spec.Subscriptions.UpstreamEnvs {
				if e.Spec.Subscriptions.UpstreamEnvs[i].Namespace == "" {
					e.Spec.Subscriptions.UpstreamEnvs[i].Namespace = e.Namespace
				}
			}
		}

		if e.Spec.PromotionMechanisms != nil {
			// Default namespace for Argo CD Applications we update
			for i := range e.Spec.PromotionMechanisms.ArgoCDAppUpdates {
				if e.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace == "" {
					e.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace =
						e.Namespace
				}
			}
		}

		if e.Spec.HealthChecks != nil {
			// Default namespace for Argo CD Applications we check health of
			for i := range e.Spec.HealthChecks.ArgoCDAppChecks {
				if e.Spec.HealthChecks.ArgoCDAppChecks[i].AppNamespace == "" {
					e.Spec.HealthChecks.ArgoCDAppChecks[i].AppNamespace = e.Namespace
				}
			}
		}

	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateCreate() error {
	return e.validateCreateOrUpdate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateUpdate(old runtime.Object) error {
	return e.validateCreateOrUpdate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered
// for the type
func (e *Environment) ValidateDelete() error {
	// Nothing to validate upon delete
	return nil
}

func (e *Environment) validateCreateOrUpdate() error {
	if errs := e.validateSpec(field.NewPath("spec"), e.Spec); len(errs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "Environment",
			},
			e.Name,
			errs,
		)
	}
	return nil
}

func (e *Environment) validateSpec(
	f *field.Path,
	spec *EnvironmentSpec,
) field.ErrorList {
	if spec == nil { // nil spec is caught by declarative validations
		return nil
	}
	errs := e.validateSubs(f.Child("subscriptions"), spec.Subscriptions)
	return append(
		errs,
		e.validatePromotionMechanisms(
			f.Child("promotionMechanisms"),
			spec.PromotionMechanisms)...,
	)
}

func (e *Environment) validateSubs(
	f *field.Path,
	subs *Subscriptions,
) field.ErrorList {
	if subs == nil { // nil subs is caught by declarative validations
		return nil
	}
	// Can subscribe to repos XOR upstream Environments
	if (subs.Repos == nil && len(subs.UpstreamEnvs) == 0) ||
		(subs.Repos != nil && len(subs.UpstreamEnvs) > 0) {
		return field.ErrorList{
			field.Invalid(
				f,
				subs,
				fmt.Sprintf(
					"exactly one of %s.repos or %s.upstreamEnvs must be defined",
					f.String(),
					f.String(),
				),
			),
		}
	}
	return e.validateRepoSubs(f.Child("repos"), subs.Repos)
}

func (e *Environment) validateRepoSubs(
	f *field.Path,
	subs *RepoSubscriptions,
) field.ErrorList {
	if subs == nil {
		return nil
	}
	// Must subscribe to at least one repo of some sort
	if len(subs.Git) == 0 && len(subs.Images) == 0 && len(subs.Charts) == 0 {
		return field.ErrorList{
			field.Invalid(
				f,
				subs,
				fmt.Sprintf(
					"at least one of %s.git, %s.images, or %s.charts must be non-empty",
					f.String(),
					f.String(),
					f.String(),
				),
			),
		}
	}
	errs := e.validateImageSubs(f.Child("images"), subs.Images)
	return append(errs, e.validateChartSubs(f.Child("charts"), subs.Charts)...)
}

func (e *Environment) validateImageSubs(
	f *field.Path,
	subs []ImageSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, e.validateImageSub(f.Index(i), sub)...)
	}
	return errs
}

func (e *Environment) validateImageSub(
	f *field.Path,
	sub ImageSubscription,
) field.ErrorList {
	var errs field.ErrorList
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = field.ErrorList{err}
	}
	if sub.Platform != "" {
		if _, _, _, err := image.ParsePlatform(sub.Platform); err != nil {
			errs = append(errs, field.Invalid(f.Child("platform"), sub.Platform, ""))
		}
	}
	return errs
}

func (e *Environment) validateChartSubs(
	f *field.Path,
	subs []ChartSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, e.validateChartSub(f.Index(i), sub)...)
	}
	return errs
}

func (e *Environment) validateChartSub(
	f *field.Path,
	sub ChartSubscription,
) field.ErrorList {
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		return field.ErrorList{err}
	}
	return nil
}

func (e *Environment) validatePromotionMechanisms(
	f *field.Path,
	promoMechs *PromotionMechanisms,
) field.ErrorList {
	if promoMechs == nil { // nil promoMechs is caught by declarative validations
		return nil
	}
	// Must define at least one mechanism
	if len(promoMechs.GitRepoUpdates) == 0 &&
		len(promoMechs.ArgoCDAppUpdates) == 0 {
		return field.ErrorList{
			field.Invalid(
				f,
				promoMechs,
				fmt.Sprintf(
					"at least one of %s.gitRepoUpdates or %s.argoCDAppUpdates must "+
						"be non-empty",
					f.String(),
					f.String(),
				),
			),
		}
	}
	return e.validateGitRepoUpdates(
		f.Child("gitRepoUpdates"),
		promoMechs.GitRepoUpdates,
	)
}

func (e *Environment) validateGitRepoUpdates(
	f *field.Path,
	updates []GitRepoUpdate,
) field.ErrorList {
	var errs field.ErrorList
	for i, update := range updates {
		errs = append(errs, e.validateGitRepoUpdate(f.Index(i), update)...)
	}
	return errs
}

func (e *Environment) validateGitRepoUpdate(
	f *field.Path,
	update GitRepoUpdate,
) field.ErrorList {
	var count int
	if update.Bookkeeper != nil {
		count++
	}
	if update.Kustomize != nil {
		count++
	}
	if update.Helm != nil {
		count++
	}
	if count != 1 {
		return field.ErrorList{
			field.Invalid(
				f,
				update,
				fmt.Sprintf(
					"exactly one of %s.bookkeeper, or %s.kustomize, or %s.helm must "+
						"be defined",
					f.String(),
					f.String(),
					f.String(),
				),
			),
		}
	}
	return e.validateHelmPromotionMechanism(f.Child("helm"), update.Helm)
}

func (e *Environment) validateHelmPromotionMechanism(
	f *field.Path,
	promoMech *HelmPromotionMechanism,
) field.ErrorList {
	if promoMech == nil {
		return nil
	}
	// This mechanism must define at least one change to apply
	if len(promoMech.Images) == 0 && len(promoMech.Charts) == 0 {
		return field.ErrorList{
			field.Invalid(
				f,
				promoMech,
				fmt.Sprintf(
					"at least one of %s.images or %s.charts must be non-empty",
					f.String(),
					f.String(),
				),
			),
		}
	}
	return nil
}

func validateSemverConstraint(
	f *field.Path,
	semverConstraint string,
) *field.Error {
	if semverConstraint == "" {
		return nil
	}
	if _, err := semver.NewConstraint(semverConstraint); err != nil {
		return field.Invalid(f, semverConstraint, "")
	}
	return nil
}
