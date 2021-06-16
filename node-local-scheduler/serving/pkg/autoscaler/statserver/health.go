package statserver

import (
	"context"

	health "knative.dev/serving/pkg/autoscaler/statserver/grpc_health"
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
