package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetProjectConfig(t *testing.T) {
	const testProjectName = "fake-project"

	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.ProjectConfig, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, projectCfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, projectCfg)
			},
		},
		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testProjectName,
						Namespace: testProjectName,
					},
				},
			).Build(),
			assertions: func(t *testing.T, projectCfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, testProjectName, projectCfg.Name)
				require.Equal(t, testProjectName, projectCfg.Namespace)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			projectCfg, err := GetProjectConfig(
				context.Background(),
				testCase.client,
				testProjectName,
			)
			testCase.assertions(t, projectCfg, err)
		})
	}
}
