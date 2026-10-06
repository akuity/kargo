package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sValidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/subscription"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/akuity/kargo/pkg/validation"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

var warehouseGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "Warehouse",
}

type webhook struct {
	client             client.Client
	subscriberRegistry subscription.SubscriberRegistry
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(
		mgr.GetClient(),
		subscription.DefaultSubscriberRegistry,
	)
	return libWebhook.SetupValidatingAndDefaultingWebhook(mgr, &kargoapi.Warehouse{}, w)
}

func newWebhook(
	kubeClient client.Client,
	subscriberRegistry subscription.SubscriberRegistry,
) *webhook {
	return &webhook{
		client:             kubeClient,
		subscriberRegistry: subscriberRegistry,
	}
}

const defaultDiscoveryLimit = int32(20)

func (w *webhook) Default(ctx context.Context, warehouse *kargoapi.Warehouse) error {
	// Sync the shard label to the convenience shard field
	if warehouse.Spec.Shard != "" {
		if warehouse.Labels == nil {
			warehouse.Labels = make(map[string]string, 1)
		}
		warehouse.Labels[kargoapi.LabelKeyShard] = warehouse.Spec.Shard
	} else {
		delete(warehouse.Labels, kargoapi.LabelKeyShard)
	}

	for i := range warehouse.Spec.InternalSubscriptions {
		sub := &warehouse.Spec.InternalSubscriptions[i]
		subReg, err := w.subscriberRegistry.Get(ctx, *sub)
		if err != nil {
			return err
		}
		// The registration's value is a factory function
		subscriber, err := subReg.Value(ctx, nil)
		if err != nil {
			return fmt.Errorf("error instantiating subscriber: %w", err)
		}

		// Default common elements of generic subscriptions
		if sub.Subscription != nil {
			if sub.Subscription.DiscoveryLimit == 0 {
				sub.Subscription.DiscoveryLimit = defaultDiscoveryLimit
			}
		}

		if err := subscriber.ApplySubscriptionDefaults(ctx, sub); err != nil {
			return fmt.Errorf("error applying defaults to subscriptions: %w", err)
		}
	}

	return nil
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (admission.Warnings, error) {
	var errs field.ErrorList
	if err := libWebhook.ValidateProject(
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
	errs = append(errs, w.validateSpec(ctx, field.NewPath("spec"), &warehouse.Spec)...)
	if errs = append(
		errs,
		validateRemovedSubFields(ctx, field.NewPath("spec", "subscriptions"))...,
	); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldWarehouse *kargoapi.Warehouse,
	warehouse *kargoapi.Warehouse,
) (admission.Warnings, error) {
	errs := w.validateSpec(ctx, field.NewPath("spec"), &warehouse.Spec)
	if errs = append(
		errs,
		w.validateRemovedSubFieldsOnUpdate(
			ctx,
			field.NewPath("spec", "subscriptions"),
			oldWarehouse,
			warehouse,
		)...,
	); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	*kargoapi.Warehouse,
) (admission.Warnings, error) {
	// No-op
	return nil, nil
}

func (w *webhook) validateSpec(
	ctx context.Context,
	f *field.Path,
	spec *kargoapi.WarehouseSpec,
) field.ErrorList {
	if spec == nil { // nil spec is caught by declarative validations
		return nil
	}
	return w.validateSubs(ctx, f.Child("subscriptions"), spec.InternalSubscriptions)
}

// removedSubFields lists fields that have been removed from the original
// subscription types, keyed by subscription type. Decoding a Warehouse
// silently drops unknown fields, so these would otherwise be ignored without
// any indication that the tag filtering they once provided no longer applies.
var removedSubFields = []struct {
	subType string
	fields  []string
}{
	{subType: "git", fields: []string{"allowTags", "ignoreTags"}},
	{subType: "image", fields: []string{"allowTags", "ignoreTags"}},
}

// validateRemovedSubFields rejects subscriptions that still use any of the
// removedSubFields. Because the typed Warehouse has already lost those fields,
// this inspects the raw object from the admission request in ctx. If ctx
// carries no admission request, there is nothing to inspect and no errors are
// returned.
func validateRemovedSubFields(ctx context.Context, f *field.Path) field.ErrorList {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil
	}
	return removedSubFieldErrs(req.Object.Raw, f)
}

// validateRemovedSubFieldsOnUpdate is like validateRemovedSubFields, but
// permits an update that leaves the spec unchanged (e.g. a refresh annotation)
// to keep removed fields that oldWarehouse already used, so that a Warehouse
// that has yet to be migrated can still be refreshed. Any other update must
// also remove them.
func (w *webhook) validateRemovedSubFieldsOnUpdate(
	ctx context.Context,
	f *field.Path,
	oldWarehouse *kargoapi.Warehouse,
	warehouse *kargoapi.Warehouse,
) field.ErrorList {
	errs := validateRemovedSubFields(ctx, f)
	if len(errs) == 0 || !w.specUnchanged(ctx, oldWarehouse, warehouse) {
		return errs
	}
	// errs is only non-empty if ctx carries an admission request
	req, _ := admission.RequestFromContext(ctx)
	oldErrs := removedSubFieldErrs(req.OldObject.Raw, f)
	for _, err := range errs {
		// Typed specs don't include removed fields, so an update that adds one
		// still leaves the spec "unchanged." Such an update is rejected.
		if !slices.ContainsFunc(oldErrs, func(oldErr *field.Error) bool {
			return oldErr.Field == err.Field
		}) {
			return errs
		}
	}
	return nil
}

// specUnchanged reports whether an update leaves the Warehouse's spec as it
// was. oldWarehouse is defaulted first, just as the defaulting webhook has
// already defaulted warehouse, so that defaults introduced since oldWarehouse
// was last written don't count as changes.
func (w *webhook) specUnchanged(
	ctx context.Context,
	oldWarehouse *kargoapi.Warehouse,
	warehouse *kargoapi.Warehouse,
) bool {
	if oldWarehouse == nil {
		return false
	}
	old := oldWarehouse.DeepCopy()
	if err := w.Default(ctx, old); err != nil {
		return false
	}
	// Default() only updates InternalSubscriptions, so the raw Subscriptions,
	// which aren't defaulted, are left out of the comparison.
	oldSpec, newSpec := old.Spec, warehouse.Spec
	oldSpec.Subscriptions, newSpec.Subscriptions = nil, nil
	return equality.Semantic.DeepEqual(oldSpec, newSpec)
}

// removedSubFieldErrs returns an error for each of the removedSubFields used
// by a subscription of the raw, JSON-encoded Warehouse. Malformed input is
// caught when decoding the Warehouse, so it yields no errors here.
func removedSubFieldErrs(raw []byte, f *field.Path) field.ErrorList {
	var obj struct {
		Spec struct {
			Subscriptions []map[string]any `json:"subscriptions"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	var errs field.ErrorList
	for i, sub := range obj.Spec.Subscriptions {
		for _, removed := range removedSubFields {
			cfg, _ := sub[removed.subType].(map[string]any)
			for _, name := range removed.fields {
				if _, ok := cfg[name]; ok {
					errs = append(errs, field.Forbidden(
						f.Index(i).Child(removed.subType, name),
						fmt.Sprintf("%s has been removed; use %sRegexes instead", name, name),
					))
				}
			}
		}
	}
	return errs
}

func (w *webhook) validateSubs(
	ctx context.Context,
	f *field.Path,
	subs []kargoapi.RepoSubscription,
) field.ErrorList {
	if len(subs) == 0 {
		return nil
	}
	var errs field.ErrorList
	seen := make(uniqueSubSet, len(subs))
	seenNames := make(map[string]*field.Path, len(subs))
	for i, sub := range subs {
		errs = append(errs, w.validateSub(ctx, f.Index(i), sub, seen)...)
		namePath := f.Index(i).Child("name")
		// Generic subscriptions are identified by name alone, so, unlike the
		// original three subscription types, a name is required.
		if sub.Name == "" {
			if sub.Subscription != nil {
				errs = append(errs, field.Required(
					namePath,
					"a name is required for subscriptions of this type",
				))
			}
			continue
		}
		for _, msg := range k8sValidation.IsDNS1123Label(sub.Name) {
			errs = append(errs, field.Invalid(namePath, sub.Name, msg))
		}
		if prev, exists := seenNames[sub.Name]; exists {
			errs = append(errs, field.Invalid(
				namePath,
				sub.Name,
				fmt.Sprintf("subscription name %q already used at %q", sub.Name, prev),
			))
		} else {
			seenNames[sub.Name] = namePath
		}
	}
	return errs
}

func (w *webhook) validateSub(
	ctx context.Context,
	f *field.Path,
	sub kargoapi.RepoSubscription,
	seen uniqueSubSet,
) field.ErrorList {
	// A small bit of special-casing is required here because, unlike generic
	// subscriptions, the original three subscription types do not have a field
	// that indicates their type.
	switch {
	case sub.Chart != nil:
		f = f.Child("chart")
	case sub.Git != nil:
		f = f.Child("git")
	case sub.Image != nil:
		f = f.Child("image")
	case sub.Subscription != nil:
		f = f.Child(sub.Subscription.SubscriptionType)
	}

	subReg, err := w.subscriberRegistry.Get(ctx, sub)
	if err != nil {
		return field.ErrorList{field.Invalid(
			f,
			"",
			fmt.Sprintf("subscriber registry lookup failed: %v", err),
		)}
	}
	// The registration's value is a factory function
	subscriber, err := subReg.Value(ctx, nil)
	if err != nil {
		return field.ErrorList{field.Invalid(
			f,
			"",
			fmt.Sprintf("subscriber instantiation failed: %v", err),
		)}
	}

	var errs field.ErrorList

	// Validate the common elements of generic subscriptions
	if sub.Subscription != nil {
		errs = append(errs, w.validateGenericSub(f, *sub.Subscription)...)
	}

	// Subscriber-specific validation
	errs = append(errs, subscriber.ValidateSubscription(ctx, f, sub)...)

	// Validate uniqueness
	if err := seen.addSub(f, sub); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func (w *webhook) validateGenericSub(
	f *field.Path,
	sub kargoapi.Subscription,
) field.ErrorList {
	var errs field.ErrorList

	// Validate SubscriptionType: MinLength=1
	if err := validation.MinLength(
		f.Child("subscriptionType"),
		sub.SubscriptionType,
		1,
	); err != nil {
		errs = append(errs, err)
	}

	// Validate DiscoveryLimit: Minimum=1, Maximum=100
	if sub.DiscoveryLimit < 1 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be >= 1",
		))
	} else if sub.DiscoveryLimit > 100 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be <= 100",
		))
	}

	return errs
}

type subscriptionKey struct {
	kind string
	id   string
}

type uniqueSubSet map[subscriptionKey]*field.Path

// TODO(krancour): This method will require substantial refactoring when we
// eventually move toward permitting Warehouses to have multiple subscriptions
// to the same repository, as long as they are qualified with different names.
// See https://github.com/akuity/kargo/issues/6724.
func (s uniqueSubSet) addSub(
	f *field.Path,
	sub kargoapi.RepoSubscription,
) *field.Error {
	// A small bit of special-casing is required here because, unlike generic
	// subscriptions, the original three subscription types do not have one common
	// way to identify them uniquely.
	switch {
	case sub.Chart != nil:
		k := subscriptionKey{
			kind: "chart",
			id:   urls.NormalizeChart(sub.Chart.RepoURL),
		}
		isHTTP := strings.HasPrefix(sub.Chart.RepoURL, "http://") || strings.HasPrefix(sub.Chart.RepoURL, "https://")
		if isHTTP {
			// For classical HTTP(S) Helm chart repositories, the chart name is part
			// of the uniqueness criteria
			k.id = k.id + ":" + sub.Chart.Name
		}
		if _, exists := s[k]; exists {
			var errMsg string
			if isHTTP {
				errMsg = fmt.Sprintf(
					"subscription for chart %q already exists at %q",
					sub.Chart.Name, s[k],
				)
			} else {
				errMsg = fmt.Sprintf("subscription for chart already exists at %q", s[k])
			}
			return field.Invalid(f.Child("chart"), sub.Chart.RepoURL, errMsg)
		}
		s[k] = f
	case sub.Git != nil:
		k := subscriptionKey{
			kind: "git",
			id:   urls.NormalizeGit(sub.Git.RepoURL),
		}
		if _, exists := s[k]; exists {
			return field.Invalid(
				f.Child("git"),
				sub.Git.RepoURL,
				fmt.Sprintf("subscription for Git repository already exists at %q", s[k]),
			)
		}
		s[k] = f
	case sub.Image != nil:
		k := subscriptionKey{
			kind: "image",
			id:   urls.NormalizeImage(sub.Image.RepoURL),
		}
		if _, exists := s[k]; exists {
			return field.Invalid(
				f.Child("image"),
				sub.Image.RepoURL,
				fmt.Sprintf("subscription for image repository already exists at %q", s[k]),
			)
		}
		s[k] = f
	}
	// Generic subscriptions have no repository URL to be deduplicated by. They
	// are distinguished from one another by name alone, which validateSubs
	// requires and verifies to be unique.
	return nil
}
