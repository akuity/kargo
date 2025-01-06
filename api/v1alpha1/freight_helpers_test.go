package v1alpha1

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// TODO(krancour): If we move our actual indexers to this package, we can use
// them here instead of duplicating them for the sake of avoiding an import
// cycle.
const freightByCurrentStagesField = "currentlyIn"

func freightByCurrentStagesIndexer(obj client.Object) []string {
	freight, ok := obj.(*Freight)
	if !ok {
		return nil
	}
	currentStages := make([]string, len(freight.Status.CurrentlyIn))
	var i int
	for stage := range freight.Status.CurrentlyIn {
		currentStages[i] = stage
		i++
	}
	return currentStages
}

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

func TestListFreightByCurrentStage(t *testing.T) {
	const testProject = "fake-project"
	testStage := &Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "fake-stage",
		},
	}

	testCases := []struct {
		name        string
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []Freight, error)
	}{
		{
			name: "error listing Freight",
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.ErrorContains(t, err, "error listing Freight")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			objects: []client.Object{
				&Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: FreightStatus{
						CurrentlyIn: map[string]CurrentStage{testStage.Name: {}},
					},
				},
				&Freight{ // This should NOT be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: FreightStatus{
						CurrentlyIn: map[string]CurrentStage{"wrong-stage": {}},
					},
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-1", freight[0].Name)
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		c := fake.NewClientBuilder().WithScheme(testScheme).
			WithScheme(testScheme).
			WithIndex(&Freight{}, freightByCurrentStagesField, freightByCurrentStagesIndexer).
			WithObjects(testCase.objects...).
			WithInterceptorFuncs(testCase.interceptor).
			Build()

		t.Run(testCase.name, func(t *testing.T) {
			freight, err := ListFreightByCurrentStage(context.Background(), c, testStage)
			testCase.assertions(t, freight, err)
		})
	}
}
