package grpcclient

import (
	"errors"

	"google.golang.org/grpc"

    asmetrics "knative.dev/serving/pkg/autoscaler/metrics"
	health "knative.dev/serving/pkg/grpc/api/grpc_health_v1"
)

type Interface interface {
	Close() error
}

type Client struct {
	StatMsg  asmetrics.StatMsgClient
	Health   health.HealthClient
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
