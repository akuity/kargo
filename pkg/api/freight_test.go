package api

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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGenerateFreightID(t *testing.T) {
	freight := kargoapi.Freight{
		Origin: kargoapi.FreightOrigin{
			Kind: "fake-kind",
			Name: "fake-name",
		},
		Commits: []kargoapi.GitCommit{
			{
				RepoURL: "fake-git-repo",
				ID:      "fake-commit-id",
			},
		},
		Images: []kargoapi.Image{
			{
				RepoURL: "fake-image-repo",
				Tag:     "fake-image-tag",
			},
		},
		Charts: []kargoapi.Chart{
			{
				RepoURL: "fake-chart-repo",
				Name:    "fake-chart",
				Version: "fake-chart-version",
			},
		},
	}
	id := GenerateFreightID(&freight)
	expected := id
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		require.Equal(t, expected, GenerateFreightID(&freight))
	}
	// Changing anything should change the result
	freight.Commits[0].ID = "a-different-fake-commit"
	require.NotEqual(t, expected, GenerateFreightID(&freight))
}

// TODO(krancour): If we move our actual indexers to this package, we can use
// them here instead of duplicating them for the sake of avoiding an import
// cycle.
const freightByCurrentStagesField = "currentlyIn"

func freightByCurrentStagesIndexer(obj client.Object) []string {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return nil
	}
	currentStages := make([]string, 0, len(freight.Status.CurrentlyIn))
	for stage := range freight.Status.CurrentlyIn {
		currentStages = append(currentStages, stage)
	}
	return currentStages
}

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
				t.Context(),
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

func TestGetCurrentFreight(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	const testProject = "fake-project"
	originA := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "warehouse-a"}
	originB := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "warehouse-b"}

	twoOriginStage := func() *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "fake-stage"},
			Status: kargoapi.StageStatus{
				FreightHistory: kargoapi.FreightHistory{{
					Freight: map[string]kargoapi.FreightReference{
						originA.String(): {Name: "freight-a", Origin: originA},
						originB.String(): {Name: "freight-b", Origin: originB},
					},
				}},
			},
		}
	}

	testCases := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, map[string]*kargoapi.Freight, error)
	}{
		{
			name:  "resolves current Freight for each origin",
			stage: twoOriginStage(),
			objects: []client.Object{
				&kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "freight-a"}},
				&kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "freight-b"}},
			},
			assertions: func(t *testing.T, current map[string]*kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, current, 2)
				require.Equal(t, "freight-a", current[originA.String()].Name)
				require.Equal(t, "freight-b", current[originB.String()].Name)
			},
		},
		{
			name:  "omits origin whose Freight is gone",
			stage: twoOriginStage(),
			objects: []client.Object{
				// freight-b is intentionally absent (garbage-collected).
				&kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "freight-a"}},
			},
			assertions: func(t *testing.T, current map[string]*kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, current, 1)
				require.Contains(t, current, originA.String())
				require.NotContains(t, current, originB.String())
			},
		},
		{
			name:  "empty when no current Freight",
			stage: &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "fake-stage"}},
			assertions: func(t *testing.T, current map[string]*kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Empty(t, current)
			},
		},
		{
			name:  "fetch error fails closed",
			stage: twoOriginStage(),
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, current map[string]*kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error getting current Freight")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, current)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			current, err := GetCurrentFreight(t.Context(), c, testCase.stage)
			testCase.assertions(t, current, err)
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
							kargoapi.LabelKeyAlias: "fake-alias",
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
				t.Context(),
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
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "fake-stage",
		},
	}

	testCases := []struct {
		name        string
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []kargoapi.Freight, error)
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
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error listing Freight")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			objects: []client.Object{
				&kargoapi.Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // This should NOT be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{"wrong-stage": {}},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-1", freight[0].Name)
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		c := fake.NewClientBuilder().WithScheme(testScheme).
			WithScheme(testScheme).
			WithIndex(&kargoapi.Freight{}, freightByCurrentStagesField, freightByCurrentStagesIndexer).
			WithObjects(testCase.objects...).
			WithInterceptorFuncs(testCase.interceptor).
			Build()

		t.Run(testCase.name, func(t *testing.T) {
			freight, err := ListFreightByCurrentStage(t.Context(), c, testStage)
			testCase.assertions(t, freight, err)
		})
	}
}
