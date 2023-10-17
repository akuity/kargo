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
