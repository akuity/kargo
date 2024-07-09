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
