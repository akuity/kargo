package option

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuity/kargo/internal/kubeclient"
)

func TestUnaryServerAuth(t *testing.T) {
	testSets := map[string]struct {
		authHeaderValue    string
		expectedCredential string
	}{
		"without credential": {
			authHeaderValue:    "",
			expectedCredential: "",
		},
		"with credential": {
			authHeaderValue:    "Bearer some-token",
			expectedCredential: "some-token",
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(
				connect.NewUnaryHandler(
					"/",
					func(
						ctx context.Context,
						_ *connect.Request[grpc_health_v1.HealthCheckRequest],
					) (*connect.Response[grpc_health_v1.HealthCheckResponse], error) {
						// This is where we actually test that the interceptor did its job.
						cred, ok := kubeclient.GetCredentialFromContext(ctx)
						assert.Equal(t, cred != "", ok)
						assert.Equal(t, ts.expectedCredential, cred)
						return connect.NewResponse(&grpc_health_v1.HealthCheckResponse{}),
							nil
					},
					connect.WithInterceptors(&authInterceptor{}),
				),
			)
			t.Cleanup(srv.Close)

			client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
				srv.Client(),
				srv.URL,
				connect.WithInterceptors(
					&testClientAuthInterceptor{
						authHeaderValue: ts.authHeaderValue,
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

type testClientAuthInterceptor struct {
	authHeaderValue string
}

func (t *testClientAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", t.authHeaderValue)
		return next(ctx, req)
	}
}

func (t *testClientAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (t *testClientAuthInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	}
}
