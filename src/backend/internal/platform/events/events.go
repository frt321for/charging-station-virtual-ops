package events

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Client wraps the NATS connection.
type Client struct {
	conn *nats.Conn
}

// Connect creates and verifies a NATS connection.
func Connect(url string) (*Client, error) {
	conn, err := nats.Connect(url, nats.Name("charging-ops-api"))
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	client := &Client{conn: conn}
	if err := client.Check(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

// Check verifies the NATS connection is usable.
func (c *Client) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("check nats: %w", ctx.Err())
	default:
	}

	if !c.conn.IsConnected() {
		return fmt.Errorf("nats disconnected")
	}
	return nil
}

// Close drains and closes the NATS connection.
func (c *Client) Close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Drain()
		c.conn.Close()
	}
}
