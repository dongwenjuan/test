package grpcserver

import (
	"context"

	health "knative.dev/serving/pkg/grpc/api/grpc_health_v1"
)

type HealthServer struct {
	health.UnimplementedHealthServer
}

var _ health.HealthServer = &HealthServer{}

func NewHealthServer() *HealthServer {
	return &HealthServer{health.UnimplementedHealthServer{}}
}

func (s *HealthServer) Check(ctx context.Context, req *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}
