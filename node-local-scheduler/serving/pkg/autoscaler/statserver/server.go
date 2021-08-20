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

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

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
    metrics.UnimplementedStatMsgServer
	statsCh     chan<- metrics.StatMessage
	isBktOwner  func(bktName string) bool
	logger      *zap.SugaredLogger
}

// New creates a Server which will receive autoscaler statistics and forward them to statsCh until Shutdown is called.
func New(statsServerAddr string, statsCh chan<- metrics.StatMessage, logger *zap.SugaredLogger, isBktOwner func(bktName string) bool) Server {
	return Server{
		statsCh:     statsCh,
		isBktOwner:  isBktOwner,
		logger:      logger.Named("stats-server").With("address", statsServerAddr),
	}
}

func (s *Server) HandlerStatMsg(ctx context.Context, r *metrics.WireStatMessages) (*empty.Empty, error) {
	out := new(empty.Empty)

    headers, ok := metadata.FromIncomingContext(ctx)
    if !ok {
	    s.logger.Errorw("Failed to get metadata from ctx!")
		return out, nil
    }
    host := headers[":authority"][0]

	if s.isBktOwner != nil && isBucketHost(host) {
		bkt := strings.SplitN(host, ".", 2)[0]
		// It won't affect connections via Autoscaler service (used by Activator) or IP address.
		if !s.isBktOwner(bkt) {
			s.logger.Warn("Closing grpc because not the owner of the bucket ", bkt)
			return out, nil
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

	return out, nil
}
