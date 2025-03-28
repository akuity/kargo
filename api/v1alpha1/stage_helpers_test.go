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

func TestVerificationRequest_Equals(t *testing.T) {
	tests := []struct {
		name     string
		r1       *VerificationRequest
		r2       *VerificationRequest
		expected bool
	}{
		{
			name:     "both nil",
			r1:       nil,
			r2:       nil,
			expected: true,
		},
		{
			name:     "one nil",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			r2:       nil,
			expected: false,
		},
		{
			name:     "other nil",
			r1:       nil,
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different IDs",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			r2:       &VerificationRequest{ID: "other-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actors",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "other-actor", ControlPlane: true},
			expected: false,
		},
		{
			name:     "different control plane flags",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "equal",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.r1.Equals(tt.r2), tt.expected)
		})
	}
}

func TestVerificationRequest_HasID(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.False(t, r.HasID())
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.False(t, r.HasID())
	})

	t.Run("verification request has ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "foo",
		}
		require.True(t, r.HasID())
	})
}

func TestVerificationRequest_ForID(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.False(t, r.ForID("foo"))
	})

	t.Run("verification request has ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "foo",
		}
		require.True(t, r.ForID("foo"))
		require.False(t, r.ForID("bar"))
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.False(t, r.ForID(""))
		require.False(t, r.ForID("foo"))
	})
}

func TestVerificationRequest_String(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.Empty(t, r.String())
	})

	t.Run("verification request is empty", func(t *testing.T) {
		r := &VerificationRequest{}
		require.Empty(t, r.String())
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.Empty(t, r.String())
	})

	t.Run("verification request has data", func(t *testing.T) {
		r := &VerificationRequest{
			ID:           "foo",
			Actor:        "fake-actor",
			ControlPlane: true,
		}
		require.Equal(t, `{"id":"foo","actor":"fake-actor","controlPlane":true}`, r.String())
	})
}

func TestGetStage(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Stage, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, stage *Stage, err error) {
				require.NoError(t, err)
				require.Nil(t, stage)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, stage *Stage, err error) {
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

func TestStage_ListAvailableFreight(t *testing.T) {
	const testProject = "fake-namespace"
	const testWarehouse1 = "fake-warehouse1"
	const testWarehouse2 = "fake-warehouse2"
	const testStage = "fake-stage"

	testWarehouse1Origin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: testWarehouse1,
	}

	testWarehouse2Origin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: testWarehouse2,
	}

	testCases := []struct {
		name        string
		reqs        []FreightRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []Freight, error)
	}{
		{
			name: "error getting Warehouse",
			reqs: []FreightRequest{{}},
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
			assertions: func(t *testing.T, _ []Freight, err error) {
				require.ErrorContains(t, err, "error getting Warehouse")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "Warehouse not found",
			reqs: []FreightRequest{{}},
			assertions: func(t *testing.T, _ []Freight, err error) {
				require.ErrorContains(t, err, "Warehouse")
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "error listing Freight",
			reqs: []FreightRequest{{Origin: testWarehouse1Origin}},
			objects: []client.Object{
				&Warehouse{
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
			assertions: func(t *testing.T, _ []Freight, err error) {
				require.ErrorContains(t, err, "error listing Freight for Warehouse")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reqs: []FreightRequest{
				{
					Origin:  testWarehouse1Origin,
					Sources: FreightSources{Direct: true},
				},
				{
					Origin: testWarehouse2Origin,
					Sources: FreightSources{
						Stages:           []string{testStage},
						RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
					},
				},
			},
			objects: []client.Object{
				&Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testWarehouse1,
					},
				},
				&Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testWarehouse2,
					},
				},
				&Freight{ // Available because Freight is requested directly from this Warehouse
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Origin: testWarehouse1Origin,
				},
				&Freight{ // Not available because Freight is not verified in upstream Stage
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Origin: testWarehouse2Origin,
				},
				&Freight{ // Not available because verification time not recorded
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Origin: testWarehouse2Origin,
					Status: FreightStatus{
						VerifiedIn: map[string]VerifiedStage{
							testStage: {},
						},
					},
				},
				&Freight{ // Not available because soak time has not elapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Origin: testWarehouse2Origin,
					Status: FreightStatus{
						VerifiedIn: map[string]VerifiedStage{
							testStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute},
							},
						},
					},
				},
				&Freight{ // Available because soak time has elapsed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-5",
					},
					Origin: testWarehouse2Origin,
					Status: FreightStatus{
						VerifiedIn: map[string]VerifiedStage{
							testStage: {
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, freight []Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				require.Equal(t, "fake-freight-1", freight[0].Name)
				require.Equal(t, "fake-freight-5", freight[1].Name)
			},
		},
	}

	testScheme := k8sruntime.NewScheme()
	err := SchemeBuilder.AddToScheme(testScheme)
	require.NoError(t, err)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme).
				WithScheme(testScheme).
				WithIndex(&Freight{}, warehouseField, warehouseIndexer).
				WithIndex(&Freight{}, approvedField, approvedForIndexer).
				WithIndex(&Freight{}, verifiedInField, verifiedInIndexer).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()

			stage := &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testStage,
				},
				Spec: StageSpec{
					RequestedFreight: testCase.reqs,
				},
			}
			freight, err := stage.ListAvailableFreight(context.Background(), c)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestStage_IsFreightAvailable(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testWarehouse = "fake-warehouse"
	const testStage = "fake-stage"
	const testFreight = "fake-freight"
	testStageMeta := metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testStage,
	}
	testFreightMeta := metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testFreight,
	}
	testOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: testWarehouse,
	}

	testCases := []struct {
		name     string
		stage    *Stage
		freight  *Freight
		expected bool
	}{
		{
			name:     "stage is nil",
			freight:  &Freight{ObjectMeta: testFreightMeta},
			expected: false,
		},
		{
			name:     "freight is nil",
			stage:    &Stage{ObjectMeta: testStageMeta},
			expected: false,
		},
		{
			name:  "stage and freight are in different namespaces",
			stage: &Stage{ObjectMeta: testStageMeta},
			freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: "wrong-namespace"},
			},
			expected: false,
		},
		{
			name:  "freight is approved for stage",
			stage: &Stage{ObjectMeta: testStageMeta},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
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
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Direct: true,
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream; soak not required",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream stage with no longestCompletedSoak; soak required",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: false,
		},
		{
			name: "freight is verified in an upstream stage with longestCompletedSoak; soak required but not elapsed",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: 2 * time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {
							LongestCompletedSoak: &metav1.Duration{Duration: time.Hour},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "freight is verified in an upstream stage with longestCompletedSoak; soak required and is elapsed",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {
							LongestCompletedSoak: &metav1.Duration{Duration: time.Hour},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight from origin not requested",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
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
				testCase.stage.IsFreightAvailable(testCase.freight),
			)
		})
	}
}

func TestReverifyStageFreight(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

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
			&Stage{
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []VerificationInfo{{}},
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []VerificationInfo{{
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
		require.Equal(t, (&VerificationRequest{
			ID: "fake-id",
		}).String(), stage.Annotations[AnnotationKeyReverify])
	})
}

func TestAbortStageFreightVerification(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

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
			&Stage{
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []VerificationInfo{{}},
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
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []VerificationInfo{{
								ID:    "fake-id",
								Phase: VerificationPhaseError,
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
		_, ok := stage.Annotations[AnnotationKeyAbort]
		require.False(t, ok)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					FreightHistory: FreightHistory{
						{
							Freight: map[string]FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: []VerificationInfo{{
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
		require.Equal(t, (&VerificationRequest{
			ID: "fake-id",
		}).String(), stage.Annotations[AnnotationKeyAbort])
	})
}
