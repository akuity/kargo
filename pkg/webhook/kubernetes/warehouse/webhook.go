package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	return ctrl.NewWebhookManagedBy(mgr, &kargoapi.Warehouse{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
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

const defaultDiscoveryLimit = int64(20)

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

		// Apply DiscoveryLimit defaults.
		w.applyDiscoveryLimitDefaults(sub)

		if err := subscriber.ApplySubscriptionDefaults(ctx, sub); err != nil {
			return fmt.Errorf("error applying defaults to subscriptions: %w", err)
		}
	}

	return nil
}

func (w *webhook) applyDiscoveryLimitDefaults(sub *kargoapi.RepoSubscription) {
	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	subLimit := 0
	switch {
	case sub.Chart != nil:
		subLimit = int(sub.Chart.DiscoveryLimit)
	case sub.Git != nil:
		subLimit = int(sub.Git.DiscoveryLimit)
	case sub.Image != nil:
		subLimit = int(sub.Image.DiscoveryLimit)
	case sub.Subscription != nil:
		subLimit = int(sub.Subscription.DiscoveryLimit)
	}

	// Subscription lower-level limit is not set, we can set a top-level default
	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	if subLimit == 0 {
		if sub.DiscoveryLimit == 0 {
			sub.DiscoveryLimit = defaultDiscoveryLimit
		}
	}
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
	if errs = append(
		errs,
		w.validateSpec(ctx, field.NewPath("spec"), &warehouse.Spec)...,
	); len(errs) > 0 {
		return nil, apierrors.NewInvalid(warehouseGroupKind, warehouse.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	_ *kargoapi.Warehouse,
	warehouse *kargoapi.Warehouse,
) (admission.Warnings, error) {
	if errs := w.validateSpec(ctx, field.NewPath("spec"), &warehouse.Spec); len(errs) > 0 {
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
	basePath *field.Path,
	sub kargoapi.RepoSubscription,
	seen uniqueSubSet,
) field.ErrorList {

	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	innerDiscoveryLimit := 0
	var subPath *field.Path
	// A small bit of special-casing is required here because, unlike generic
	// subscriptions, the original three subscription types do not have a field
	// that indicates their type.
	switch {
	case sub.Chart != nil:
		subPath = basePath.Child("chart")
		innerDiscoveryLimit = int(sub.Chart.DiscoveryLimit)
	case sub.Git != nil:
		subPath = basePath.Child("git")
		innerDiscoveryLimit = int(sub.Git.DiscoveryLimit)
	case sub.Image != nil:
		subPath = basePath.Child("image")
		innerDiscoveryLimit = int(sub.Image.DiscoveryLimit)
	case sub.Subscription != nil:
		subPath = basePath.Child(sub.Subscription.SubscriptionType)
		innerDiscoveryLimit = int(sub.Subscription.DiscoveryLimit)
	}

	subReg, err := w.subscriberRegistry.Get(ctx, sub)
	if err != nil {
		return field.ErrorList{field.Invalid(
			subPath,
			"",
			fmt.Sprintf("subscriber registry lookup failed: %v", err),
		)}
	}
	// The registration's value is a factory function
	subscriber, err := subReg.Value(ctx, nil)
	if err != nil {
		return field.ErrorList{field.Invalid(
			subPath,
			"",
			fmt.Sprintf("subscriber instantiation failed: %v", err),
		)}
	}

	var errs field.ErrorList

	// Validate base level subscription
	errs = append(errs, w.validateBaseSub(basePath, subPath, sub, innerDiscoveryLimit)...)

	// Validate the common elements of generic subscriptions
	if sub.Subscription != nil {
		errs = append(errs, w.validateGenericSub(subPath, *sub.Subscription)...)
	}

	// Subscriber-specific validation
	errs = append(errs, subscriber.ValidateSubscription(ctx, subPath, sub)...)

	// Validate uniqueness
	if err := seen.addSub(basePath, sub); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func (w *webhook) validateBaseSub(
	basePath *field.Path,
	subPath *field.Path,
	sub kargoapi.RepoSubscription,
	innerDiscoveryLimit int,
) field.ErrorList {
	var errs field.ErrorList

	// Prohibit both limits being set at the same time
	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	if sub.DiscoveryLimit != 0 && innerDiscoveryLimit != 0 {
		errs = append(errs, field.Invalid(
			subPath.Child("discoveryLimit"),
			innerDiscoveryLimit,
			"cannot be set at the same time as discoveryLimit at the subscription level",
		))
	}

	// Validate top level DiscoveryLimit if inner limit is not set
	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	if sub.DiscoveryLimit < 1 && innerDiscoveryLimit < 1 {
		errs = append(errs, field.Invalid(
			basePath.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be >= 1",
		))
	} else if sub.DiscoveryLimit > 100 {
		errs = append(errs, field.Invalid(
			basePath.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be <= 100",
		))
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

	// Lower-bound validation is moved to base subscription validation
	// TODO: clean this up when removing DiscoveryLimits from specific subscriptions
	if sub.DiscoveryLimit > 100 {
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
