package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/soumya-codes/realtime-hub/hubserver/internal/message"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/redis"
	"go.uber.org/zap"
)

// MessageHandler manages all active WebSocket connections and message broadcasting.
type MessageHandler struct {
	connections      map[string]*Connection
	mu               sync.RWMutex
	broadcastCh      chan message.MessageDetails
	remove           chan string
	redisPubSub      *redis.PubSub
	pubSubChannel    string
	hubID            string
	broadcastWorkers int
	logger           *zap.Logger
}

func NewMessageHandler(redisClient *redis.Client, pubSubChannel, hubID string, broadcastWorkers int, logger *zap.Logger) (*MessageHandler, error) {
	broadcastCh := make(chan message.MessageDetails, 1024) // Increased buffer size to handle bursts

	handler := &MessageHandler{
		connections:      make(map[string]*Connection),
		broadcastCh:      broadcastCh,
		remove:           make(chan string, 256),
		redisPubSub:      redis.NewPubSub(redisClient, pubSubChannel, hubID, broadcastCh, logger),
		pubSubChannel:    pubSubChannel,
		hubID:            hubID,
		broadcastWorkers: broadcastWorkers,
		logger:           logger,
	}

	return handler, nil
}

// ServeHTTP handles HTTP requests and upgrades them to WebSocket connections.
func (h *MessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.createAndAddConnection(w, r)
	if err != nil {
		h.logger.Error("Failed to create and add connection", zap.Error(err))
		return
	}
	go h.handleIncomingMessages(conn)
}

// createAndAddConnection adds a new WebSocket connection to the map and starts handling its messages.
func (h *MessageHandler) createAndAddConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	conn, err := Upgrade(w, r, h)
	if err != nil {
		return nil, fmt.Errorf("error creating websocket connection: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.connections[conn.id]; exists {
		return nil, fmt.Errorf("connection already registered")
	}

	h.connections[conn.id] = conn
	return conn, nil
}

// handleIncomingMessages handles messages read from the connection's read channel.
func (h *MessageHandler) handleIncomingMessages(conn *Connection) {
	defer func() {
		h.remove <- conn.id
	}()

	for msg := range conn.readCh {
		md := message.NewMessageDetails(conn.id, h.hubID, conn.id, msg)
		h.broadcastCh <- md
	}

	h.logger.Error("Read channel closed for the connection", zap.String("conn-id", conn.id))
}

// broadcastWorker processes messages from the broadcast channel.
func (h *MessageHandler) broadcastWorker() {
	ctx := context.Background()

	for md := range h.broadcastCh {
		h.logger.Info("Received message from broadcastCh", zap.String("senderID", md.SenderID))
		h.broadcastToConnections(md)
		h.forwardToRedisIfNeeded(ctx, md)
	}
}

func (h *MessageHandler) broadcastToConnections(md message.MessageDetails) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, conn := range h.connections {
		if md.ShouldBroadcastToClient(id) {
			select {
			case conn.writeCh <- md:
			default:
				h.logger.Warn("Write channel is full, dropping message",
					zap.String("connID", id),
					zap.String("senderID", md.SenderID),
					zap.ByteString("message", md.Message))
			}
		}
	}
}

func (h *MessageHandler) forwardToRedisIfNeeded(ctx context.Context, md message.MessageDetails) {
	if !md.IsFromPubSub(h.pubSubChannel) {
		if err := h.redisPubSub.Publish(ctx, &md); err != nil {
			h.logger.Error("Failed to publish message to Redis", zap.Error(err))
		}
	}
}

// Run starts the message handler's main loop.
func (h *MessageHandler) Run() {
	ctx := context.Background()
	go h.redisPubSub.Subscribe(ctx)

	// Start multiple workers for broadcasting messages.
	for i := 0; i < h.broadcastWorkers; i++ {
		go h.broadcastWorker()
	}

	// Handle connection removals in a range loop
	for connID := range h.remove {
		h.closeAndRemoveConnection(connID)
	}
}

// closeAndRemoveConnection removes a WebSocket connection from the map.
func (h *MessageHandler) closeAndRemoveConnection(connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.connections[connID]; ok {
		delete(h.connections, connID)
		err := conn.Close()
		if err != nil {
			h.logger.Error("Error closing connection", zap.String("conn-id", connID), zap.Error(err))
			return
		}
		h.logger.Info("Connection closed successfully", zap.String("conn-id", connID))
	} else {
		h.logger.Info("Connection already closed", zap.String("conn-id", connID))
	}
}

// Close cleans up resources used by the message handler.
func (h *MessageHandler) Close() error {
	h.closeAndRemoveAllConnections()

	if err := h.redisPubSub.Unsubscribe(context.Background()); err != nil {
		h.logger.Error("Failed to unsubscribe from Redis pub-sub channel", zap.Error(err))
	}

	if err := h.redisPubSub.Close(); err != nil {
		h.logger.Error("Failed to close Redis pub-sub connection", zap.Error(err))
		return fmt.Errorf("failed to close Redis pub-sub connection: %w", err)
	}

	return nil
}

// closeAndRemoveAllConnections closes all the WebSocket connections.
func (h *MessageHandler) closeAndRemoveAllConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for connID, conn := range h.connections {
		err := conn.Close()
		if err != nil {
			h.logger.Warn("Failed to close connection", zap.String("conn-id", connID))
		}
	}
	h.connections = nil
	h.logger.Info("All connections closed and map set to nil")
}
