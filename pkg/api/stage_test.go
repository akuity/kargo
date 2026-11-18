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
	"github.com/akuity/kargo/pkg/api/freightapi"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient/testutil"
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

	freight := func(origin kargoapi.FreightOrigin, name string) kargoapi.Freight {
		return kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      name,
			},
			Origin: origin,
		}
	}

	testCases := []struct {
		name       string
		reqs       []kargoapi.FreightRequest
		lister     *fakeFreightWarehouseLister
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name:   "no Freight from Warehouse",
			reqs:   []kargoapi.FreightRequest{{Origin: testWarehouse1Origin}},
			lister: &fakeFreightWarehouseLister{},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Empty(t, freight)
			},
		},
		{
			name:   "error listing Freight",
			reqs:   []kargoapi.FreightRequest{{Origin: testWarehouse1Origin}},
			lister: &fakeFreightWarehouseLister{err: errors.New("something went wrong")},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
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
			lister: &fakeFreightWarehouseLister{
				freightByWarehouse: map[string][]kargoapi.Freight{
					testWarehouse1: {
						freight(testWarehouse1Origin, "fake-freight-3"),
						freight(testWarehouse1Origin, "fake-freight-1"),
					},
					testWarehouse2: {
						freight(testWarehouse2Origin, "fake-freight-2"),
						// Duplicate of Freight returned for the other Warehouse
						freight(testWarehouse2Origin, "fake-freight-1"),
					},
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				// Freight from all Warehouses is combined, sorted by name, and
				// de-duped
				require.Len(t, freight, 3)
				require.Equal(t, "fake-freight-1", freight[0].Name)
				require.Equal(t, "fake-freight-2", freight[1].Name)
				require.Equal(t, "fake-freight-3", freight[2].Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
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
				t.Context(),
				testCase.lister,
				stage,
			)
			testCase.assertions(t, freight, err)
		})
	}
}

// fakeFreightWarehouseLister is a test-only FreightWarehouseLister that returns
// the Freight fixtures configured for each Warehouse as-is, without applying
// any filtering.
type fakeFreightWarehouseLister struct {
	freightByWarehouse map[string][]kargoapi.Freight
	err                error
}

func (f *fakeFreightWarehouseLister) ListFreightFromWarehouse(
	_ context.Context,
	_ string,
	warehouse string,
	_ *freightapi.ListWarehouseFreightOptions,
) ([]kargoapi.Freight, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.freightByWarehouse[warehouse], nil
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

		err := AnnotateStageWithArgoCDContext(
			t.Context(),
			c,
			&kargoapi.Promotion{},
			types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-stage",
			},
		)
		require.ErrorContains(t, err, "not found")
	})

	testCases := []struct {
		name     string
		promo    *kargoapi.Promotion
		expected string
	}{
		{
			// Argo CD-aware steps did not always report the Applications they
			// resolved as step output. Health check criteria remain the only
			// source of Argo CD context for such Promotions.
			name: "health check criteria only",
			promo: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					HealthChecks: []kargoapi.HealthCheckStep{{
						Uses: "argocd-update",
						Config: &apiextensionsv1.JSON{
							Raw: []byte(`{"apps":[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]}`),
						},
					}},
				},
			},
			expected: `[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]`,
		},
		{
			name: "step output only",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "argocd-wait", As: "wait"}},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{
						Raw: []byte(
							`{"wait":{"apps":[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]}}`,
						),
					},
				},
			},
			expected: `[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]`,
		},
		{
			name: "step output from a task step",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "argocd-wait", As: "task-1::wait"}},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{
						Raw: []byte(
							`{"task-1::wait":{"apps":[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]}}`,
						),
					},
				},
			},
			expected: `[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]`,
		},
		{
			// An argocd-update step reports the same Applications through both
			// its output and the health check criteria it registers, and a
			// subsequent argocd-wait step commonly targets those same
			// Applications.
			name: "duplicate apps across sources are deduped",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{Uses: "argocd-update", As: "update"},
						{Uses: "argocd-wait", As: "wait"},
					},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{
						Raw: []byte(`{
							"update":{"apps":[
								{"name":"fake-argo-app","namespace":"fake-argo-namespace"},
								{"name":"other-argo-app","namespace":"fake-argo-namespace"}
							]},
							"wait":{"apps":[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]}
						}`),
					},
					HealthChecks: []kargoapi.HealthCheckStep{{
						Uses: "argocd-update",
						Config: &apiextensionsv1.JSON{
							Raw: []byte(`{"apps":[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]}`),
						},
					}},
				},
			},
			expected: `[{"name":"fake-argo-app","namespace":"fake-argo-namespace"},` +
				`{"name":"other-argo-app","namespace":"fake-argo-namespace"}]`,
		},
		{
			name: "output of non-Argo CD steps is ignored",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "compose-output", As: "compose"}},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{
						Raw: []byte(`{"compose":{"apps":[{"name":"not-an-argo-app"}]}}`),
					},
				},
			},
		},
		{
			name: "malformed output contributes nothing",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{Uses: "argocd-wait", As: "wait"},
						{Uses: "argocd-wait", As: "other-wait"},
					},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{
						Raw: []byte(`{"wait":{"apps":"not-a-list"},"other-wait":{"apps":[42,{}]}}`),
					},
				},
			},
		},
		{
			name: "unparsable state",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "argocd-wait", As: "wait"}},
				},
				Status: kargoapi.PromotionStatus{
					State: &apiextensionsv1.JSON{Raw: []byte(`{"wait":`)},
				},
			},
		},
		{
			name:  "no ArgoCD apps",
			promo: &kargoapi.Promotion{},
		},
		{
			name: "nil Promotion",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
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

			stageKey := types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-stage",
			}
			require.NoError(t, AnnotateStageWithArgoCDContext(
				t.Context(), c, testCase.promo, stageKey,
			))

			stage, err := GetStage(t.Context(), c, stageKey)
			require.NoError(t, err)
			if testCase.expected == "" {
				require.NotContains(t, stage.Annotations, kargoapi.AnnotationKeyArgoCDContext)
				return
			}
			require.Equal(
				t,
				testCase.expected,
				stage.Annotations[kargoapi.AnnotationKeyArgoCDContext],
			)
		})
	}
}

func TestListFreightAvailableToStageViaClient(t *testing.T) {
	const project = "fake-project"
	const stageName = "fake-stage"
	freight := func(warehouse, name string) kargoapi.Freight {
		return kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: name},
			Origin: kargoapi.FreightOrigin{
				Name: warehouse,
				Kind: kargoapi.FreightOriginKindWarehouse,
			},
		}
	}

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: stageName},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh1"},
					Sources: kargoapi.FreightSources{Direct: true},
				},
				{
					Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh2"},
					Sources: kargoapi.FreightSources{
						Stages:           []string{"up"},
						RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
					},
				},
			},
		},
	}

	t.Run("resolves each Warehouse via the client and combines, de-duped", func(t *testing.T) {
		listFreightFromWarehouseCalls := []listFreightFromWarehouseCall{}
		mockClient := &testutil.MockFreightClient{
			ListFreightFromWarehouseFun: func(_ context.Context, _, warehouse string,
				opts *freightapi.ListWarehouseFreightOptions,
			) ([]kargoapi.Freight, error) {
				listFreightFromWarehouseCalls = append(listFreightFromWarehouseCalls,
					listFreightFromWarehouseCall{project, warehouse, opts})
				if warehouse == "wh1" {
					return []kargoapi.Freight{
						freight("wh1", "f1"), freight("wh1", "shared"),
					}, nil
				}
				if warehouse == "wh2" {
					return []kargoapi.Freight{
						freight("wh2", "f2"), freight("wh2", "shared"),
					}, nil
				}
				return nil, nil
			},
		}
		got, err := ListFreightAvailableToStage(t.Context(), mockClient, stage)
		require.NoError(t, err)

		names := make([]string, len(got))
		for i, f := range got {
			names[i] = f.Name
		}
		require.Equal(t, []string{"f1", "f2", "shared"}, names)

		require.Len(t, listFreightFromWarehouseCalls, 2)
		// Direct input -> nil options.
		require.Equal(t, "wh1", listFreightFromWarehouseCalls[0].warehouse)
		require.Nil(t, listFreightFromWarehouseCalls[0].opts)
		// Non-direct input -> options mirror ListFreightAvailableToStage.
		require.Equal(t, "wh2", listFreightFromWarehouseCalls[1].warehouse)
		require.NotNil(t, listFreightFromWarehouseCalls[1].opts)
		require.Equal(t, stageName, listFreightFromWarehouseCalls[1].opts.ApprovedFor)
		require.Equal(t, []string{"up"}, listFreightFromWarehouseCalls[1].opts.VerifiedIn)
		require.Equal(t, time.Hour, listFreightFromWarehouseCalls[1].opts.RequiredSoakTime.Duration)
	})

	t.Run("sets CurrentlyIn for the MatchUpstream selection policy", func(t *testing.T) {
		matchUpstream := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: stageName},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh"},
					Sources: kargoapi.FreightSources{
						Stages: []string{"up"},
						AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
							SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
						},
					},
				}},
			},
		}

		listFreightFromWarehouseCalls := []listFreightFromWarehouseCall{}
		mockClient := &testutil.MockFreightClient{
			ListFreightFromWarehouseFun: func(_ context.Context, project, warehouse string,
				opts *freightapi.ListWarehouseFreightOptions,
			) ([]kargoapi.Freight, error) {
				listFreightFromWarehouseCalls = append(listFreightFromWarehouseCalls,
					listFreightFromWarehouseCall{project, warehouse, opts})
				return nil, nil
			},
		}

		_, err := ListFreightAvailableToStage(t.Context(), mockClient, matchUpstream)
		require.NoError(t, err)
		require.Len(t, listFreightFromWarehouseCalls, 1)
		require.Equal(t, "up", listFreightFromWarehouseCalls[0].opts.CurrentlyIn)
	})

	t.Run("propagates the client error", func(t *testing.T) {
		mockClient := &testutil.MockFreightClient{
			ListFreightFromWarehouseFun: func(_ context.Context, _, _ string,
				_ *freightapi.ListWarehouseFreightOptions,
			) ([]kargoapi.Freight, error) {
				return nil, errors.New("boom")
			},
		}
		_, err := ListFreightAvailableToStage(t.Context(), mockClient, stage)
		require.ErrorContains(t, err, "boom")
	})
}

type listFreightFromWarehouseCall struct {
	project   string
	warehouse string
	opts      *freightapi.ListWarehouseFreightOptions
}
