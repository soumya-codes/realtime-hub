package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/message"
	"go.uber.org/zap"
)

// PubSub manages the Redis pub/sub operations.
type PubSub struct {
	client      *Client
	pubSub      *redis.PubSub
	channel     string
	hubID       string
	broadcastCh chan<- message.MessageDetails
	logger      *zap.Logger
}

// NewPubSub creates a new PubSub instance.
func NewPubSub(client *Client, channel, hubID string, broadcastCh chan<- message.MessageDetails, logger *zap.Logger) *PubSub {
	return &PubSub{
		client:      client,
		channel:     channel,
		hubID:       hubID,
		broadcastCh: broadcastCh,
		logger:      logger,
	}
}

func (ps *PubSub) Subscribe(ctx context.Context) {
	ps.pubSub = ps.client.Subscribe(ctx, ps.channel)
	for msg := range ps.pubSub.Channel() {
		var md message.MessageDetails
		if err := md.FromJSON([]byte(msg.Payload)); err != nil {
			ps.logger.Error("Failed to unmarshal message", zap.Error(err))
			continue
		}

		if md.HubID != ps.hubID {
			md.SenderID = ps.channel
			ps.broadcastCh <- md
		}
	}
}

// Unsubscribe unsubscribes from the Redis pub/sub channel.
func (ps *PubSub) Unsubscribe(ctx context.Context) error {
	if err := ps.pubSub.Unsubscribe(ctx, ps.channel); err != nil {
		ps.logger.Error("Failed to unsubscribe from Redis channel", zap.String("channel", ps.channel), zap.Error(err))
		return fmt.Errorf("failed to unsubscribe from Redis channel: %s, error: %w", ps.channel, err)
	}

	ps.logger.Info("Unsubscribed from Redis channel", zap.String("channel", ps.channel))
	return nil
}

// Publish publishes a message to the Redis pub/sub channel.
func (ps *PubSub) Publish(ctx context.Context, md *message.MessageDetails) error {
	data, err := md.ToJSON()
	if err != nil {
		ps.logger.Error("Failed to marshal message", zap.Error(err))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	result := ps.client.Publish(ctx, ps.channel, data)
	if err := result.Err(); err != nil {
		ps.logger.Error("Failed to publish message to Redis", zap.Error(err))
		return err
	}

	return nil
}

// Close closes the PubSub connection.
func (ps *PubSub) Close() error {
	if err := ps.pubSub.Close(); err != nil {
		ps.logger.Error("Failed to close Redis pubsub connection", zap.String("channel", ps.channel), zap.Error(err))
		return fmt.Errorf("failed to close Redis pubsub connection: %w", err)
	}

	ps.logger.Info("Redis pubsub connection closed successfully", zap.String("channel", ps.channel))
	return nil
}
