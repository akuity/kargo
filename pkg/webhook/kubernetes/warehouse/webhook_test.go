package warehouse

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/subscription"
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
						// predictable change to the name field instead, because it will
						// give us a way to verify that subscriber-specific defaulting logic
						// works for generic subscriptions.
						sub.Subscription.Name = "fake"
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
		err := w.Default(context.Background(), warehouse)
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
		err := w.Default(context.Background(), warehouse)
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
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		require.Empty(t, warehouse.Spec.Shard)
		_, ok := warehouse.Labels[kargoapi.LabelKeyShard]
		require.False(t, ok)
	})

	t.Run("defaulting is delegated to subscribers", func(t *testing.T) {
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
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		const testDiscoveryLimit int64 = 42
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[0].Git.DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[1].Image.DiscoveryLimit)
		require.Equal(t, testDiscoveryLimit, warehouse.Spec.InternalSubscriptions[2].Chart.DiscoveryLimit)
		require.Equal(t, "fake", warehouse.Spec.InternalSubscriptions[3].Subscription.Name)
	})

	t.Run("common elements of generic subscriptions are defaulted", func(t *testing.T) {
		warehouse := &kargoapi.Warehouse{
			Spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					{Subscription: &kargoapi.Subscription{SubscriptionType: "fake"}},
				},
			},
		}
		err := w.Default(context.Background(), warehouse)
		require.NoError(t, err)
		require.Equal(
			t,
			defaultDiscoveryLimit,
			warehouse.Spec.InternalSubscriptions[0].Subscription.DiscoveryLimit,
		)
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.webhook.subscriberRegistry = subscription.DefaultSubscriberRegistry
			_, err := testCase.webhook.ValidateCreate(
				context.Background(),
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
		name       string
		webhook    *webhook
		warehouse  *kargoapi.Warehouse
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.webhook.subscriberRegistry = subscription.DefaultSubscriberRegistry
			_, err := testCase.webhook.ValidateUpdate(
				context.Background(),
				nil,
				testCase.warehouse,
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateDelete(t *testing.T) {
	w := &webhook{}
	_, err := w.ValidateDelete(context.Background(), nil)
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
					{Subscription: &kargoapi.Subscription{
						SubscriptionType: "fake",
						Name:             "fake-sub",
						DiscoveryLimit:   20,
					}},
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
						Subscription: &kargoapi.Subscription{
							// Name is empty and discovery limit is zero
							SubscriptionType: "fake",
						},
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
				require.Contains(t, fields, "spec.subscriptions[0].fake.name")
				require.Contains(t, fields, "spec.subscriptions[0].fake.discoveryLimit")
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
