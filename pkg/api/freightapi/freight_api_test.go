package freightapi

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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// TODO(krancour): If we move our actual indexers to this package, we can use
// them here instead of duplicating them for the sake of avoiding an import
// cycle.
const warehouseField = "warehouse"

func warehouseIndexer(obj client.Object) []string {
	return []string{obj.(*kargoapi.Freight).Origin.Name} // nolint: forcetypeassert
}

const approvedField = "approvedFor"

func approvedForIndexer(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	var approvedFor []string
	for stage := range freight.Status.ApprovedFor {
		approvedFor = append(approvedFor, stage)
	}
	return approvedFor
}

const verifiedInField = "verifiedIn"

func verifiedInIndexer(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	var verifiedIn []string
	for stage := range freight.Status.VerifiedIn {
		verifiedIn = append(verifiedIn, stage)
	}
	return verifiedIn
}

const currentlyInField = "currentlyIn"

func currentlyInIndexer(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	var currentlyIn []string
	for stage := range freight.Status.CurrentlyIn {
		currentlyIn = append(currentlyIn, stage)
	}
	return currentlyIn
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
			WithIndex(&kargoapi.Freight{}, currentlyInField, currentlyInIndexer).
			WithObjects(testCase.objects...).
			WithInterceptorFuncs(testCase.interceptor).
			Build()

		t.Run(testCase.name, func(t *testing.T) {
			freight, err := ListFreightByCurrentStage(t.Context(), c, testProject, testStage.Name)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestListFreightFromOrigin(t *testing.T) {
	const testProject = "fake-project"
	const testWarehouse = "fake-warehouse"
	const testStage = "fake-stage"
	const testUpstreamStage = "fake-upstream-stage"
	const testUpstreamStage2 = "fake-upstream-stage2"
	const testUnrelatedStage = "fake-unrelated-stage"

	testCases := []struct {
		name        string
		opts        *ListFreightFromOriginOptions
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
			name: "success with no options",
			objects: []client.Object{
				&kargoapi.Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
				},
				&kargoapi.Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "another-fake-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "wrong-warehouse",
					},
				},
				&kargoapi.Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "wrong-project",
						Name:      "another-fake-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight", freight[0].Name)
			},
		},
		{
			name: "success with VerifiedIn and RequiredSoakTime options",
			opts: &ListFreightFromOriginOptions{
				ApprovedFor:      testStage,
				VerifiedIn:       []string{testUpstreamStage},
				RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
			},
			objects: []client.Object{
				&kargoapi.Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "wrong-warehouse",
					},
					Status: kargoapi.FreightStatus{
						// Doesn't matter that it's approved for the stage, because this is
						// the wrong warehouse
						ApprovedFor: map[string]kargoapi.ApprovedStage{testStage: {}},
						// Doesn't matter that it's verified upstream, because this is the
						// wrong warehouse
						VerifiedIn: map[string]kargoapi.VerifiedStage{testUpstreamStage: {
							LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
						}},
					},
				},
				&kargoapi.Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					// This is not approved or verified in any Stages
				},
				&kargoapi.Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is approved for the Stage
						ApprovedFor: map[string]kargoapi.ApprovedStage{testStage: {}},
					},
				},
				&kargoapi.Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is verified in the upstream Stage, but the soak time has not
						// yet elapsed
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute},
							},
						},
					},
				},
				&kargoapi.Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-5",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is verified in the upstream Stage and the soak time has
						// elapsed
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-3", freight[0].Name)
				require.Equal(t, testProject, freight[1].Namespace)
				require.Equal(t, "fake-freight-5", freight[1].Name)
			},
		},
		{
			name: "success with AvailabilityStrategy set to FreightAvailabilityStrategyAll",
			opts: &ListFreightFromOriginOptions{
				AvailabilityStrategy: kargoapi.FreightAvailabilityStrategyAll,
				ApprovedFor:          testStage,
				VerifiedIn:           []string{testUpstreamStage, testUpstreamStage2},
				RequiredSoakTime:     &metav1.Duration{Duration: time.Hour},
			},
			objects: []client.Object{
				&kargoapi.Freight{ // This should be returned as its approved for the Stage
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is approved for the Stage
						ApprovedFor: map[string]kargoapi.ApprovedStage{testStage: {}},
						// This is only verified in a single upstream Stage
						VerifiedIn: map[string]kargoapi.VerifiedStage{},
					},
				},
				&kargoapi.Freight{
					// This should be returned because its verified in both upstream Stages and soak time has lapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]kargoapi.ApprovedStage{},
						// This is verified in all of the upstream Stages and the soak time has lapsed
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
							testUpstreamStage2: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
				&kargoapi.Freight{ // This should not be returned because it's not verified in all of the upstream Stages
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]kargoapi.ApprovedStage{},
						// This is not verified in all of the upstream Stages
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {},
						},
					},
				},
				&kargoapi.Freight{ // This should not be returned because its not passed the soak time in all Stages
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]kargoapi.ApprovedStage{},
						// This is verified in all of the upstream Stages but only passed the soak time of one
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
							testUpstreamStage2: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-1", freight[0].Name)
				require.Equal(t, testProject, freight[1].Namespace)
				require.Equal(t, "fake-freight-2", freight[1].Name)
			},
		},
		{
			name: "soak time elapsed in an unrelated Stage does not count",
			opts: &ListFreightFromOriginOptions{
				VerifiedIn:       []string{testUpstreamStage},
				RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
			},
			objects: []client.Object{
				// This should not be returned because the soak time has only
				// elapsed in a Stage that isn't upstream.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute},
							},
							testUnrelatedStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Empty(t, freight)
			},
		},
		{
			name: "soak time elapsed in an unrelated Stage does not prevent availability",
			opts: &ListFreightFromOriginOptions{
				AvailabilityStrategy: kargoapi.FreightAvailabilityStrategyAll,
				VerifiedIn:           []string{testUpstreamStage},
				RequiredSoakTime:     &metav1.Duration{Duration: time.Hour},
			},
			objects: []client.Object{
				// This should be returned because the soak time has elapsed in
				// every upstream Stage. Having also elapsed in a Stage that isn't
				// upstream doesn't matter.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
							testUnrelatedStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, "fake-freight", freight[0].Name)
			},
		},
		{
			name: "success listing Freight currently in upstream Stage",
			opts: &ListFreightFromOriginOptions{
				AvailabilityStrategy: kargoapi.FreightAvailabilityStrategyAll,
				ApprovedFor:          testStage,
				VerifiedIn:           []string{testUpstreamStage},
				RequiredSoakTime:     &metav1.Duration{Duration: time.Hour},
				CurrentlyIn:          testUpstreamStage,
			},
			objects: []client.Object{
				// This would ordinarily be returned because it is approved for the
				// Stage, verified in the upstream Stage, and the soak time has elapsed,
				// BUT this test case asks only for Freight that's currently in the
				// upstream Stage.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is approved for the Stage.
						ApprovedFor: map[string]kargoapi.ApprovedStage{testStage: {}},
						// This is verified in the upstream Stage and the soak time has
						// elapsed.
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
				// This should be returned because: Its verified upstream, soak time has
				// elapsed, and it's currently in use by the upstream Stage.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: kargoapi.FreightStatus{
						// This is approved for the Stage.
						ApprovedFor: map[string]kargoapi.ApprovedStage{testStage: {}},
						// This is verified in the upstream Stage and the soak time has
						// elapsed.
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testUpstreamStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
						// This Freight is currently in the upstream Stage.
						CurrentlyIn: map[string]kargoapi.CurrentStage{testUpstreamStage: {}},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-2", freight[0].Name)
			},
		},
		{
			name: "failure with invalid AvailabilityStrategy",
			opts: &ListFreightFromOriginOptions{
				AvailabilityStrategy: "Invalid",
				ApprovedFor:          testStage,
				VerifiedIn:           []string{testUpstreamStage, testUpstreamStage2},
				RequiredSoakTime:     &metav1.Duration{Duration: time.Hour},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "unsupported AvailabilityStrategy")
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		c := fake.NewClientBuilder().WithScheme(testScheme).
			WithScheme(testScheme).
			WithIndex(&kargoapi.Freight{}, warehouseField, warehouseIndexer).
			WithIndex(&kargoapi.Freight{}, approvedField, approvedForIndexer).
			WithIndex(&kargoapi.Freight{}, verifiedInField, verifiedInIndexer).
			WithIndex(&kargoapi.Freight{}, currentlyInField, currentlyInIndexer).
			WithObjects(testCase.objects...).
			WithInterceptorFuncs(testCase.interceptor).
			Build()

		t.Run(testCase.name, func(t *testing.T) {
			freight, err := ListFreightFromOrigin(
				t.Context(), c, testProject,
				// TODO: add different origins when implemented
				kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "fake-warehouse",
				}, testCase.opts,
			)
			testCase.assertions(t, freight, err)
		})
	}
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
