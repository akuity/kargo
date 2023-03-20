package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

func TestGRPCHealthCheckV1(t *testing.T) {
	t.Parallel()
	testSets := map[string]struct {
		in       *grpc_health_v1.HealthCheckRequest
		expected *grpc_health_v1.HealthCheckResponse
	}{
		"health check request": {
			in: &grpc_health_v1.HealthCheckRequest{},
			expected: &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_SERVING,
			},
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			actual, err := NewGRPCHealthCheckV1()(context.Background(), ts.in)
			require.NoError(t, err)
			require.True(t, proto.Equal(ts.expected, actual))
		})
	}
}
