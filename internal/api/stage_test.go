package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
				context.Background(),
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
								VerifiedAt: &metav1.Time{Time: time.Now()},
							},
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
								VerifiedAt: &metav1.Time{Time: time.Now().Add(-time.Hour * 2)},
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
				context.Background(), c, stage,
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

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
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

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
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

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
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

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
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

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(context.TODO(), c, types.NamespacedName{
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

		err := AbortStageFreightVerification(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)

		stage, err := GetStage(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		require.Equal(t, (&kargoapi.VerificationRequest{
			ID: "fake-id",
		}).String(), stage.Annotations[kargoapi.AnnotationKeyAbort])
	})
}

func TestInjectArgoCDContextToStage(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := InjectArgoCDContextToStage(context.TODO(), c, []kargoapi.HealthCheckStep{}, &kargoapi.Stage{
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

		err := InjectArgoCDContextToStage(context.TODO(), c, []kargoapi.HealthCheckStep{
			{
				Uses: "argocd-update",
				Config: &v1.JSON{
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

		stage, err := GetStage(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		require.Equal(t,
			`[{"name":"fake-argo-app","namespace":"fake-argo-namespace"}]`,
			stage.Annotations[kargoapi.AnnotationKeyArgoCDContext])
	})
}
