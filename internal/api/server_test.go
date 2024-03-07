package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
)

func TestNewServer(t *testing.T) {
	testServerConfig := config.ServerConfig{}
	testClient, err := kubernetes.NewClient(
		context.Background(),
		&rest.Config{},
		kubernetes.ClientOptions{
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
			) (client.Client, error) {
				return fake.NewClientBuilder().Build(), nil
			},
		},
	)
	require.NoError(t, err)
	s, ok := NewServer(testServerConfig, testClient, fake.NewClientBuilder().Build()).(*server)
	require.True(t, ok)
	require.NotNil(t, s)
	require.Same(t, testClient, s.client)
	require.Equal(t, testServerConfig, s.cfg)
	require.NotNil(t, s.validateProjectExistsFn)
	require.NotNil(t, s.externalValidateProjectFn)
	require.NotNil(t, s.getStageFn)
	require.NotNil(t, s.getFreightByNameOrAliasFn)
	require.NotNil(t, s.isFreightAvailableFn)
	require.NotNil(t, s.createPromotionFn)
	require.NotNil(t, s.findStageSubscribersFn)
	require.NotNil(t, s.listFreightFn)
	require.NotNil(t, s.getAvailableFreightForStageFn)
	require.NotNil(t, s.getFreightFromWarehouseFn)
	require.NotNil(t, s.getVerifiedFreightFn)
	require.NotNil(t, s.patchFreightAliasFn)
	require.NotNil(t, s.patchFreightStatusFn)
	require.NotNil(t, s.authorizeFn)
}
