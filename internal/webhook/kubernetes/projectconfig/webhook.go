package projectconfig

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/akuity/kargo/internal/pattern"
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
	var errs field.ErrorList

	// Map of stage names to their indices in the promotionPolicies slice
	stageNames := make(map[string][]int)

	for i, policy := range promotionPolicies {
		stage := policy.Stage // nolint:staticcheck
		if policy.StageSelector != nil {
			stage = policy.StageSelector.Name
		}

		// Skip empty stage names
		if stage == "" {
			continue
		}

		// Handle patterns (a name containing an identifier, e.g. "glob:")
		stageParts := strings.SplitN(stage, ":", 2)
		if len(stageParts) > 1 {
			m, err := pattern.ParseNamePattern(stage)
			if err != nil {
				errs = append(errs, field.Invalid(
					// NB: Our validation rule only allows patterns to be set
					// on the stageSelector's name field. The deprecated stage
					// field has to adhere to Kubernetes naming rules.
					f.Index(i).Child("stageSelector").Child("name"),
					stage,
					err.Error(),
				))
				continue
			}

			// The behavior of the pattern parser is to fall back to an exact
			// matcher. However, if a ":" is present we do not expect this
			// fallback to ever happen.
			if _, isExact := m.(*pattern.ExactMatcher); isExact {
				errs = append(errs, field.Invalid(
					f.Index(i).Child("stageSelector").Child("name"),
					stage,
					fmt.Sprintf(`invalid pattern identifier %q: must be "regex", "regexp" or "glob"`, stageParts[0]),
				))
			}

			continue
		}

		// Track stages by name and their positions
		stageNames[stage] = append(stageNames[stage], i)
	}

	// Generate an error for each duplicate field
	for stage, indices := range stageNames {
		if len(indices) > 1 {
			// Skip the first occurrence (it's not a duplicate)
			for _, i := range indices[1:] {
				errs = append(errs, field.Invalid(
					// TODO(hidde): When the deprecated "stage" field is removed,
					// this can become more specific. I.e.
					// f.Index(i).Child("stageSelector").Child("name"),
					f.Index(i),
					stage,
					fmt.Sprintf("stage name already defined at %s", f.Index(indices[0])),
				))
			}
		}
	}

	return errs
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
