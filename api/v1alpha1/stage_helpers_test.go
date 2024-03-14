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

func TestGetStage(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*Stage, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(stage *Stage, err error) {
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
			assertions: func(stage *Stage, err error) {
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
			testCase.assertions(stage, err)
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
					CurrentFreight: &FreightReference{},
				},
			},
		).Build()

		err := ReverifyStageFreight(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.ErrorContains(t, err, "stage has no existing verification info")
	})

	t.Run("missing verification info ID", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Status: StageStatus{
					CurrentFreight: &FreightReference{
						VerificationInfo: &VerificationInfo{},
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
					CurrentFreight: &FreightReference{
						VerificationInfo: &VerificationInfo{
							ID: "fake-id",
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
		require.Equal(t, "fake-id", stage.Annotations[AnnotationKeyReverify])
	})
}

func TestClearStageReverify(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	t.Run("no annotations", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
			},
		).Build()

		err := ClearStageReverify(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
		})
		require.NoError(t, err)
	})

	t.Run("no reconfirm annotation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "fake-stage",
					Namespace:   "fake-namespace",
					Annotations: map[string]string{},
				},
			},
		).Build()

		err := ClearStageReverify(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "fake-stage",
				Namespace:   "fake-namespace",
				Annotations: map[string]string{},
			},
		})
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						AnnotationKeyReverify: "fake-id",
					},
				},
			},
		).Build()

		err := ClearStageReverify(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
				Annotations: map[string]string{
					AnnotationKeyReverify: "fake-id",
				},
			},
		})
		require.NoError(t, err)

		stage, err := GetStage(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-stage",
		})
		require.NoError(t, err)
		_, ok := stage.Annotations[AnnotationKeyReverify]
		require.False(t, ok)
	})
}

func TestClearStageAbort(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	t.Run("no annotations", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
			},
		).Build()

		err := ClearStageAbort(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
		})
		require.NoError(t, err)
	})

	t.Run("no abort annotation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "fake-stage",
					Namespace:   "fake-namespace",
					Annotations: map[string]string{},
				},
			},
		).Build()

		err := ClearStageAbort(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "fake-stage",
				Namespace:   "fake-namespace",
				Annotations: map[string]string{},
			},
		})
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						AnnotationKeyAbort: "fake-id",
					},
				},
			},
		).Build()

		err := ClearStageAbort(context.TODO(), c, &Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
				Annotations: map[string]string{
					AnnotationKeyAbort: "fake-id",
				},
			},
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
}
