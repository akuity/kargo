package external

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewServer(t *testing.T) {
	testCfg := ServerConfig{}
	testClient := fake.NewFakeClient()
	s, ok := NewServer(ServerConfig{}, testClient).(*server)
	require.True(t, ok)
	require.Equal(t, testCfg, s.cfg)
	require.Same(t, testClient, s.client)
}