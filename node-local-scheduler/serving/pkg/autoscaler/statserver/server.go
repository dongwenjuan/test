/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package statserver

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"go.uber.org/zap"
	"knative.dev/serving/pkg/autoscaler/bucket"
	"knative.dev/serving/pkg/autoscaler/metrics"
)

const closeCodeServiceRestart = 1012 // See https://www.iana.org/assignments/websocket/websocket.xhtml

// isBucketHost is the function deciding whether a host of a request is
// of an Autoscaler bucket service. It is set to bucket.IsBucketHost
// in production while can be overridden for testing.
var isBucketHost = bucket.IsBucketHost

// Server receives autoscaler statistics over WebSocket and sends them to a channel.
type Server struct {
	statsCh     chan<- metrics.StatMessage
	isBktOwner  func(bktName string) bool
	logger      *zap.SugaredLogger
}

// New creates a Server which will receive autoscaler statistics and forward them to statsCh until Shutdown is called.
func New(statsCh chan<- metrics.StatMessage, logger *zap.SugaredLogger, isBktOwner func(bktName string) bool) *Server {
	return &Server{
		statsCh:     statsCh,
		isBktOwner:  isBktOwner,
		logger:      logger.Named("stats-websocket-server").With("address", statsServerAddr),
	}
}

func (s *Server) HandlerStatMsg(context.Context, r *metrics.WireStatMessages) (*empty.Empty, error) {

	if s.isBktOwner != nil && isBucketHost(r.Host) {
		bkt := strings.SplitN(r.Host, ".", 2)[0]
		// It won't affect connections via Autoscaler service (used by Activator) or IP address.
		if !s.isBktOwner(bkt) {
			s.logger.Warn("Closing websocket because not the owner of the bucket ", bkt)
			return
		}
	}

    for _, wsm := range r.Messages {
        if wsm.Stat == nil {
            // To allow for future protobuf schema changes.
            continue
        }

        sm := wsm.ToStatMessage()
        s.logger.Debugf("Received stat message: %+v", sm)
        s.statsCh <- sm
    }

	return nil, status.Errorf(codes.Unimplemented, "method HandlerStatMsg not implemented")
}
