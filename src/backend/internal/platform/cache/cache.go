package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Options configures the Valkey client.
type Options struct {
	Addr     string
	Password string
	DB       int
}

// Client wraps the Valkey connection.
type Client struct {
	client *redis.Client
}

// Connect creates and verifies a Valkey connection.
func Connect(ctx context.Context, options Options) (*Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password,
		DB:       options.DB,
	})

	client := &Client{client: redisClient}
	if err := client.Check(ctx); err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	return client, nil
}

// Check verifies Valkey is reachable.
func (c *Client) Check(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping valkey: %w", err)
	}
	return nil
}

// Close releases the Valkey client.
func (c *Client) Close() {
	if c != nil && c.client != nil {
		_ = c.client.Close()
	}
}
