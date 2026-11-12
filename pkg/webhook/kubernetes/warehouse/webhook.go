package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	jsonpatchapply "github.com/evanphx/json-patch/v5"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
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
	// An update that changes only metadata (e.g. a refresh annotation) is
	// permitted even if the spec is invalid. Any other update must leave the
	// spec valid.
	if !w.specChanged(ctx) {
		return nil, nil
	}
	if errs := w.validateSpec(
		ctx,
		field.NewPath("spec"),
		&warehouse.Spec,
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

// specChanged reports whether the update in the admission request in ctx
// changes the Warehouse's spec. The raw objects are compared, because decoding
// a Warehouse silently drops subscription fields that aren't defined by the
// subscription's type. The old object is first given the defaults that the
// defaulting webhook has already given the new one, so that defaults
// introduced since the old object was last written don't count as changes. If
// ctx carries no admission request, or the old object can't be defaulted, the
// spec is assumed to have changed.
func (w *webhook) specChanged(ctx context.Context) bool {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return true
	}
	oldRaw, err := w.defaultRaw(ctx, req.OldObject.Raw)
	if err != nil {
		return true
	}
	var oldObj, newObj struct {
		Spec any `json:"spec"`
	}
	if json.Unmarshal(oldRaw, &oldObj) != nil ||
		json.Unmarshal(req.Object.Raw, &newObj) != nil {
		return true
	}
	return !reflect.DeepEqual(oldObj.Spec, newObj.Spec)
}

// defaultRaw applies Default() to the raw, JSON-encoded Warehouse the same way
// the defaulting webhook does: by patching the raw Warehouse with the
// difference that Default() makes to the decoded one. Fields that decoding
// drops are left intact.
func (w *webhook) defaultRaw(ctx context.Context, raw []byte) ([]byte, error) {
	warehouse := &kargoapi.Warehouse{}
	if err := json.Unmarshal(raw, warehouse); err != nil {
		return nil, fmt.Errorf("error decoding Warehouse: %w", err)
	}
	before, err := json.Marshal(warehouse)
	if err != nil {
		return nil, fmt.Errorf("error encoding Warehouse: %w", err)
	}
	if err = w.Default(ctx, warehouse); err != nil {
		return nil, fmt.Errorf("error defaulting Warehouse: %w", err)
	}
	after, err := json.Marshal(warehouse)
	if err != nil {
		return nil, fmt.Errorf("error encoding defaulted Warehouse: %w", err)
	}
	ops, err := jsonpatch.CreatePatch(before, after)
	if err != nil {
		return nil, fmt.Errorf("error computing defaults patch: %w", err)
	}
	if len(ops) == 0 {
		return raw, nil
	}
	opsJSON, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("error encoding defaults patch: %w", err)
	}
	patch, err := jsonpatchapply.DecodePatch(opsJSON)
	if err != nil {
		return nil, fmt.Errorf("error decoding defaults patch: %w", err)
	}
	if raw, err = patch.Apply(raw); err != nil {
		return nil, fmt.Errorf("error applying defaults patch: %w", err)
	}
	return raw, nil
}

func (w *webhook) validateSpec(
	ctx context.Context,
	f *field.Path,
	spec *kargoapi.WarehouseSpec,
) field.ErrorList {
	if spec == nil { // nil spec is caught by declarative validations
		return nil
	}
	return w.validateSubs(
		ctx,
		f.Child("subscriptions"),
		spec.InternalSubscriptions,
		rawSubscriptions(ctx),
	)
}

// rawSubscriptions returns the raw, JSON-encoded subscriptions of the
// Warehouse in the admission request in ctx, keyed by their top-level fields.
// Decoding a Warehouse silently drops subscription fields that aren't defined
// by the subscription's type, so only the raw subscriptions can be fully
// validated. If ctx carries no admission request, or its object is malformed
// (which decoding the Warehouse would already have caught), nil is returned.
func rawSubscriptions(ctx context.Context) []map[string]json.RawMessage {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil
	}
	var obj struct {
		Spec struct {
			Subscriptions []map[string]json.RawMessage `json:"subscriptions"`
		} `json:"spec"`
	}
	if err = json.Unmarshal(req.Object.Raw, &obj); err != nil {
		return nil
	}
	return obj.Spec.Subscriptions
}

// validateSubs validates subs, the typed subscriptions of a Warehouse.
// rawSubs, if non-nil, are the same subscriptions as raw, JSON-encoded objects
// keyed by their top-level fields.
func (w *webhook) validateSubs(
	ctx context.Context,
	f *field.Path,
	subs []kargoapi.RepoSubscription,
	rawSubs []map[string]json.RawMessage,
) field.ErrorList {
	if len(subs) == 0 {
		return nil
	}
	var errs field.ErrorList
	seen := make(uniqueSubSet, len(subs))
	seenNames := make(map[string]*field.Path, len(subs))
	for i, sub := range subs {
		var rawSub map[string]json.RawMessage
		if i < len(rawSubs) {
			rawSub = rawSubs[i]
		}
		errs = append(errs, w.validateSub(ctx, f.Index(i), sub, rawSub, seen)...)
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

// validateSub validates sub, a typed subscription. rawSub, if non-nil, is the
// same subscription as a raw, JSON-encoded object keyed by its top-level
// fields.
func (w *webhook) validateSub(
	ctx context.Context,
	f *field.Path,
	sub kargoapi.RepoSubscription,
	rawSub map[string]json.RawMessage,
	seen uniqueSubSet,
) field.ErrorList {
	// A small bit of special-casing is required here because, unlike generic
	// subscriptions, the original three subscription types do not have a field
	// that indicates their type.
	var subType string
	switch {
	case sub.Chart != nil:
		subType = "chart"
	case sub.Git != nil:
		subType = "git"
	case sub.Image != nil:
		subType = "image"
	case sub.Subscription != nil:
		subType = sub.Subscription.SubscriptionType
	}
	f = f.Child(subType)

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

	// Validate the raw subscription against the subscriber's schema, if it has
	// one. This catches fields that decoding the subscription dropped.
	if subSchema := subReg.Metadata.Schema; subSchema != nil && rawSub[subType] != nil {
		errs = append(errs, subSchema.Validate(f, rawSub[subType])...)
	}

	// Validate the common elements of generic subscriptions
	if sub.Subscription != nil {
		errs = append(errs, w.validateGenericSub(f, *sub.Subscription)...)
	}

	// Subscriber-specific validation of what the schema can't express
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
