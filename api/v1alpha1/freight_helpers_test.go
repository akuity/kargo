package v1alpha1

import (
	"context"
	"errors"
	"testing"
	"time"

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
	currentStages := make([]string, 0, len(freight.Status.CurrentlyIn))
	for stage := range freight.Status.CurrentlyIn {
		currentStages = append(currentStages, stage)
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

func TestFreight_IsCurrentlyIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsCurrentlyIn(testStage))
	freight.Status.CurrentlyIn = map[string]CurrentStage{testStage: {}}
	require.True(t, freight.IsCurrentlyIn(testStage))
}

func TestFreight_IsVerifiedIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsVerifiedIn(testStage))
	freight.Status.VerifiedIn = map[string]VerifiedStage{testStage: {}}
	require.True(t, freight.IsVerifiedIn(testStage))
}

func TestFreight_IsApprovedFor(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsApprovedFor(testStage))
	freight.Status.ApprovedFor = map[string]ApprovedStage{testStage: {}}
	require.True(t, freight.IsApprovedFor(testStage))
}

func TestFreight_GetLongestSoak(t *testing.T) {
	testStage := "fake-stage"
	testCases := []struct {
		name       string
		status     FreightStatus
		assertions func(t *testing.T, status FreightStatus, longestSoak time.Duration)
	}{
		{
			name: "Freight is not currently in the Stage and was never verified there",
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is not currently in the Stage but was verified there",
			status: FreightStatus{
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Equal(t, time.Hour, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage but was never verified there",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; current soak is longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; a previous soak was longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight := &Freight{
				Status: testCase.status,
			}
			testCase.assertions(t, freight.Status, freight.GetLongestSoak(testStage))
		})
	}
}

func TestFreightStatus_AddCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already in current", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddCurrentStage(testStage, newTime)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, oldTime, record.Since.Time)
	})
	t.Run("not already in current", func(t *testing.T) {
		status := FreightStatus{}
		status.AddCurrentStage(testStage, now)
		require.NotNil(t, status.CurrentlyIn)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, now, record.Since.Time)
	})
}

func TestFreightStatus_RemoveCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	t.Run("not verified", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
	})
	t.Run("verified; old soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, 2*time.Hour, record.LongestCompletedSoak.Duration)
	})
	t.Run("verified; new soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		// Expect these to be equal within a second. TODO(krancour): There's probably a
		// more elegant way to do this, but I consider good enough.
		require.GreaterOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour)
		require.LessOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour+time.Second)
	})
}

func TestFreightStatus_AddVerifiedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already verified", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			VerifiedIn: map[string]VerifiedStage{
				testStage: {VerifiedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddVerifiedStage(testStage, newTime)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, oldTime, record.VerifiedAt.Time)
	})
	t.Run("not already verified", func(t *testing.T) {
		status := FreightStatus{}
		testTime := time.Now()
		status.AddVerifiedStage(testStage, testTime)
		require.NotNil(t, status.VerifiedIn)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, testTime, record.VerifiedAt.Time)
	})
}

func TestFreightStatus_AddApprovedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already approved", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			ApprovedFor: map[string]ApprovedStage{
				testStage: {ApprovedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddApprovedStage(testStage, newTime)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, oldTime, record.ApprovedAt.Time)
	})
	t.Run("not already approved", func(t *testing.T) {
		status := FreightStatus{}
		status.AddApprovedStage(testStage, now)
		require.NotNil(t, status.ApprovedFor)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, now, record.ApprovedAt.Time)
	})
}
