package projectconfig

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
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
)

var (
	projectConfigGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "ProjectConfig",
	}
	projectConfigGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "projectconfigs",
	}
)

type webhook struct {
	client client.Client
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.ProjectConfig{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	projectCfg := obj.(*kargoapi.ProjectConfig) // nolint: forcetypeassert

	var errs field.ErrorList
	if metaErrs := w.validateObjectMeta(
		field.NewPath("metadata"),
		projectCfg.ObjectMeta,
	); len(metaErrs) > 0 {
		errs = append(errs, metaErrs...)
	}

	if specErrs := w.validateSpec(
		field.NewPath("spec"),
		projectCfg.Spec,
	); len(specErrs) > 0 {
		errs = append(errs, specErrs...)
	}

	if len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			projectConfigGroupKind,
			projectCfg.Name,
			errs,
		)
	}

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(
			fmt.Errorf("error getting admission request from context: %w", err),
		)
	}

	if req.DryRun != nil && *req.DryRun {
		return nil, nil
	}

	if err = w.ensureProjectNamespace(ctx, projectCfg.ObjectMeta); err != nil {
		return nil, err
	}

	return nil, nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	projectCfg := newObj.(*kargoapi.ProjectConfig) // nolint: forcetypeassert
	if errs := w.validateSpec(field.NewPath("spec"), projectCfg.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(projectConfigGroupKind, projectCfg.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}

func (w *webhook) validateObjectMeta(
	f *field.Path,
	meta metav1.ObjectMeta,
) field.ErrorList {
	if meta.Name != meta.Namespace {
		return field.ErrorList{
			field.Invalid(
				f.Child("name"),
				meta.Name,
				fmt.Sprintf(
					"name %q must match project name %q",
					meta.Name,
					meta.Namespace,
				),
			),
		}
	}
	return nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec kargoapi.ProjectConfigSpec,
) field.ErrorList {
	return w.validatePromotionPolicies(
		f.Child("promotionPolicies"),
		spec.PromotionPolicies,
	)
}

func (w *webhook) validatePromotionPolicies(
	f *field.Path,
	promotionPolicies []kargoapi.PromotionPolicy,
) field.ErrorList {
	stageNames := make(map[string]struct{}, len(promotionPolicies))
	for _, promotionPolicy := range promotionPolicies {
		if _, found := stageNames[promotionPolicy.Stage]; found {
			return field.ErrorList{
				field.Invalid(
					f,
					promotionPolicies,
					fmt.Sprintf(
						"multiple %s reference stage %q",
						f.String(),
						promotionPolicy.Stage,
					),
				),
			}
		}
		stageNames[promotionPolicy.Stage] = struct{}{}
	}
	return nil
}

func (w *webhook) ensureProjectNamespace(ctx context.Context, meta metav1.ObjectMeta) error {
	ns := &corev1.Namespace{}
	if err := w.client.Get(ctx, types.NamespacedName{Name: meta.Namespace}, ns); err != nil {
		return apierrors.NewInternalError(
			fmt.Errorf("error getting namespace %q: %w", meta.Namespace, err),
		)
	}

	v, ok := ns.Labels[kargoapi.ProjectLabelKey]
	if !ok || v != kargoapi.LabelTrueValue {
		return apierrors.NewForbidden(
			projectConfigGroupResource,
			meta.Name,
			fmt.Errorf(
				"namespace %q does not belong to Kargo project (missing %q label)",
				meta.Namespace, kargoapi.ProjectLabelKey,
			),
		)
	}

	return nil
}
