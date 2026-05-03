package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps the PostgreSQL connection pool.
type Client struct {
	pool *pgxpool.Pool
}

// Connect creates and verifies a PostgreSQL connection pool.
func Connect(ctx context.Context, databaseURL string) (*Client, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	client := &Client{pool: pool}
	if err := client.Check(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return client, nil
}

// Check verifies the database is reachable.
func (c *Client) Check(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// Close releases the database pool.
func (c *Client) Close() {
	if c != nil && c.pool != nil {
		c.pool.Close()
	}
}
