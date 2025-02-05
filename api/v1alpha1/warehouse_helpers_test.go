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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// TODO(krancour): If we move our actual indexers to this package, we can use
// them here instead of duplicating them for the sake of avoiding an import
// cycle.
const warehouseField = "warehouse"

func warehouseIndexer(obj client.Object) []string {
	return []string{obj.(*Freight).Origin.Name} // nolint: forcetypeassert
}

const approvedField = "approvedFor"

func approvedForIndexer(obj client.Object) []string {
	freight := obj.(*Freight) // nolint: forcetypeassert
	var approvedFor []string
	for stage := range freight.Status.ApprovedFor {
		approvedFor = append(approvedFor, stage)
	}
	return approvedFor
}

const verifiedInField = "verifiedIn"

func verifiedInIndexer(obj client.Object) []string {
	freight := obj.(*Freight) // nolint: forcetypeassert
	var verifiedIn []string
	for stage := range freight.Status.VerifiedIn {
		verifiedIn = append(verifiedIn, stage)
	}
	return verifiedIn
}

func TestGetWarehouse(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Warehouse, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, warehouse *Warehouse, err error) {
				require.NoError(t, err)
				require.Nil(t, warehouse)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, warehouse *Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-warehouse", warehouse.Name)
				require.Equal(t, "fake-namespace", warehouse.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			warehouse, err := GetWarehouse(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-warehouse",
				},
			)
			testCase.assertions(t, warehouse, err)
		})
	}
}

func TestWarehouse_ListFreight(t *testing.T) {
	const testProject = "fake-project"
	const testWarehouse = "fake-warehouse"
	const testStage = "fake-stage"
	const testUpstreamStage = "fake-upstream-stage"
	const testUpstreamStage2 = "fake-upstream-stage2"

	testCases := []struct {
		name        string
		opts        *ListWarehouseFreightOptions
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
			name: "success with no options",
			objects: []client.Object{
				&Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
				},
				&Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "another-fake-freight",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: "wrong-warehouse",
					},
				},
				&Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "wrong-project",
						Name:      "another-fake-freight",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight", freight[0].Name)
			},
		},
		{
			name: "success with VerifiedIn and VerifiedBefore options",
			opts: &ListWarehouseFreightOptions{
				ApprovedFor:    testStage,
				VerifiedIn:     []string{testUpstreamStage},
				VerifiedBefore: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			objects: []client.Object{
				&Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: "wrong-warehouse",
					},
					Status: FreightStatus{
						// Doesn't matter that it's approved for the stage, because this is
						// the wrong warehouse
						ApprovedFor: map[string]ApprovedStage{testStage: {}},
						// Doesn't matter that it's verified upstream, because this is the
						// wrong warehouse
						VerifiedIn: map[string]VerifiedStage{testUpstreamStage: {}},
					},
				},
				&Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					// This is not approved or verified in any Stages
				},
				&Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is approved for the Stage
						ApprovedFor: map[string]ApprovedStage{testStage: {}},
					},
				},
				&Freight{ // This should not be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is verified in the upstream Stage, but the soak time has not
						// yet elapsed
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.Now()),
							},
						},
					},
				},
				&Freight{ // This should be returned
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-5",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is verified in the upstream Stage and the soak time has
						// elapsed
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))),
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
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
			opts: &ListWarehouseFreightOptions{
				AvailabilityStrategy: FreightAvailabilityStrategyAll,
				ApprovedFor:          testStage,
				VerifiedIn:           []string{testUpstreamStage, testUpstreamStage2},
				VerifiedBefore:       &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			objects: []client.Object{
				&Freight{ // This should be returned as its approved for the Stage
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is approved for the Stage
						ApprovedFor: map[string]ApprovedStage{testStage: {}},
						// This is only verified in a single upstream Stage
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.Now()),
							},
						},
					},
				},
				&Freight{ // This should be returned because its verified in both upstream Stages and soak time has lapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]ApprovedStage{},
						// This is verified in all of the upstream Stages and the soak time has lapsed
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))),
							},
							testUpstreamStage2: {
								VerifiedAt: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))),
							},
						},
					},
				},
				&Freight{ // This should not be returned because it's not verified in all of the upstream Stages
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]ApprovedStage{},
						// This is not verified in all of the upstream Stages
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.Now()),
							},
						},
					},
				},
				&Freight{ // This should not be returned because its not passed the soak time in all Stages
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: FreightOrigin{
						Kind: FreightOriginKindWarehouse,
						Name: testWarehouse,
					},
					Status: FreightStatus{
						// This is not approved for any Stage
						ApprovedFor: map[string]ApprovedStage{},
						// This is verified in all of the upstream Stages but only passed the soak time of one
						VerifiedIn: map[string]VerifiedStage{
							testUpstreamStage: {
								VerifiedAt: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))),
							},
							testUpstreamStage2: {
								VerifiedAt: ptr.To(metav1.Now()),
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				require.Equal(t, testProject, freight[0].Namespace)
				require.Equal(t, "fake-freight-1", freight[0].Name)
				require.Equal(t, testProject, freight[1].Namespace)
				require.Equal(t, "fake-freight-2", freight[1].Name)
			},
		},
		{
			name: "failure with invalid AvailabilityStrategy",
			opts: &ListWarehouseFreightOptions{
				AvailabilityStrategy: "Invalid",
				ApprovedFor:          testStage,
				VerifiedIn:           []string{testUpstreamStage, testUpstreamStage2},
				VerifiedBefore:       &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.ErrorContains(t, err, "unsupported AvailabilityStrategy")
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		c := fake.NewClientBuilder().WithScheme(testScheme).
			WithScheme(testScheme).
			WithIndex(&Freight{}, warehouseField, warehouseIndexer).
			WithIndex(&Freight{}, approvedField, approvedForIndexer).
			WithIndex(&Freight{}, verifiedInField, verifiedInIndexer).
			WithObjects(testCase.objects...).
			WithInterceptorFuncs(testCase.interceptor).
			Build()

		t.Run(testCase.name, func(t *testing.T) {
			warehouse := &Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "fake-warehouse",
				},
			}
			freight, err := warehouse.ListFreight(context.Background(), c, testCase.opts)
			testCase.assertions(t, freight, err)
		})
	}
}
