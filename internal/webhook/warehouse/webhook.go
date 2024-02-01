package warehouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/image"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

var warehouseGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "Warehouse",
}

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	validateProjectFn func(
		context.Context,
		client.Client,
		schema.GroupKind,
		client.Object,
	) error

	validateCreateOrUpdateFn func(*kargoapi.Warehouse) (admission.Warnings, error)

	validateSpecFn func(*field.Path, *kargoapi.WarehouseSpec) field.ErrorList
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
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
	warehouse := obj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	if err := w.validateProjectFn(
		ctx,
		w.client,
		warehouseGroupKind,
		warehouse,
	); err != nil {
		return nil, err
	}
	return w.validateCreateOrUpdateFn(warehouse)
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	warehouse := newObj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	return w.validateCreateOrUpdateFn(warehouse)
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	// No-op
	return nil, nil
}

func (w *webhook) validateCreateOrUpdate(
	warehouse *kargoapi.Warehouse,
) (admission.Warnings, error) {
	if errs :=
		w.validateSpecFn(field.NewPath("spec"), warehouse.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec *kargoapi.WarehouseSpec,
) field.ErrorList {
	if spec == nil { // nil spec is caught by declarative validations
		return nil
	}
	return w.validateSubs(f.Child("subscriptions"), spec.Subscriptions)
}

func (w *webhook) validateSubs(
	f *field.Path,
	subs []kargoapi.RepoSubscription,
) field.ErrorList {
	if len(subs) == 0 {
		return nil
	}
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, w.validateSub(f.Index(i), sub)...)
	}
	return errs
}

func (w *webhook) validateSub(
	f *field.Path,
	sub kargoapi.RepoSubscription,
) field.ErrorList {
	var errs field.ErrorList
	var repoTypes int
	if sub.Git != nil {
		repoTypes++
	}
	if sub.Image != nil {
		repoTypes++
		errs = append(errs, w.validateImageSub(f.Child("image"), *sub.Image)...)
	}
	if sub.Chart != nil {
		repoTypes++
		errs = append(errs, w.validateChartSub(f.Child("chart"), *sub.Chart)...)
	}
	if repoTypes != 1 {
		errs = append(
			errs,
			field.Invalid(
				f,
				sub,
				fmt.Sprintf(
					"exactly one of %s.git, %s.images, or %s.charts must be non-empty",
					f.String(),
					f.String(),
					f.String(),
				),
			),
		)
	}
	return errs
}

func (w *webhook) validateImageSub(
	f *field.Path,
	sub kargoapi.ImageSubscription,
) field.ErrorList {
	var errs field.ErrorList
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = field.ErrorList{err}
	}
	if sub.Platform != "" {
		if !image.ValidatePlatformConstraint(sub.Platform) {
			errs = append(errs, field.Invalid(f.Child("platform"), sub.Platform, ""))
		}
	}
	return errs
}

func (w *webhook) validateChartSub(
	f *field.Path,
	sub kargoapi.ChartSubscription,
) field.ErrorList {
	errs := field.ErrorList{}
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = append(errs, err)
	}
	if strings.HasPrefix(sub.RepoURL, "oci://") && sub.Name != "" {
		errs = append(
			errs,
			field.Invalid(
				f.Child("name"),
				sub.Name,
				"must be empty if repoURL starts with oci://",
			),
		)
	}
	if (strings.HasPrefix(sub.RepoURL, "http://") || strings.HasPrefix(sub.RepoURL, "https://")) && sub.Name == "" {
		errs = append(
			errs,
			field.Invalid(
				f.Child("name"),
				sub.Name,
				"must be non-empty if repoURL starts with http:// or https://",
			),
		)
	}
	if len(errs) > 0 {
		return errs
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
