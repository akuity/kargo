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

func TestGetClusterConfig(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.ClusterConfig, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, clusterCfg *kargoapi.ClusterConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, clusterCfg)
			},
		},
		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.ClusterConfig{
					ObjectMeta: metav1.ObjectMeta{Name: ClusterConfigName},
				},
			).Build(),
			assertions: func(t *testing.T, clusterCfg *kargoapi.ClusterConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, ClusterConfigName, clusterCfg.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clusterCfg, err := GetClusterConfig(context.Background(), testCase.client)
			testCase.assertions(t, clusterCfg, err)
		})
	}
}
