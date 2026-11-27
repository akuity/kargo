package warehouses

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/metrics"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/subscription"
)

func TestSyncWarehouse_metrics(t *testing.T) {
	t.Parallel()

	discoveredAt := metav1.Now()
	histogramOf := func(t *testing.T, vec *prometheus.HistogramVec) *dto.Histogram {
		t.Helper()
		observer, ok := vec.WithLabelValues("fake-project").(prometheus.Metric)
		require.True(t, ok)
		var out dto.Metric
		require.NoError(t, observer.Write(&out))
		return out.GetHistogram()
	}
	createdOf := func(t *testing.T, m *metrics.WarehouseMetrics) float64 {
		t.Helper()
		var out dto.Metric
		require.NoError(t, m.FreightCreated.WithLabelValues("fake-project").Write(&out))
		return out.GetCounter().GetValue()
	}

	testCases := []struct {
		name      string
		previous  *kargoapi.DiscoveredArtifacts
		createErr error
		assert    func(*testing.T, *metrics.WarehouseMetrics)
	}{
		{
			name: "discovery after a previous one",
			previous: &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: metav1.Time{Time: discoveredAt.Add(-5 * time.Minute)},
			},
			assert: func(t *testing.T, m *metrics.WarehouseMetrics) {
				require.Equal(t, uint64(1), histogramOf(t, m.DiscoveryDuration).GetSampleCount())
				interval := histogramOf(t, m.DiscoveryInterval)
				require.Equal(t, uint64(1), interval.GetSampleCount())
				require.InDelta(t, 300, interval.GetSampleSum(), 0.001)
				require.Equal(t, 1.0, createdOf(t, m))
			},
		},
		{
			name: "first discovery observes no interval",
			assert: func(t *testing.T, m *metrics.WarehouseMetrics) {
				require.Equal(t, uint64(1), histogramOf(t, m.DiscoveryDuration).GetSampleCount())
				require.Zero(t, histogramOf(t, m.DiscoveryInterval).GetSampleCount())
			},
		},
		{
			name: "Freight that already exists is not counted",
			createErr: apierrors.NewAlreadyExists(
				schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "freights"},
				"fake-freight",
			),
			assert: func(t *testing.T, m *metrics.WarehouseMetrics) {
				require.Zero(t, createdOf(t, m))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			histogram := func(name string) *prometheus.HistogramVec {
				return prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, []string{"project"})
			}
			// WarehouseMetrics is a process-wide singleton; use standalone collectors.
			m := &metrics.WarehouseMetrics{
				DiscoveryDuration: histogram("test_discovery_duration_seconds"),
				DiscoveryInterval: histogram("test_discovery_interval_seconds"),
				FreightCreated: prometheus.NewCounterVec(
					prometheus.CounterOpts{Name: "test_freight_created_total"},
					[]string{"project"},
				),
			}
			r := &reconciler{
				whMetrics: m,
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: discoveredAt,
						Git: []kargoapi.GitDiscoveryResult{{
							RepoURL: "fake-repo",
							Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
						}},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{Name: "fake-freight", Namespace: "fake-project"},
						Commits:    []kargoapi.GitCommit{{ID: "fake-commit"}},
					}, nil
				},
				createFreightFn: func(
					_ context.Context,
					freight *kargoapi.Freight,
				) (*kargoapi.Freight, error) {
					if testCase.createErr != nil {
						return nil, testCase.createErr
					}
					return freight, nil
				},
				patchStatusFn: func(
					context.Context,
					*kargoapi.Warehouse,
					func(*kargoapi.WarehouseStatus),
				) error {
					return nil
				},
			}
			warehouse := &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "fake-project",
					Name:       "fake-warehouse",
					Generation: 1,
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
					Interval:              metav1.Duration{Duration: time.Minute},
				},
				Status: kargoapi.WarehouseStatus{
					ObservedGeneration:  1,
					DiscoveredArtifacts: testCase.previous,
				},
			}
			_, err := r.syncWarehouse(t.Context(), warehouse)
			require.NoError(t, err)
			testCase.assert(t, m)
		})
	}
}

func TestDiscoverArtifacts_failureMetrics(t *testing.T) {
	t.Parallel()

	registry := func(res any, err error) subscription.SubscriberRegistry {
		return subscription.MustNewSubscriberRegistry(subscription.SubscriberRegistration{
			Predicate: func(_ context.Context, sub kargoapi.RepoSubscription) (bool, error) {
				return sub.Subscription == nil, nil
			},
			Value: func(context.Context, credentials.Database) (subscription.Subscriber, error) {
				return &subscription.MockSubscriber{
					DiscoverArtifactsFn: func(context.Context, string, kargoapi.RepoSubscription, any) (any, error) {
						return res, err
					},
				}, nil
			},
		})
	}

	testCases := []struct {
		name     string
		registry subscription.SubscriberRegistry
		subs     []kargoapi.RepoSubscription
		expected map[string]float64
	}{
		{
			name:     "success",
			registry: registry(kargoapi.GitDiscoveryResult{}, nil),
			subs:     []kargoapi.RepoSubscription{{Git: &kargoapi.GitSubscription{}}},
			expected: map[string]float64{},
		},
		{
			name:     "subscriber error",
			registry: registry(nil, errors.New("something went wrong")),
			subs: []kargoapi.RepoSubscription{
				{Image: &kargoapi.ImageSubscription{}},
				{Git: &kargoapi.GitSubscription{}},
			},
			expected: map[string]float64{"image": 1},
		},
		{
			name:     "unrecognized result type",
			registry: registry("bogus", nil),
			subs:     []kargoapi.RepoSubscription{{Chart: &kargoapi.ChartSubscription{}}},
			expected: map[string]float64{"chart": 1},
		},
		{
			name:     "no subscriber",
			registry: registry(kargoapi.GitDiscoveryResult{}, nil),
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{}},
				{Subscription: &kargoapi.Subscription{SubscriptionType: "fake"}},
			},
			expected: map[string]float64{"other": 1},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// WarehouseMetrics is a process-wide singleton; use a standalone collector.
			failures := prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: "test_discovery_failures_total"},
				[]string{"project", "subscriptionType"},
			)
			r := &reconciler{
				subscriberRegistry: testCase.registry,
				whMetrics:          &metrics.WarehouseMetrics{DiscoveryFailures: failures},
			}
			_, err := r.discoverArtifacts(t.Context(), "fake-project", testCase.subs, nil)
			if len(testCase.expected) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, len(testCase.expected), testutil.CollectAndCount(failures))
			for subType, value := range testCase.expected {
				require.Equal(
					t,
					value,
					testutil.ToFloat64(failures.WithLabelValues("fake-project", subType)),
				)
			}
		})
	}
}
