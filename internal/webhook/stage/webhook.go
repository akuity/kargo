package stage

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

var (
	stageGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "Stage",
	}
)

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	validateProjectFn func(
		context.Context,
		client.Client,
		schema.GroupKind,
		client.Object,
	) error

	validateCreateOrUpdateFn func(*kargoapi.Stage) (admission.Warnings, error)

	validateSpecFn func(*field.Path, *kargoapi.StageSpec) field.ErrorList
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Stage{}).
		WithValidator(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	w := &webhook{
		client: kubeClient,
	}
	w.validateProjectFn = libWebhook.ValidateProject
	w.validateCreateOrUpdateFn = w.validateCreateOrUpdate
	w.validateSpecFn = w.validateSpec
	return w
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	if err :=
		w.validateProjectFn(ctx, w.client, stageGroupKind, stage); err != nil {
		return nil, err
	}
	return w.validateCreateOrUpdateFn(stage)
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	stage := newObj.(*kargoapi.Stage) // nolint: forcetypeassert
	return w.validateCreateOrUpdateFn(stage)
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	// No-op
	return nil, nil
}

func (w *webhook) validateCreateOrUpdate(
	s *kargoapi.Stage,
) (admission.Warnings, error) {
	if errs := w.validateSpecFn(field.NewPath("spec"), s.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(stageGroupKind, s.Name, errs)
	}
	return nil, nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec *kargoapi.StageSpec,
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
	subs *kargoapi.Subscriptions,
) field.ErrorList {
	if subs == nil { // nil subs is caught by declarative validations
		return nil
	}
	// Can subscribe to Warehouse XOR upstream Stages
	if (subs.Warehouse == "" && len(subs.UpstreamStages) == 0) ||
		(subs.Warehouse != "" && len(subs.UpstreamStages) > 0) {
		return field.ErrorList{
			field.Invalid(
				f,
				subs,
				fmt.Sprintf(
					"exactly one of %s.warehouse or %s.upstreamStages must be defined",
					f.String(),
					f.String(),
				),
			),
		}
	}
	return nil
}

func (w *webhook) validatePromotionMechanisms(
	f *field.Path,
	promoMechs *kargoapi.PromotionMechanisms,
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
	updates []kargoapi.GitRepoUpdate,
) field.ErrorList {
	var errs field.ErrorList
	for i, update := range updates {
		errs = append(errs, w.validateGitRepoUpdate(f.Index(i), update)...)
	}
	return errs
}

func (w *webhook) validateGitRepoUpdate(
	f *field.Path,
	update kargoapi.GitRepoUpdate,
) field.ErrorList {
	var count int
	if update.Render != nil {
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
					"no more than one of %s.render, or %s.kustomize, or %s.helm may "+
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
	promoMech *kargoapi.HelmPromotionMechanism,
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
