package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetStage(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Stage, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Nil(t, stage)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-stage", stage.Name)
				require.Equal(t, "fake-namespace", stage.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stage, err := GetStage(
				t.Context(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-stage",
				},
			)
			testCase.assertions(t, stage, err)
		})
	}
}

func TestListStagesByWarehouses(t *testing.T) {
	const testProject = "fake-namespace"
	const otherProject = "other-namespace"
	const testWarehouse1 = "fake-warehouse1"
	const testWarehouse2 = "fake-warehouse2"

	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	stageInProjectFromWarehouse1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "stage-1",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: testWarehouse1,
				},
			}},
		},
	}
	stageInProjectFromWarehouse2 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "stage-2",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: testWarehouse2,
				},
			}},
		},
	}
	stageInOtherProject := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: otherProject,
			Name:      "stage-3",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: testWarehouse1,
				},
			}},
		},
	}

	testCases := []struct {
		name        string
		opts        *ListStagesOptions
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []kargoapi.Stage, error)
	}{
		{
			name: "error listing Stages",
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
			assertions: func(t *testing.T, stages []kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, stages)
			},
		},
		{
			name: "nil opts returns all Stages in Project",
			objects: []client.Object{
				stageInProjectFromWarehouse1,
				stageInProjectFromWarehouse2,
				stageInOtherProject,
			},
			assertions: func(t *testing.T, stages []kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Len(t, stages, 2)
			},
		},
		{
			name: "empty Warehouses filter returns all Stages in Project",
			opts: &ListStagesOptions{},
			objects: []client.Object{
				stageInProjectFromWarehouse1,
				stageInProjectFromWarehouse2,
				stageInOtherProject,
			},
			assertions: func(t *testing.T, stages []kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Len(t, stages, 2)
			},
		},
		{
			name: "Warehouses filter returns only matching Stages",
			opts: &ListStagesOptions{
				Warehouses: []string{testWarehouse1},
			},
			objects: []client.Object{
				stageInProjectFromWarehouse1,
				stageInProjectFromWarehouse2,
				stageInOtherProject,
			},
			assertions: func(t *testing.T, stages []kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Len(t, stages, 1)
				require.Equal(t, "stage-1", stages[0].Name)
			},
		},
		{
			name: "Warehouses filter with no matches returns empty",
			opts: &ListStagesOptions{
				Warehouses: []string{"unknown-warehouse"},
			},
			objects: []client.Object{
				stageInProjectFromWarehouse1,
				stageInProjectFromWarehouse2,
			},
			assertions: func(t *testing.T, stages []kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, stages)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			stages, err := ListStagesByWarehouses(
				t.Context(), c, testProject, testCase.opts,
			)
			testCase.assertions(t, stages, err)
		})
	}
}

func TestStageMatchesAnyWarehouse(t *testing.T) {
	const testWarehouse1 = "fake-warehouse1"
	const testWarehouse2 = "fake-warehouse2"

	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		warehouses []string
		expected   bool
	}{
		{
			name:       "no requested freight",
			stage:      &kargoapi.Stage{},
			warehouses: []string{testWarehouse1},
			expected:   false,
		},
		{
			name: "empty warehouses list",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse1,
						},
					}},
				},
			},
			warehouses: nil,
			expected:   false,
		},
		{
			name: "no matching warehouse",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse1,
						},
					}},
				},
			},
			warehouses: []string{testWarehouse2},
			expected:   false,
		},
		{
			name: "name matches but origin kind is not Warehouse",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKind("OtherKind"),
							Name: testWarehouse1,
						},
					}},
				},
			},
			warehouses: []string{testWarehouse1},
			expected:   false,
		},
		{
			name: "single requested freight matches",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse1,
						},
					}},
				},
			},
			warehouses: []string{testWarehouse1},
			expected:   true,
		},
		{
			name: "matches one of multiple warehouses",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse2,
						},
					}},
				},
			},
			warehouses: []string{testWarehouse1, testWarehouse2},
			expected:   true,
		},
		{
			name: "matches one of multiple requested freight",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "unrelated-warehouse",
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: testWarehouse1,
							},
						},
					},
				},
			},
			warehouses: []string{testWarehouse1},
			expected:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				StageMatchesAnyWarehouse(testCase.stage, testCase.warehouses),
			)
		})
	}
}

func TestListFreightAvailableToStage(t *testing.T) {
	const testProject = "fake-namespace"
	const testWarehouse1 = "fake-warehouse1"
	const testWarehouse2 = "fake-warehouse2"
	const testStage = "fake-stage"

	testWarehouse1Origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: testWarehouse1,
	}

	testWarehouse2Origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: testWarehouse2,
	}

	testCases := []struct {
		name        string
		reqs        []kargoapi.FreightRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error getting Warehouse",
			reqs: []kargoapi.FreightRequest{{}},
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
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error getting Warehouse")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "Warehouse not found",
			reqs: []kargoapi.FreightRequest{{}},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "Warehouse")
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "error listing Freight",
			reqs: []kargoapi.FreightRequest{{Origin: testWarehouse1Origin}},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testWarehouse1,
					},
				},
			},
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
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error listing Freight for Warehouse")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reqs: []kargoapi.FreightRequest{
				{
					Origin:  testWarehouse1Origin,
					Sources: kargoapi.FreightSources{Direct: true},
				},
				{
					Origin: testWarehouse2Origin,
					Sources: kargoapi.FreightSources{
						Stages:           []string{testStage},
						RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testWarehouse1,
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testWarehouse2,
					},
				},
				&kargoapi.Freight{ // Available because Freight is requested directly from this Warehouse
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: testWarehouse1Origin,
				},
				&kargoapi.Freight{ // Not available because Freight is not verified in upstream Stage
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: testWarehouse2Origin,
				},
				&kargoapi.Freight{ // Not available because verification time not recorded
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: testWarehouse2Origin,
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage: {},
						},
					},
				},
				&kargoapi.Freight{ // Not available because soak time has not elapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: testWarehouse2Origin,
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute}},
						},
					},
				},
				&kargoapi.Freight{ // Available because soak time has elapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-5",
					},
					Origin: testWarehouse2Origin,
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				require.Equal(t, "fake-freight-1", freight[0].Name)
				require.Equal(t, "fake-freight-5", freight[1].Name)
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := kargoapi.SchemeBuilder.AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme).
				WithScheme(testScheme).
				WithIndex(&kargoapi.Freight{}, warehouseField, warehouseIndexer).
				WithIndex(&kargoapi.Freight{}, approvedField, approvedForIndexer).
				WithIndex(&kargoapi.Freight{}, verifiedInField, verifiedInIndexer).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()

			stage := &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testStage,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: testCase.reqs,
				},
			}
			freight, err := ListFreightAvailableToStage(
				t.Context(), c, stage,
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestReverifyStageFreight(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := ReverifyStageFreight(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("missing current freight", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
			},
		).Build()

		err := ReverifyStageFreight(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage has no current freight")
	})

	t.Run("missing verification info", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
						},
					},
				},
			},
		).Build()

		err := ReverifyStageFreight(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage has no current verification info")
	})

	t.Run("missing verification info ID", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{}},
						},
					},
				},
			},
		).Build()

		err := ReverifyStageFreight(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage verification info has no ID")
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{
								ID: "fake-id",
							}},
						},
					},
				},
			},
		).Build()

		err := ReverifyStageFreight(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		require.Equal(t, (&kargoapi.VerificationRequest{
			ID: "fake-id",
		}).String(), stage.Annotations[kargoapi.AnnotationKeyReverify])
	})
}

func TestAbortStageFreightVerification(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("missing current freight", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
			},
		).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage has no current freight")
	})

	t.Run("missing verification info", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
						},
					},
				},
			},
		).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage has no current verification info")
	})

	t.Run("missing verification info ID", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{}},
						},
					},
				},
			},
		).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage verification info has no ID")
	})

	t.Run("verification in terminal phase", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{
								ID:    "fake-id",
								Phase: kargoapi.VerificationPhaseError,
							}},
						},
					},
				},
			},
		).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		_, ok := stage.Annotations[kargoapi.AnnotationKeyAbort]
		require.False(t, ok)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{
								ID: "fake-id",
							}},
						},
					},
				},
			},
		).Build()

		err := AbortStageFreightVerification(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		require.Equal(t, (&kargoapi.VerificationRequest{
			ID: "fake-id",
		}).String(), stage.Annotations[kargoapi.AnnotationKeyAbort])
	})
}

func TestAnnotateStageWithArgoCDContext(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := AnnotateStageWithArgoCDContext(t.Context(), c, []kargoapi.HealthCheckStep{}, &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
			},
		).Build()

		err := AnnotateStageWithArgoCDContext(t.Context(), c, []kargoapi.HealthCheckStep{
			{
				Uses: "argocd-update",
				Config: &apiextensionsv1.JSON{
					Raw: []byte(`{"apps": [{"name": "fake-argo-app", "namespace": "fake-argo-namespace"}]}`),
				},
			},
		}, &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
		})
		require.NoError(t, err)

		stage, err := GetStage(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		require.Equal(t,
			`[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]`,
			stage.Annotations[kargoapi.AnnotationKeyArgoCDContext])
	})

	t.Run("no ArgoCD apps", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyArgoCDContext: "fake-annotation",
					},
				},
			},
		).Build()

		err := AnnotateStageWithArgoCDContext(t.Context(), c, []kargoapi.HealthCheckStep{}, &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
		})
		require.NoError(t, err)

		stage, err := GetStage(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		require.NotContains(t, stage.Annotations, kargoapi.AnnotationKeyArgoCDContext)
	})
}

func TestStripStageForSummary(t *testing.T) {
	rawConfig := &apiextensionsv1.JSON{Raw: []byte(`{"to":"out"}`)}
	rawOutput := &apiextensionsv1.JSON{Raw: []byte(`{"status":"Healthy"}`)}

	testCases := []struct {
		name   string
		stage  *kargoapi.Stage
		assert func(*testing.T, *kargoapi.Stage)
	}{
		{
			name:   "nil Stage is a no-op",
			stage:  nil,
			assert: func(*testing.T, *kargoapi.Stage) {},
		},
		{
			name:  "empty Stage is left untouched",
			stage: &kargoapi.Stage{},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Empty(t, s.Status.FreightHistory)
				require.Nil(t, s.Spec.PromotionTemplate)
				require.Nil(t, s.Status.Health)
			},
		},
		{
			name: "FreightHistory with one entry is preserved",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f0"}}},
					},
				},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Len(t, s.Status.FreightHistory, 1)
				require.Equal(t, "f0", s.Status.FreightHistory[0].Freight["w"].Name)
			},
		},
		{
			name: "FreightHistory is truncated to the current entry",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f0"}}},
						{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f1"}}},
						{Freight: map[string]kargoapi.FreightReference{"w": {Name: "f2"}}},
					},
				},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Len(t, s.Status.FreightHistory, 1)
				require.Equal(t, "f0", s.Status.FreightHistory[0].Freight["w"].Name)
			},
		},
		{
			name: "PromotionTemplate step configs are cleared but skeletons kept",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{Uses: "copy", As: "step-0", Config: rawConfig},
								{Uses: "helm-update-image", As: "step-1", Config: rawConfig},
							},
						},
					},
				},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Len(t, s.Spec.PromotionTemplate.Spec.Steps, 2)
				require.Equal(t, "copy", s.Spec.PromotionTemplate.Spec.Steps[0].Uses)
				require.Equal(t, "step-0", s.Spec.PromotionTemplate.Spec.Steps[0].As)
				require.Nil(t, s.Spec.PromotionTemplate.Spec.Steps[0].Config)
				require.Equal(t, "helm-update-image", s.Spec.PromotionTemplate.Spec.Steps[1].Uses)
				require.Nil(t, s.Spec.PromotionTemplate.Spec.Steps[1].Config)
			},
		},
		{
			name: "nil PromotionTemplate is left untouched",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{PromotionTemplate: nil},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Nil(t, s.Spec.PromotionTemplate)
			},
		},
		{
			name: "Health.Output is cleared, other Health fields preserved",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
						Output: rawOutput,
					},
				},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.NotNil(t, s.Status.Health)
				require.Equal(t, kargoapi.HealthStateHealthy, s.Status.Health.Status)
				require.Nil(t, s.Status.Health.Output)
			},
		},
		{
			name: "nil Health is left untouched",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{Health: nil},
			},
			assert: func(t *testing.T, s *kargoapi.Stage) {
				require.Nil(t, s.Status.Health)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			StripStageForSummary(testCase.stage)
			testCase.assert(t, testCase.stage)
		})
	}
}

func TestListStageHealthOutputs(t *testing.T) {
	const testProject = "fake-namespace"
	const otherProject = "other-namespace"

	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	withHealth := func(namespace, name, rawJSON string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
			Status: kargoapi.StageStatus{
				Health: &kargoapi.Health{
					Output: &apiextensionsv1.JSON{Raw: []byte(rawJSON)},
				},
			},
		}
	}
	withoutHealth := func(namespace, name string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		}
	}

	testCases := []struct {
		name        string
		stageNames  []string
		objects     []client.Object
		interceptor interceptor.Funcs
		assert      func(*testing.T, map[string]string, error)
	}{
		{
			name:       "nil stageNames returns empty map",
			stageNames: nil,
			objects:    []client.Object{withHealth(testProject, "stage-1", `{"s":1}`)},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Empty(t, out)
				require.NotNil(t, out)
			},
		},
		{
			name:       "only empty strings returns empty map without listing",
			stageNames: []string{"", "", ""},
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					t.Fatal("List should not be called when there are no real names")
					return nil
				},
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Empty(t, out)
			},
		},
		{
			name:       "matching Stage returns its health output",
			stageNames: []string{"stage-1"},
			objects: []client.Object{
				withHealth(testProject, "stage-1", `{"s":1}`),
				withHealth(testProject, "stage-2", `{"s":2}`),
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, out, 1)
				require.JSONEq(t, `{"s":1}`, out["stage-1"])
			},
		},
		{
			name:       "duplicate entries are deduplicated",
			stageNames: []string{"stage-1", "stage-1", ""},
			objects: []client.Object{
				withHealth(testProject, "stage-1", `{"s":1}`),
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, out, 1)
				require.JSONEq(t, `{"s":1}`, out["stage-1"])
			},
		},
		{
			name:       "Stage without health output is omitted",
			stageNames: []string{"stage-1", "stage-no-health"},
			objects: []client.Object{
				withHealth(testProject, "stage-1", `{"s":1}`),
				withoutHealth(testProject, "stage-no-health"),
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, out, 1)
				require.Contains(t, out, "stage-1")
				require.NotContains(t, out, "stage-no-health")
			},
		},
		{
			name:       "unknown name is silently omitted",
			stageNames: []string{"stage-1", "does-not-exist"},
			objects:    []client.Object{withHealth(testProject, "stage-1", `{"s":1}`)},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, out, 1)
				require.Contains(t, out, "stage-1")
			},
		},
		{
			name:       "Stages in other namespaces are not returned",
			stageNames: []string{"stage-1"},
			objects: []client.Object{
				withHealth(testProject, "stage-1", `{"s":"ours"}`),
				withHealth(otherProject, "stage-1", `{"s":"theirs"}`),
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, out, 1)
				require.JSONEq(t, `{"s":"ours"}`, out["stage-1"])
			},
		},
		{
			name:       "List error propagates",
			stageNames: []string{"stage-1"},
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("boom")
				},
			},
			assert: func(t *testing.T, out map[string]string, err error) {
				require.ErrorContains(t, err, "boom")
				require.Nil(t, out)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			out, err := ListStageHealthOutputs(
				t.Context(), c, testProject, testCase.stageNames,
			)
			testCase.assert(t, out, err)
		})
	}
}
