package client

import (
	"context"
	"errors"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

    asmetrics "knative.dev/serving/pkg/autoscaler/metrics"
	health "knative.dev/serving/pkg/grpc/api/grpc_health"
)

type Interface interface {
    HandlerStatMsg(ctx context.Context, in *asmetrics.WireStatMessages, opts ...grpc.CallOption) (*empty.Empty, error)
	HealthCheck(ctx context.Context, reconnectTimeout time.Duration) (bool, error)
	Close() error
}

type Client struct {
	StatMsg  asmetrics.StatMsgClient
	Health   health.HealthClient
	Conn     *grpc.ClientConn
}

var _ Interface = &Client{}
var EmptyRep = empty.Empty{}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return NewClientFromConn(conn), nil
}

func NewClientFromConn(conn *grpc.ClientConn) *Client {
	return &Client{
		StatMsg:  asmetrics.NewStatMsgClient(conn),
		Health:   health.NewHealthClient(conn),
		Conn:     conn,
	}
}

func (c *Client) Close() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *Client) HandlerStatMsg(ctx context.Context, in *asmetrics.WireStatMessages, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := c.StatMsg.HandlerStatMsg(ctx, in)
	if err != nil {
		if c.Conn.GetState() == connectivity.TransientFailure {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			if !c.Conn.WaitForStateChange(ctx, connectivity.TransientFailure) {
				return &EmptyRep, errors.New("connection didn't recover from TransientFailure")
			}
		}
		return &EmptyRep, err
	}

	return &EmptyRep, nil
}

func (c *Client) HealthCheck(ctx context.Context, reconnectTimeout time.Duration) (bool, error) {
	res, err := c.Health.Check(ctx, &health.HealthCheckRequest{Service: "Registry"})
	if err != nil {
		if c.Conn.GetState() == connectivity.TransientFailure {
			ctx, cancel := context.WithTimeout(ctx, reconnectTimeout)
			defer cancel()
			if !c.Conn.WaitForStateChange(ctx, connectivity.TransientFailure) {
				return false, errors.New("connection didn't recover from TransientFailure")
			}
		}
		return false, err
	}
	if res.Status != health.HealthCheckResponse_SERVING {
		return false, nil
	}
	return true, nil
}

