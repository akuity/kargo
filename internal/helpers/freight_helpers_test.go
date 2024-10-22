package helpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetFreight(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-freight", freight.Name)
				require.Equal(t, "fake-namespace", freight.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := GetFreight(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-freight",
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetFreightByAlias(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight",
						Namespace: "fake-namespace",
						Labels: map[string]string{
							kargoapi.AliasLabelKey: "fake-alias",
						},
					},
				},
			).Build(),
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-freight", freight.Name)
				require.Equal(t, "fake-namespace", freight.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := GetFreightByAlias(
				context.Background(),
				testCase.client,
				"fake-namespace",
				"fake-alias",
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestIsFreightAvailable(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testWarehouse = "fake-warehouse"
	const testStage = "fake-stage"

	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		Freight  *kargoapi.Freight
		expected bool
	}{
		{
			name:     "stage is nil",
			expected: false,
		},
		{
			name:     "freight is nil",
			expected: false,
		},
		{
			name: "stage and freight are in different namespaces",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
			},
			Freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "wrong-namespace",
				},
			},
			expected: false,
		},
		{
			name: "freight is approved for stage",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testStage,
				},
			},
			Freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testStage,
				},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{
						testStage: {},
					},
				},
			},
			expected: true,
		},
		{
			name: "stage accepts freight direct from origin",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: kargoapi.FreightSources{
							Direct: true,
						},
					}},
				},
			},
			Freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: testWarehouse,
				},
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream stage",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: kargoapi.FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			Freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: testWarehouse,
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight from origin not requested",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: kargoapi.FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			Freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "wrong-warehouse",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				IsFreightAvailable(testCase.stage, testCase.Freight),
			)
		})
	}
}
