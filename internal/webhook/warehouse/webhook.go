package warehouse

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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

	validateCreateOrUpdateFn func(*kargoapi.Warehouse) error

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
) error {
	warehouse := obj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	if err := w.validateProjectFn(
		ctx,
		w.client,
		warehouseGroupKind,
		warehouse,
	); err != nil {
		return err
	}
	return w.validateCreateOrUpdateFn(warehouse)
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) error {
	warehouse := newObj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	return w.validateCreateOrUpdateFn(warehouse)
}

func (w *webhook) ValidateDelete(context.Context, runtime.Object) error {
	// No-op
	return nil
}

func (w *webhook) validateCreateOrUpdate(warehouse *kargoapi.Warehouse) error {
	if errs :=
		w.validateSpecFn(field.NewPath("spec"), warehouse.Spec); len(errs) > 0 {
		return apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil
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
	subs *kargoapi.RepoSubscriptions,
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
	subs []kargoapi.ImageSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, w.validateImageSub(f.Index(i), sub)...)
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
		if _, _, _, err := image.ParsePlatform(sub.Platform); err != nil {
			errs = append(errs, field.Invalid(f.Child("platform"), sub.Platform, ""))
		}
	}
	return errs
}

func (w *webhook) validateChartSubs(
	f *field.Path,
	subs []kargoapi.ChartSubscription,
) field.ErrorList {
	var errs field.ErrorList
	for i, sub := range subs {
		errs = append(errs, w.validateChartSub(f.Index(i), sub)...)
	}
	return errs
}

func (w *webhook) validateChartSub(
	f *field.Path,
	sub kargoapi.ChartSubscription,
) field.ErrorList {
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		return field.ErrorList{err}
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
