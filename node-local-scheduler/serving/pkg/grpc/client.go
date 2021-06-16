package client

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/operator-framework/operator-registry/pkg/api"
	"github.com/operator-framework/operator-registry/pkg/api/grpc_health_v1"
)

type Interface interface {
    HandlerStatMsg(ctx context.Context, in *WireStatMessages, opts ...grpc.CallOption) (*empty.Empty, error)
	HealthCheck(ctx context.Context, reconnectTimeout time.Duration) (bool, error)
	Close() error
}

type Client struct {
	StatMsg  grpc_api.StatMsgClient
	Health   grpc_health_v1.HealthClient
	Conn     *grpc.ClientConn
}

var _ Interface = &Client{}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return NewClientFromConn(conn), nil
}

func NewClientFromConn(conn *grpc.ClientConn) *Client {
	return &Client{
		StatMsg:  grpc_api.NewStatMsgClient(conn),
		Health:   grpc_health.NewHealthClient(conn),
		Conn:     conn,
	}
}

func (c *Client) Close() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *Client) HandlerStatMsg(ctx context.Context, in *WireStatMessages, opts ...grpc.CallOption) (*empty.Empty, error) {
	res, err := c.StatMsg.HandlerStatMsg(ctx, &in)
	if err != nil {
		if c.Conn.GetState() == connectivity.TransientFailure {
			ctx, cancel := context.WithTimeout(ctx, reconnectTimeout)
			defer cancel()
			if !c.Conn.WaitForStateChange(ctx, connectivity.TransientFailure) {
				return false, NewHealthError(c.Conn, HealthErrReasonUnrecoveredTransient, "connection didn't recover from TransientFailure")
			}
		}
		return false, NewHealthError(c.Conn, HealthErrReasonConnection, err.Error())
	}
	if res.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return false, nil
	}
	return true, nil
}

func (c *Client) HealthCheck(ctx context.Context, reconnectTimeout time.Duration) (bool, error) {
	res, err := c.Health.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "Registry"})
	if err != nil {
		if c.Conn.GetState() == connectivity.TransientFailure {
			ctx, cancel := context.WithTimeout(ctx, reconnectTimeout)
			defer cancel()
			if !c.Conn.WaitForStateChange(ctx, connectivity.TransientFailure) {
				return false, NewHealthError(c.Conn, HealthErrReasonUnrecoveredTransient, "connection didn't recover from TransientFailure")
			}
		}
		return false, NewHealthError(c.Conn, HealthErrReasonConnection, err.Error())
	}
	if res.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return false, nil
	}
	return true, nil
}

