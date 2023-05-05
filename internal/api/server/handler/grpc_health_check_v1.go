package handler

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCHealthCheckV1Func func(
	context.Context,
	*grpc_health_v1.HealthCheckRequest,
) (*grpc_health_v1.HealthCheckResponse, error)

func NewGRPCHealthCheckV1() GRPCHealthCheckV1Func {
	return func(
		context.Context,
		*grpc_health_v1.HealthCheckRequest,
	) (*grpc_health_v1.HealthCheckResponse, error) {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}
}
