package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetProject(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Project, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, project *Project, err error) {
				require.NoError(t, err)
				require.Nil(t, project)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
				},
			).Build(),
			assertions: func(t *testing.T, project *Project, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-project", project.Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			project, err := GetProject(
				context.Background(),
				testCase.client,
				"fake-project",
			)
			testCase.assertions(t, project, err)
		})
	}
}
