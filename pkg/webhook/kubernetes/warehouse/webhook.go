package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
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

const defaultDiscoveryLimit = int32(20)

func (w *webhook) Default(ctx context.Context, obj runtime.Object) error {
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
	obj runtime.Object,
) (admission.Warnings, error) {
	warehouse := obj.(*kargoapi.Warehouse) // nolint: forcetypeassert
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
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	warehouse := newObj.(*kargoapi.Warehouse) // nolint: forcetypeassert
	if errs := w.validateSpec(ctx, field.NewPath("spec"), &warehouse.Spec); len(errs) > 0 {
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
	for i, sub := range subs {
		errs = append(errs, w.validateSub(ctx, f.Index(i), sub, seen)...)
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

	// Validate Name: MinLength=1
	if err := validation.MinLength(f.Child("name"), sub.Name, 1); err != nil {
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
	case sub.Subscription != nil:
		k := subscriptionKey{
			kind: "sub",
			id:   strings.TrimSpace(strings.ToLower(sub.Subscription.Name)),
		}
		if _, exists := s[k]; exists {
			return field.Invalid(
				f.Child("subscription"),
				sub.Subscription.Name,
				fmt.Sprintf("subscription with name %q already exists at %q", sub.Subscription.Name, s[k]),
			)
		}
	}
	return nil
}
