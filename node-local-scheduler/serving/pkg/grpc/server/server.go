package server

import (
	"net"

	"google.golang.org/grpc"
	"go.uber.org/zap"

	asmetrics "knative.dev/serving/pkg/autoscaler/metrics"
	"knative.dev/serving/pkg/autoscaler/statserver"
)

type Server struct {
    addr          string
	server        *grpc.Server
	statServer    statserver.Server
	healthServer  HealthServer
	logger        *zap.SugaredLogger
}

// New creates a Server which will receive autoscaler statistics and forward them to statsCh until Shutdown is called.
func NewServer(addr string, statsCh chan<- asmetrics.StatMessage, logger *zap.SugaredLogger, isBktOwner func(bktName string) bool) Server {
	Server := Server{
		addr:          addr,
		server:        grpc.NewServer(),
		statServer:    statserver.New(addr, statsCh, logger, isBktOwner),
		healthServer:  NewHealthServer(),
		logger:        logger.Named("grpc-server").With("address", addr),
	}
    asmetrics.RegisterStatMsgServer(Server.server, &(Server.statServer))
    health.RegisterHealthServer(Server.server, &(Server.healthServer))

    return Server
}

func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Fatalw("net.Listen err", zap.Error(err))
		return err
	}

	return s.server.Serve(lis)
}

func (s *Server) Shutdown() {
    s.server.GracefulStop()
}