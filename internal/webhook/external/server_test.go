package external

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

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

func TestServer_Healthz(t *testing.T) {
	testClient := fake.NewFakeClient()
	s, ok := NewServer(ServerConfig{}, testClient).(*server)
	require.True(t, ok)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// We don't care about the error here because we're going to cancel the
		// context, which will cause the server to shut down gracefully.
		_ = s.Serve(ctx, l)
	}()

	// Make a request to the healthz endpoint
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://%s/healthz", l.Addr().String()),
		nil,
	)
	require.NoError(t, err)

	// We'll retry a few times in case the server isn't quite ready yet.
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
