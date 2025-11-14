package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/image"
	"github.com/akuity/kargo/pkg/urls"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
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
		client.Object,
	) error

	validateSpecFn func(*field.Path, *kargoapi.WarehouseSpec) field.ErrorList
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	w := &webhook{
		client: kubeClient,
	}
	w.validateProjectFn = libWebhook.ValidateProject
	w.validateSpecFn = w.validateSpec
	return w
}

func (w *webhook) Default(_ context.Context, obj runtime.Object) error {
	warehouse := obj.(*kargoapi.Warehouse) // nolint: forcetypeassert

	// Sync the shard label to the convenience shard field
	if warehouse.Spec.Shard != "" {
		if warehouse.Labels == nil {
			warehouse.Labels = make(map[string]string, 1)
		}
		warehouse.Labels[kargoapi.LabelKeyShard] = warehouse.Spec.Shard
	} else {
		delete(warehouse.Labels, kargoapi.LabelKeyShard)
	}

	return nil
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	warehouse := obj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	var errs field.ErrorList
	if err := w.validateProjectFn(
		ctx,
		w.client,
		warehouse,
	); err != nil {
		var statusErr *apierrors.StatusError
		if ok := errors.As(err, &statusErr); ok {
			return nil, statusErr
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); !ok {
			return nil, apierrors.NewInternalError(err)
		}
		errs = append(errs, fieldErr)
	}
	if errs = append(
		errs,
		w.validateSpecFn(field.NewPath("spec"), &warehouse.Spec)...,
	); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	warehouse := newObj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	if errs := w.validateSpecFn(field.NewPath("spec"), &warehouse.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	// No-op
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
	seen := make(uniqueSubSet, len(subs))
	for i, sub := range subs {
		errs = append(errs, w.validateSub(f.Index(i), sub, seen)...)
	}
	return errs
}

func (w *webhook) validateSub(
	f *field.Path,
	sub kargoapi.RepoSubscription,
	seen uniqueSubSet,
) field.ErrorList {
	var errs field.ErrorList
	var repoTypes int
	if sub.Git != nil {
		repoTypes++
		errs = append(errs, w.validateGitSub(f.Child("git"), *sub.Git, seen)...)
	}
	if sub.Image != nil {
		repoTypes++
		errs = append(errs, w.validateImageSub(f.Child("image"), *sub.Image, seen)...)
	}
	if sub.Chart != nil {
		repoTypes++
		errs = append(errs, w.validateChartSub(f.Child("chart"), *sub.Chart, seen)...)
	}
	if repoTypes != 1 {
		errs = append(
			errs,
			field.Invalid(
				f,
				sub,
				fmt.Sprintf(
					"exactly one of %s.git, %s.image, or %s.chart must be non-empty",
					f.String(),
					f.String(),
					f.String(),
				),
			),
		)
	}
	return errs
}

func (w *webhook) validateGitSub(
	f *field.Path,
	sub kargoapi.GitSubscription,
	seen uniqueSubSet,
) field.ErrorList {
	var errs field.ErrorList
	if err := validateSemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = append(errs, err)
	}
	if err := seen.addGit(sub, f); err != nil {
		errs = append(errs, field.Invalid(f, sub.RepoURL, err.Error()))
	}
	return errs
}

func (w *webhook) validateImageSub(
	f *field.Path,
	sub kargoapi.ImageSubscription,
	seen uniqueSubSet,
) field.ErrorList {
	var errs field.ErrorList
	if sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategySemVer || sub.ImageSelectionStrategy == "" {
		if sub.Constraint != "" {
			if err := validateSemverConstraint(
				f.Child("constraint"),
				sub.Constraint,
			); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if sub.Platform != "" {
		if !image.ValidatePlatformConstraint(sub.Platform) {
			errs = append(errs, field.Invalid(f.Child("platform"), sub.Platform, ""))
		}
	}
	if err := seen.addImage(sub, f); err != nil {
		errs = append(errs, field.Invalid(f, sub.RepoURL, err.Error()))
	}
	return errs
}

func (w *webhook) validateChartSub(
	f *field.Path,
	sub kargoapi.ChartSubscription,
	seen uniqueSubSet,
) field.ErrorList {
	var errs field.ErrorList
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
	isHTTP := strings.HasPrefix(sub.RepoURL, "http://") || strings.HasPrefix(sub.RepoURL, "https://")
	if isHTTP && sub.Name == "" {
		errs = append(
			errs,
			field.Invalid(
				f.Child("name"),
				sub.Name,
				"must be non-empty if repoURL starts with http:// or https://",
			),
		)
	}
	if err := seen.addChart(sub, isHTTP, f); err != nil {
		errs = append(errs, field.Invalid(f, sub.RepoURL, err.Error()))
	}
	return errs
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

type subscriptionKey struct {
	kind string
	id   string
}

type uniqueSubSet map[subscriptionKey]*field.Path

func (s uniqueSubSet) addGit(sub kargoapi.GitSubscription, p *field.Path) error {
	k := subscriptionKey{kind: "git", id: urls.NormalizeGit(sub.RepoURL)}
	if _, exists := s[k]; exists {
		return fmt.Errorf("subscription for Git repository already exists at %q", s[k])
	}
	s[k] = p
	return nil
}

func (s uniqueSubSet) addImage(sub kargoapi.ImageSubscription, p *field.Path) error {
	// The normalization of Helm chart repository URLs can also be used here
	// to ensure the uniqueness of the image reference as it does the job of
	// ensuring lower-casing, etc. without introducing unwanted side effects.
	k := subscriptionKey{kind: "image", id: urls.NormalizeChart(sub.RepoURL)}
	if _, exists := s[k]; exists {
		return fmt.Errorf("subscription for image repository already exists at %q", s[k])
	}
	s[k] = p
	return nil
}

func (s uniqueSubSet) addChart(sub kargoapi.ChartSubscription, isHTTP bool, p *field.Path) error {
	k := subscriptionKey{kind: "chart", id: urls.NormalizeChart(sub.RepoURL)}
	if isHTTP {
		k.id = k.id + ":" + sub.Name
	}
	if _, exists := s[k]; exists {
		if isHTTP {
			return fmt.Errorf("subscription for chart %q already exists at %q", sub.Name, s[k])
		}
		return fmt.Errorf("subscription for chart already exists at %q", s[k])
	}
	s[k] = p
	return nil
}
