package client

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestInterceptorUnaryServerAuth(t *testing.T) {
	testSets := map[string]struct {
		credential              string
		expectedAuthHeaderValue string
	}{
		"without credential": {
			credential:              "",
			expectedAuthHeaderValue: "",
		},
		"with credential": {
			credential:              "some-token",
			expectedAuthHeaderValue: "Bearer some-token",
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(
				connect.NewUnaryHandler(
					"/",
					func(
						_ context.Context,
						req *connect.Request[grpc_health_v1.HealthCheckRequest],
					) (*connect.Response[grpc_health_v1.HealthCheckResponse], error) {
						// This is where we actually test that the interceptor did its job.
						assert.Equal(t, ts.expectedAuthHeaderValue, req.Header().Get(authHeaderKey))
						return connect.NewResponse(&grpc_health_v1.HealthCheckResponse{}),
							nil
					},
				),
			)
			t.Cleanup(srv.Close)

			client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
				srv.Client(),
				srv.URL,
				connect.WithInterceptors(
					&authInterceptor{
						credential: ts.credential,
					},
				),
			)
			_, err := client.CallUnary(
				context.Background(),
				connect.NewRequest[grpc_health_v1.HealthCheckRequest](&grpc_health_v1.HealthCheckRequest{}),
			)
			require.NoError(t, err)
		})
	}
}
