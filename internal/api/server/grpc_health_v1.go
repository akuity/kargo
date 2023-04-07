package server

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuity/kargo/internal/api/server/handler"
)

type grpcHealthV1Server struct {
	grpc_health_v1.UnimplementedHealthServer
}

func newGRPCHealthV1Server() grpc_health_v1.HealthServer {
	return &grpcHealthV1Server{}
}

func (s *grpcHealthV1Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return handler.NewGRPCHealthCheckV1()(ctx, req)
}
