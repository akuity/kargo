package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/subscription"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

// testRegistry is a subscriber registry for use in tests.
var testRegistry = subscription.MustNewSubscriberRegistry()

func init() {
	// Populate the test registry with a mock subscriber registration whose
	// predicate matches all subscriptions and whose mock subscriber always
	// returns one predictable validation error.
	testRegistry.MustRegister(subscription.SubscriberRegistration{
		Predicate: func(context.Context, kargoapi.RepoSubscription) (bool, error) {
			// Match all subscriptions for testing purposes
			return true, nil
		},
		Value: func(context.Context, credentials.Database) (subscription.Subscriber, error) {
			const testDiscoveryLimit = 42
			return &subscription.MockSubscriber{
				ApplySubscriptionDefaultsFn: func(_ context.Context, sub *kargoapi.RepoSubscription) error {
					// Make a predictable change to all types of subscriptions
					switch {
					case sub.Chart != nil:
						sub.Chart.DiscoveryLimit = testDiscoveryLimit
					case sub.Git != nil:
						sub.Git.DiscoveryLimit = testDiscoveryLimit
					case sub.Image != nil:
						sub.Image.DiscoveryLimit = testDiscoveryLimit
					case sub.Subscription != nil:
						// Although discovery limit is integral to generic subscriptions, we
						// don't want to modify it here and expect to see that applied in
						// our tests because it will interfere with testing that common
						// elements of generic subscriptions are defaulted properly. So,
						// even though this is a nonsensical thing to do, we'll make a
						// predictable change to the subscription's name instead, because it
						// will give us a way to verify that subscriber-specific defaulting
						// logic works for generic subscriptions.
						sub.Name = "fake"
					}
					return nil
				},
				ValidateSubscriptionFn: func(
					_ context.Context,
					f *field.Path,
					sub kargoapi.RepoSubscription,
				) field.ErrorList {
					// Always return a predictable validation error
					return field.ErrorList{{
						Type:     field.ErrorTypeInvalid,
						Field:    f.String(),
						BadValue: sub,
						Detail:   "mock validation error",
					}}
				},
			}, nil
		},
	})
}

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(kubeClient, subscription.DefaultSubscriberRegistry)
	require.Same(t, kubeClient, w.client)
	require.Same(t, subscription.DefaultSubscriberRegistry, w.subscriberRegistry)
}

func Test_webhook_Default(t *testing.T) {
	const testShardName = "fake-shard"

	w := &webhook{subscriberRegistry: testRegistry}

	t.Run("shard stays default when not specified at all", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{}
		err := w.Default(t.Context(), warehouse)
		require.NoError(t, err)
		require.Empty(t, warehouse.Labels)
		require.Empty(t, warehouse.Spec.Shard)
	})

	t.Run("sync shard label to non-empty shard field", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			Spec: kargoapi.WarehouseSpec{
				Shard: testShardName,
			},
		}
		err := w.Default(t.Context(), warehouse)
		require.NoError(t, err)
		require.Equal(t, testShardName, warehouse.Spec.Shard)
		require.Equal(t, testShardName, warehouse.Labels[kargoapi.LabelKeyShard])
	})

	t.Run("sync shard label to empty shard field", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					kargoapi.LabelKeyShard: testShardName,
				},
			},
		}
		err := w.Default(t.Context(), warehouse)
		require.NoError(t, err)
		require.Empty(t, warehouse.Spec.Shard)
		_, ok := warehouse.Labels[kargoapi.LabelKeyShard]
		require.False(t, ok)
	})

	t.Run("discovery limit default is set of RepoSubscription", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			Spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					// The one mock subscriber in the test registry will apply
					// predicated changes to each of these subscriptions.
					{Git: &kargoapi.GitSubscription{}},
					{Image: &kargoapi.ImageSubscription{}},
					{Chart: &kargoapi.ChartSubscription{}},
					{Subscription: &kargoapi.Subscription{SubscriptionType: "fake"}},
				},
			},
		}
		err := w.Default(t.Context(), warehouse)
		require.NoError(t, err)
		const testDiscoveryLimit int64 = 20
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[0].DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[1].DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[2].DiscoveryLimit)
	})

	t.Run("defaulting in subscribers", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			Spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					// The one mock subscriber in the test registry will apply
					// predicated changes to each of these subscriptions.
					{Git: &kargoapi.GitSubscription{}},
					{Image: &kargoapi.ImageSubscription{}},
					{Chart: &kargoapi.ChartSubscription{}},
					{Subscription: &kargoapi.Subscription{SubscriptionType: "fake"}},
				},
			},
		}
		err := w.Default(t.Context(), warehouse)
		require.NoError(t, err)
		const testDiscoveryLimit int64 = 42
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[0].Git.DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[1].Image.DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[2].Chart.DiscoveryLimit)
		require.Equal(t, "fake", warehouse.Spec.InternalSubscriptions[3].Name)
	})
}

func Test_webhook_ValidateCreate(t *testing.T) {
	const testProject = "fake-project"

	testScheme := runtime.NewScheme()
	err := corev1.AddToScheme(testScheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		webhook    *webhook
		req        *admission.Request
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating project",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonNotFound,
					statusErr.ErrStatus.Reason,
				)
			},
		},
		{
			name: "error validating warehouse",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{RepoURL: "bogus"},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(
					t,
					statusErr.ErrStatus.Message,
					"spec.subscriptions[0].git.repoURL",
				)
			},
		},
		{
			name: "success",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "subscription names must be unique",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{
						{
							Name: "alpha",
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-url-1",
							},
						},
						{
							Name: "alpha",
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-url-2",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[1].name")
				require.NotContains(t, err.Error(), ".image.name")
			},
		},
		{
			name: "subscription name is not a valid DNS label",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Name:  "Not A Valid Name",
						Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].name")
				require.Contains(t, err.Error(), "must consist of lower case alphanumeric characters")
			},
		},
		{
			name: "subscription name is too long",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Name:  strings.Repeat("a", 64),
						Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].name")
				require.Contains(t, err.Error(), "must be no more than 63 characters")
			},
		},
		{
			name: "generic subscription without a name",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Subscription: &kargoapi.Subscription{
							SubscriptionType: "fake-type",
							DiscoveryLimit:   20,
						},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].name")
				require.Contains(t, err.Error(), "Required value")
			},
		},
		{
			name: "removed subscription fields",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			req: &admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","allowTags":"^v1"}}]}}`),
					},
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Image: &kargoapi.ImageSubscription{RepoURL: "fake-url"},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].image.allowTags")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.webhook.subscriberRegistry = subscription.DefaultSubscriberRegistry
			ctx := t.Context()
			if testCase.req != nil {
				ctx = admission.NewContextWithRequest(ctx, *testCase.req)
			}
			_, err := testCase.webhook.ValidateCreate(
				ctx,
				testCase.warehouse,
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateUpdate(t *testing.T) {
	const testProject = "fake-project"

	testScheme := runtime.NewScheme()
	err := corev1.AddToScheme(testScheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		webhook   *webhook
		warehouse *kargoapi.Warehouse
		// oldRaw and newRaw, when set, are the raw old and new objects of the
		// admission request. newRaw then also takes the place of warehouse.
		oldRaw     string
		newRaw     string
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating warehouse",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{RepoURL: "bogus"},
					}},
				},
			},
			assertions: func(t *testing.T, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(
					t,
					statusErr.ErrStatus.Message,
					"spec.subscriptions[0].git.repoURL",
				)
			},
		},
		{
			name: "success",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "subscription names must be unique",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
							},
						},
					},
				).Build(),
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject},
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{
						{
							Name: "alpha",
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-url-1",
							},
						},
						{
							Name: "alpha",
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-url-2",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[1].name")
				require.NotContains(t, err.Error(), ".image.name")
			},
		},
		{
			name:    "removed subscription fields",
			webhook: &webhook{},
			newRaw:  `{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","allowTags":"^v1"}}]}}`,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].image.allowTags")
			},
		},
		{
			name:    "removed subscription fields kept by update that changes the spec",
			webhook: &webhook{},
			oldRaw:  `{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","allowTags":"^v1"}}]}}`,
			newRaw: `{"spec":{"interval":"10m","subscriptions":[
				{"image":{"repoURL":"fake-url","allowTags":"^v1"}}
			]}}`,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].image.allowTags")
			},
		},
		{
			// The old object predates defaults that the new one has been given
			name:    "removed subscription fields kept by update that leaves the spec unchanged",
			webhook: &webhook{},
			oldRaw: `{"spec":{"subscriptions":[
				{"image":{"repoURL":"fake-url","ignoreTags":["v1.0.0"]}}
			]}}`,
			newRaw: `{"metadata":{"annotations":{"foo":"bar"}},"spec":{"subscriptions":[{"image":{
				"repoURL":"fake-url","ignoreTags":["v1.0.0"],"discoveryLimit":20,
				"imageSelectionStrategy":"SemVer","strictSemvers":true
			}}]}}`,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Changes to the values of removed fields are inconsequential
			name:    "removed subscription field changed by update that leaves the spec unchanged",
			webhook: &webhook{},
			oldRaw:  `{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","ignoreTags":["v1.0.0"]}}]}}`,
			newRaw:  `{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","ignoreTags":["v2.0.0"]}}]}}`,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "removed subscription field added by update that leaves the spec unchanged",
			webhook: &webhook{},
			oldRaw:  `{"spec":{"subscriptions":[{"image":{"repoURL":"fake-url","ignoreTags":["v1.0.0"]}}]}}`,
			newRaw: `{"spec":{"subscriptions":[
				{"image":{"repoURL":"fake-url","allowTags":"^v1","ignoreTags":["v1.0.0"]}}
			]}}`,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "spec.subscriptions[0].image.allowTags")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.webhook.subscriberRegistry = subscription.DefaultSubscriberRegistry
			ctx := t.Context()
			warehouse := testCase.warehouse
			var oldWarehouse *kargoapi.Warehouse
			if testCase.newRaw != "" {
				// By the time it's validated, the new object has been defaulted
				warehouse = decodeTestWarehouse(t, testCase.newRaw)
				require.NoError(t, testCase.webhook.Default(ctx, warehouse))
				req := admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Operation: admissionv1.Update,
						Object:    runtime.RawExtension{Raw: []byte(testCase.newRaw)},
					},
				}
				if testCase.oldRaw != "" {
					oldWarehouse = decodeTestWarehouse(t, testCase.oldRaw)
					req.OldObject = runtime.RawExtension{Raw: []byte(testCase.oldRaw)}
				}
				ctx = admission.NewContextWithRequest(ctx, req)
			}
			_, err := testCase.webhook.ValidateUpdate(
				ctx,
				oldWarehouse,
				warehouse,
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateDelete(t *testing.T) {
	w := &webhook{}
	_, err := w.ValidateDelete(t.Context(), nil)
	require.NoError(t, err, nil)
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       kargoapi.WarehouseSpec
		assertions func(*testing.T, *kargoapi.WarehouseSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "validation is delegated to subscribers",
			spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					// The one mock subscriber in the test registry will return
					// predictable errors for all of these subscriptions.
					{Git: &kargoapi.GitSubscription{}},
					{Image: &kargoapi.ImageSubscription{}},
					{Chart: &kargoapi.ChartSubscription{}},
					{
						Name: "fake-sub",
						Subscription: &kargoapi.Subscription{
							SubscriptionType: "fake",
							DiscoveryLimit:   20,
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.True(t, len(errs) >= 4)
				var fields = make([]string, len(errs))
				for i, err := range errs {
					fields[i] = err.Field
				}
				// Note that we're not at all interested in testing specific validation
				// logic here; that is the responsibility of an individual subscriber's
				// unit tests. ALL we want to verify here is that, for all three
				// original subscription types and generic subscription, validation is
				// delegated to corresponding subscribers and checking for these
				// predictable errors from the one mock subscriber in the test registry
				// accomplishes that.
				require.Contains(t, fields, "spec.subscriptions[0].git")
				require.Contains(t, fields, "spec.subscriptions[1].image")
				require.Contains(t, fields, "spec.subscriptions[2].chart")
				require.Contains(t, fields, "spec.subscriptions[3].fake")
			},
		},
		{
			name: "common elements of generic subscriptions are validated",
			spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					{
						// Name is empty and discovery limit is zero
						Subscription: &kargoapi.Subscription{SubscriptionType: "fake"},
					},
					{
						Subscription: &kargoapi.Subscription{
							SubscriptionType: "fake",
							DiscoveryLimit:   1000, // Too high
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.WarehouseSpec, errs field.ErrorList) {
				require.True(t, len(errs) >= 3)
				var fields = make([]string, len(errs))
				for i, err := range errs {
					fields[i] = err.Field
				}
				require.Contains(t, fields, "spec.subscriptions[0].name")
				require.Contains(t, fields, "spec.subscriptions[0].discoveryLimit")
				require.Contains(t, fields, "spec.subscriptions[1].fake.discoveryLimit")
			},
		},
	}
	w := &webhook{subscriberRegistry: testRegistry}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				&testCase.spec,
				w.validateSpec(
					t.Context(),
					field.NewPath("spec"),
					&testCase.spec,
				),
			)
		})
	}
}

// Test_webhook_Handle_PreservesUnrelatedDurationFormatting is a regression
// test for https://github.com/akuityio/akuity-platform/issues/12384, mirroring
// the Warehouse repro from that issue: an image subscription with no
// explicit discoveryLimit prompts a real Default() mutation, and
// spec.interval must survive untouched even though it never gets read or
// written by Default().
func Test_webhook_Handle_PreservesUnrelatedDurationFormatting(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	w := newWebhook(kubeClient, subscription.DefaultSubscriberRegistry)
	wh, err := libWebhook.NewDefaultingWebhook(scheme, &kargoapi.Warehouse{}, w)
	require.NoError(t, err)

	rawWarehouse := []byte(`{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind": "Warehouse",
		"metadata": {"name": "drift-wh", "namespace": "dur-drift-repro"},
		"spec": {
			"interval": "10m",
			"subscriptions": [{"image": {"repoURL": "public.ecr.aws/docker/library/nginx"}}]
		}
	}`)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object:    runtime.RawExtension{Raw: rawWarehouse},
		},
	}

	resp := wh.Handle(admission.NewContextWithRequest(context.Background(), req), req)
	require.True(t, resp.Allowed)

	testCases := []struct {
		name    string
		path    string
		present bool
	}{
		{
			// Sanity check: this mutation must still appear, or the test
			// below would pass by suppressing every patch, not just the
			// spurious ones.
			name:    "image discoveryLimit is defaulted",
			path:    "/spec/subscriptions/0/image/discoveryLimit",
			present: true,
		},
		{
			name:    "untouched interval is not rewritten",
			path:    "/spec/interval",
			present: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := slices.ContainsFunc(resp.Patches, func(p jsonpatch.JsonPatchOperation) bool {
				return p.Path == testCase.path
			})
			require.Equalf(t, testCase.present, got, "patches: %+v", resp.Patches)
		})
	}
}

func Test_validateRemovedSubFields(t *testing.T) {
	const removedSpec = `{"spec":{"subscriptions":[
		{"git":{"repoURL":"fake-git-url","allowTags":"^v1","ignoreTags":["v1.0.0"]}},
		{"chart":{"repoURL":"fake-chart-url"}},
		{"name":"img","image":{"repoURL":"fake-image-url","ignoreTags":["v1.0.0"]}}
	]}}`
	testCases := []struct {
		name       string
		req        *admission.Request
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "no admission request in context",
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "malformed object",
			req: &admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: []byte(`{`)},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "no removed fields",
			req: &admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{"spec":{"subscriptions":[
							{"image":{"repoURL":"fake-url","allowTagsRegexes":["^v1"]}}
						]}}`),
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "removed fields on create",
			req: &admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: []byte(removedSpec)},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 3)
				paths := make([]string, len(errs))
				for i, err := range errs {
					require.Equal(t, field.ErrorTypeForbidden, err.Type)
					paths[i] = err.Field
				}
				require.Equal(
					t,
					[]string{
						"spec.subscriptions[0].git.allowTags",
						"spec.subscriptions[0].git.ignoreTags",
						"spec.subscriptions[2].image.ignoreTags",
					},
					paths,
				)
				require.Contains(t, errs[0].Detail, "use allowTagsRegexes instead")
				require.Contains(t, errs[1].Detail, "use ignoreTagsRegexes instead")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			if testCase.req != nil {
				ctx = admission.NewContextWithRequest(ctx, *testCase.req)
			}
			testCase.assertions(
				t,
				validateRemovedSubFields(ctx, field.NewPath("spec", "subscriptions")),
			)
		})
	}
}

// decodeTestWarehouse decodes the raw, JSON-encoded Warehouse the same way the
// webhook's decoder would.
func decodeTestWarehouse(t *testing.T, raw string) *kargoapi.Warehouse {
	t.Helper()
	warehouse := &kargoapi.Warehouse{}
	require.NoError(t, json.Unmarshal([]byte(raw), warehouse))
	return warehouse
}
