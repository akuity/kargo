package v1alpha1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWarehouse_GetInterval(t *testing.T) {
	tests := []struct {
		name        string
		warehouse   *Warehouse
		minInterval time.Duration
		assertions  func(t *testing.T, w *Warehouse, interval time.Duration, minInterval time.Duration)
	}{
		{
			name: "no discovery has taken place yet, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, warehouse *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, warehouse.Spec.Interval.Duration, interval)
			},
		},
		{
			name: "no discovery has taken place yet, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, warehouse *Warehouse, interval time.Duration, minInterval time.Duration) {
				require.Equal(t, minInterval, interval)
				require.Greater(t, interval, warehouse.Spec.Interval.Duration)
			},
		},
		{
			name: "next discovery is overdue, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-6 * time.Minute)),
					},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, _ *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, interval, 100*time.Millisecond)
			},
		},
		{
			name: "next discovery is overdue, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-6 * time.Minute)),
					},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, _ *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, interval, 100*time.Millisecond)
			},
		},
		{
			name: "next discovery is not overdue, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-3 * time.Minute)),
					},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, w *Warehouse, interval time.Duration, _ time.Duration) {
				require.NotZero(t, interval)
				require.Less(t, interval, w.Spec.Interval.Duration)
				// Should be around 2 minutes (5 - 3)
				require.InDelta(t, 2*time.Minute, interval, float64(10*time.Second))
			},
		},
		{
			name: "next discovery is not overdue, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, w *Warehouse, interval time.Duration, _ time.Duration) {
				require.NotZero(t, interval)
				// Should be around 4 minutes (5 - 1) since effective interval is min (5 minutes)
				require.InDelta(t, 4*time.Minute, interval, float64(10*time.Second))
				require.Greater(t, interval, w.Spec.Interval.Duration)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.warehouse, tt.warehouse.GetInterval(tt.minInterval), tt.minInterval)
		})
	}
}

func TestArtifactReference_DeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *ArtifactReference
		b              *ArtifactReference
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &ArtifactReference{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &ArtifactReference{},
			expectedResult: false,
		},
		{
			name: "artifact type differs",
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			b: &ArtifactReference{
				ArtifactType:     "wrong-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			expectedResult: false,
		},
		{
			name: "subscription name differs",
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			b: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "bar",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			expectedResult: false,
		},
		{
			name: "version differs",
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			b: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.1.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			expectedResult: false,
		},
		{
			name: "metadata differs",
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			b: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value2"}`),
				},
			},
			expectedResult: false,
		},
		{
			name: "perfect match without metadata",
			// This is to verify the short-circuiting when both Metadata are nil
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
			},
			b: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
			},
			expectedResult: true,
		},
		{
			name: "perfect match with metadata",
			a: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			b: &ArtifactReference{
				ArtifactType:     "fake-artifact-type",
				SubscriptionName: "foo",
				Version:          "v1.0.0",
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"key":"value1"}`),
				},
			},
			expectedResult: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.a.DeepEquals(testCase.b))
			require.Equal(t, testCase.expectedResult, testCase.b.DeepEquals(testCase.a))
		})
	}
}

func TestWarehouseSpecMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		spec      *WarehouseSpec
		assertion func(
			t *testing.T,
			original *WarehouseSpec,
			roundtripped *WarehouseSpec,
		)
	}{
		{
			name: "single Git subscription",
			spec: &WarehouseSpec{
				Shard:    "test-shard",
				Interval: metav1.Duration{Duration: 5 * time.Minute},
				InternalSubscriptions: []RepoSubscription{{
					Git: &GitSubscription{
						RepoURL: "https://github.com/example/repo.git",
						Branch:  "main",
					},
				}},
			},
			assertion: func(
				t *testing.T,
				original *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 1)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Git)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Git.RepoURL,
					roundtripped.InternalSubscriptions[0].Git.RepoURL,
				)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Git.Branch,
					roundtripped.InternalSubscriptions[0].Git.Branch,
				)
			},
		},
		{
			name: "single Image subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Image: &ImageSubscription{
							RepoURL:                "nginx",
							ImageSelectionStrategy: ImageSelectionStrategySemVer,
						},
					},
				},
			},
			assertion: func(
				t *testing.T,
				original *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 1)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Image)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Image.RepoURL,
					roundtripped.InternalSubscriptions[0].Image.RepoURL,
				)
			},
		},
		{
			name: "single Chart subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Chart: &ChartSubscription{
							RepoURL: "https://charts.example.com",
							Name:    "my-chart",
						},
					},
				},
			},
			assertion: func(
				t *testing.T,
				original *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 1)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Chart)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Chart.RepoURL,
					roundtripped.InternalSubscriptions[0].Chart.RepoURL,
				)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Chart.Name,
					roundtripped.InternalSubscriptions[0].Chart.Name,
				)
			},
		},
		{
			name: "single generic subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Subscription: &Subscription{
							SubscriptionType: "s3",
							Name:             "my-bucket",
						},
					},
				},
			},
			assertion: func(
				t *testing.T,
				original *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 1)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Subscription)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Subscription.SubscriptionType,
					roundtripped.InternalSubscriptions[0].Subscription.SubscriptionType,
				)
				require.Equal(
					t,
					original.InternalSubscriptions[0].Subscription.Name,
					roundtripped.InternalSubscriptions[0].Subscription.Name,
				)
			},
		},
		{
			name: "mixed subscriptions",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Git: &GitSubscription{
							RepoURL: "https://github.com/example/repo.git",
						},
					},
					{
						Image: &ImageSubscription{
							RepoURL: "nginx",
						},
					},
					{
						Chart: &ChartSubscription{
							RepoURL: "https://charts.example.com",
							Name:    "my-chart",
						},
					},
					{
						Subscription: &Subscription{
							SubscriptionType: "custom",
							Name:             "custom-sub",
						},
					},
				},
			},
			assertion: func(
				t *testing.T,
				_ *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 4)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Git)
				require.NotNil(t, roundtripped.InternalSubscriptions[1].Image)
				require.NotNil(t, roundtripped.InternalSubscriptions[2].Chart)
				require.NotNil(t, roundtripped.InternalSubscriptions[3].Subscription)
			},
		},
		{
			name: "generic subscription with config",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Subscription: &Subscription{
							SubscriptionType: "http",
							Name:             "api-endpoint",
							Config: &apiextensionsv1.JSON{
								Raw: []byte(`{"url":"https://api.example.com","interval":"1h"}`),
							},
						},
					},
				},
			},
			assertion: func(
				t *testing.T,
				_ *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Len(t, roundtripped.InternalSubscriptions, 1)
				require.NotNil(t, roundtripped.InternalSubscriptions[0].Subscription)
				require.Equal(
					t,
					"http",
					roundtripped.InternalSubscriptions[0].Subscription.SubscriptionType,
				)
				require.NotNil(
					t,
					roundtripped.InternalSubscriptions[0].Subscription.Config,
				)
			},
		},
		{
			name: "empty subscriptions",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{},
			},
			assertion: func(
				t *testing.T,
				_ *WarehouseSpec,
				roundtripped *WarehouseSpec,
			) {
				require.Empty(t, roundtripped.InternalSubscriptions)
				require.Empty(t, roundtripped.Subscriptions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonData, err := json.Marshal(tt.spec)
			require.NoError(t, err)

			// Unmarshal back
			roundtripped := &WarehouseSpec{}
			err = json.Unmarshal(jsonData, roundtripped)
			require.NoError(t, err)

			// Run assertion
			tt.assertion(t, tt.spec, roundtripped)
		})
	}
}

func TestWarehouseSpecUnmarshalValidation(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid single Git subscription",
			jsonData:    `{"subscriptions":[{"git":{"repoURL":"https://github.com/example/repo.git"}}]}`,
			expectError: false,
		},
		{
			name:        "valid single Image subscription",
			jsonData:    `{"subscriptions":[{"image":{"repoURL":"nginx"}}]}`,
			expectError: false,
		},
		{
			name:        "valid single Chart subscription",
			jsonData:    `{"subscriptions":[{"chart":{"repoURL":"https://charts.example.com"}}]}`,
			expectError: false,
		},
		{
			name:        "valid generic subscription",
			jsonData:    `{"subscriptions":[{"s3":{"kind":"s3","name":"my-bucket"}}]}`,
			expectError: false,
		},
		{
			name: "multiple subscriptions",
			// nolint: lll
			jsonData:    `{"subscriptions":[{"git":{"repoURL":"https://github.com/example/repo.git"}},{"image":{"repoURL":"nginx"}},{"s3":{"kind":"s3","name":"bucket"}}]}`,
			expectError: false,
		},
		{
			name:        "invalid subscription with no fields",
			jsonData:    `{"subscriptions":[{}]}`,
			expectError: true,
			errorMsg:    "must be an object with exactly one top-level field",
		},
		{
			name: "invalid subscription with multiple fields",
			// nolint: lll
			jsonData:    `{"subscriptions":[{"git":{"repoURL":"https://github.com/example/repo.git"},"image":{"repoURL":"nginx"}}]}`,
			expectError: true,
			errorMsg:    "must be an object with exactly one top-level field",
		},
		{
			name:        "invalid subscription as scalar",
			jsonData:    `{"subscriptions":["not an object"]}`,
			expectError: true,
		},
		{
			name:        "invalid subscription as array",
			jsonData:    `{"subscriptions":[[{"git":{"repoURL":"https://github.com/example/repo.git"}}]]}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &WarehouseSpec{}
			err := json.Unmarshal([]byte(tt.jsonData), spec)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWarehouseSpecMarshalValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		spec        *WarehouseSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid single Git subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Git: &GitSubscription{
							RepoURL: "https://github.com/example/repo.git",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid single Image subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Image: &ImageSubscription{
							RepoURL: "nginx",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid single Chart subscription",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Chart: &ChartSubscription{
							RepoURL: "https://charts.example.com",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid generic subscription with kind",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Subscription: &Subscription{
							SubscriptionType: "s3",
							Name:             "my-bucket",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid subscription with multiple types set",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Git: &GitSubscription{
							RepoURL: "https://github.com/example/repo.git",
						},
						Image: &ImageSubscription{
							RepoURL: "nginx",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "must have exactly one of Git, Image, Chart, or Subscription set",
		},
		{
			name: "invalid subscription with no types set",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{},
				},
			},
			expectError: true,
			errorMsg:    "must have exactly one of Git, Image, Chart, or Subscription set",
		},
		{
			name: "invalid generic subscription with empty subscription type",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Subscription: &Subscription{
							Name: "my-bucket",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "empty SubscriptionType field",
		},
		{
			name: "invalid multiple subscriptions with error in second",
			spec: &WarehouseSpec{
				InternalSubscriptions: []RepoSubscription{
					{
						Git: &GitSubscription{
							RepoURL: "https://github.com/example/repo.git",
						},
					},
					{
						Git: &GitSubscription{
							RepoURL: "https://github.com/example/repo.git",
						},
						Chart: &ChartSubscription{
							RepoURL: "https://charts.example.com",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "subscription at index 1 must have exactly one of Git, Image, Chart, or Subscription set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.Marshal(tt.spec)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
