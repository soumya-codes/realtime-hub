package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Client wraps the Redis client and provides logging functionality.
type Client struct {
	*redis.Client
	logger *zap.Logger
}

// NewClient creates a new Redis client with the provided address and logger.
func NewClient(addr, username, password string, logger *zap.Logger) *Client {
	options := &redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
	}
	client := redis.NewClient(options)

	return &Client{
		Client: client,
		logger: logger,
	}
}

// Ping checks the connection to Redis.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Client.Ping(ctx).Result()
	if err != nil {
		c.logger.Error("Failed to ping Redis", zap.Error(err))
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	c.logger.Info("Successfully connected to Redis")
	return nil
}

// Close closes the Redis client.
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		c.logger.Error("Failed to close Redis client", zap.Error(err))
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	c.logger.Info("Redis client closed successfully")
	return nil
}
