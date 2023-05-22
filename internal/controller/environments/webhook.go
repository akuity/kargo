package environments

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/akuity/kargo/api/v1alpha1"
)

type webhook struct{}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Environment{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

func (w *webhook) Default(_ context.Context, obj runtime.Object) error {
	env := obj.(*api.Environment) // nolint: forcetypeassert
	// Note that defaults are applied BEFORE validation, so we do not have the
	// luxury of assuming certain required fields must be non-nil.
	if env.Spec != nil {

		if env.Spec.Subscriptions != nil {
			// Default namespace for Environments we subscribe to
			for i := range env.Spec.Subscriptions.UpstreamEnvs {
				if env.Spec.Subscriptions.UpstreamEnvs[i].Namespace == "" {
					env.Spec.Subscriptions.UpstreamEnvs[i].Namespace = env.Namespace
				}
			}
		}

		if env.Spec.PromotionMechanisms != nil {
			// Default namespace for Argo CD Applications we update
			for i := range env.Spec.PromotionMechanisms.ArgoCDAppUpdates {
				if env.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace == "" {
					env.Spec.PromotionMechanisms.ArgoCDAppUpdates[i].AppNamespace =
						env.Namespace
				}
			}
		}

	}

	return nil
}

func (w *webhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	// nolint: forcetypeassert
	return w.validateCreateOrUpdate(obj.(*api.Environment))
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) error {
	// nolint: forcetypeassert
	return w.validateCreateOrUpdate(newObj.(*api.Environment))
}

func (w *webhook) ValidateDelete(context.Context, runtime.Object) error {
	// Nothing to validate upon delete
	return nil
}

func (w *webhook) validateCreateOrUpdate(e *api.Environment) error {
	if errs := w.validateSpec(field.NewPath("spec"), e.Spec); len(errs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: api.GroupVersion.Group,
				Kind:  "Environment",
			},
			e.Name,
			errs,
		)
	}
	return nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec *api.EnvironmentSpec,
) field.ErrorList {
	if spec == nil { // nil spec is caught by declarative validations
		return nil
	}
	errs := w.validateSubs(f.Child("subscriptions"), spec.Subscriptions)
	return append(
		errs,
		w.validatePromotionMechanisms(
			f.Child("promotionMechanisms"),
			spec.PromotionMechanisms)...,
	)
}

func (w *webhook) validateSubs(
	f *field.Path,
	subs *api.Subscriptions,
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
	return w.validateRepoSubs(f.Child("repos"), subs.Repos)
}

func (w *webhook) validateRepoSubs(
	f *field.Path,
	subs *api.RepoSubscriptions,
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
	errs := w.validateImageSubs(f.Child("images"), subs.Images)
	return append(errs, w.validateChartSubs(f.Child("charts"), subs.Charts)...)
}

func (w *webhook) validateImageSubs(
	f *field.Path,
	subs []api.ImageSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, w.validateImageSub(f.Index(i), sub)...)
	}
	return errs
}

func (w *webhook) validateImageSub(
	f *field.Path,
	sub api.ImageSubscription,
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

func (w *webhook) validateChartSubs(
	f *field.Path,
	subs []api.ChartSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, w.validateChartSub(f.Index(i), sub)...)
	}
	return errs
}

func (w *webhook) validateChartSub(
	f *field.Path,
	sub api.ChartSubscription,
) field.ErrorList {
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		return field.ErrorList{err}
	}
	return nil
}

func (w *webhook) validatePromotionMechanisms(
	f *field.Path,
	promoMechs *api.PromotionMechanisms,
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
	return w.validateGitRepoUpdates(
		f.Child("gitRepoUpdates"),
		promoMechs.GitRepoUpdates,
	)
}

func (w *webhook) validateGitRepoUpdates(
	f *field.Path,
	updates []api.GitRepoUpdate,
) field.ErrorList {
	var errs field.ErrorList
	for i, update := range updates {
		errs = append(errs, w.validateGitRepoUpdate(f.Index(i), update)...)
	}
	return errs
}

func (w *webhook) validateGitRepoUpdate(
	f *field.Path,
	update api.GitRepoUpdate,
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
	if count > 1 {
		return field.ErrorList{
			field.Invalid(
				f,
				update,
				fmt.Sprintf(
					"no more than one of %s.bookkeeper, or %s.kustomize, or %s.helm may "+
						"be defined",
					f.String(),
					f.String(),
					f.String(),
				),
			),
		}
	}
	return w.validateHelmPromotionMechanism(f.Child("helm"), update.Helm)
}

func (w *webhook) validateHelmPromotionMechanism(
	f *field.Path,
	promoMech *api.HelmPromotionMechanism,
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
