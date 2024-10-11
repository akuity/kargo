package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetFreight(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Freight, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, freight *Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, freight *Freight, err error) {
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
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Freight, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, freight *Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight",
						Namespace: "fake-namespace",
						Labels: map[string]string{
							AliasLabelKey: "fake-alias",
						},
					},
				},
			).Build(),
			assertions: func(t *testing.T, freight *Freight, err error) {
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
		stage    *Stage
		Freight  *Freight
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
			stage: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
			},
			Freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "wrong-namespace",
				},
			},
			expected: false,
		},
		{
			name: "freight is approved for stage",
			stage: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testStage,
				},
			},
			Freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testStage,
				},
				Status: FreightStatus{
					ApprovedFor: map[string]ApprovedStage{
						testStage: {},
					},
				},
			},
			expected: true,
		},
		{
			name: "stage accepts freight direct from origin",
			stage: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: FreightSources{
							Direct: true,
						},
					}},
				},
			},
			Freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: FreightOrigin{
					Kind: FreightOriginKindWarehouse,
					Name: testWarehouse,
				},
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream stage",
			stage: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			Freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: FreightOrigin{
					Kind: FreightOriginKindWarehouse,
					Name: testWarehouse,
				},
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight from origin not requested",
			stage: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			Freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Origin: FreightOrigin{
					Kind: FreightOriginKindWarehouse,
					Name: "wrong-warehouse",
				},
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
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
