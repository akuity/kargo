package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuity/kargo/internal/kubeclient"
)

func TestUnaryServerAuth(t *testing.T) {
	ctx := context.Background()
	testSets := map[string]struct {
		ctx                     context.Context
		expectedAuthHeaderValue string
	}{
		"request with credential": {
			ctx:                     kubeclient.SetCredentialToContext(ctx, "some-token"),
			expectedAuthHeaderValue: "Bearer some-token",
		},
		"request with empty credential": {
			ctx:                     kubeclient.SetCredentialToContext(ctx, ""),
			expectedAuthHeaderValue: "",
		},
		"request without credential": {
			ctx:                     ctx,
			expectedAuthHeaderValue: "",
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opt := connect.WithInterceptors(newAuthInterceptor())
			mux := http.NewServeMux()
			handler := connect.NewUnaryHandler(
				"/grpc.health.v1.Health/Check",
				func(
					ctx context.Context,
					req *connect.Request[grpc_health_v1.HealthCheckRequest],
				) (*connect.Response[grpc_health_v1.HealthCheckResponse], error) {
					res := connect.NewResponse(&grpc_health_v1.HealthCheckResponse{})
					cred, ok := kubeclient.GetCredentialFromContext(ctx)
					if ok {
						setAuthHeader(res.Header(), cred)
					}
					return res, nil
				},
				opt,
			)
			mux.Handle("/grpc.health.v1.Health/Check", handler)
			srv := httptest.NewServer(mux)
			srv.EnableHTTP2 = true
			t.Cleanup(srv.Close)

			client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
				srv.Client(),
				srv.URL+"/grpc.health.v1.Health/Check",
				connect.WithGRPC(),
				opt,
			)
			res, err := client.CallUnary(ts.ctx,
				connect.NewRequest[grpc_health_v1.HealthCheckRequest](&grpc_health_v1.HealthCheckRequest{}))
			require.NoError(t, err)

			require.Equal(
				t,
				ts.expectedAuthHeaderValue,
				res.Header().Get(authHeaderKey),
			)
		})
	}
}
