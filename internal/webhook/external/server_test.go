package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/server/kubernetes"
)

func TestNewServer(t *testing.T) {
	testServerConfig := ServerConfig{}
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

	s, ok := NewServer(testServerConfig, testClient).(*server)
	require.True(t, ok)
	require.NotNil(t, s)
}
