package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/redis"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/config"
	"github.com/soumya-codes/realtime-hub/hubserver/internal/websocket"
	"go.uber.org/zap"
)

// Server represents the hub server.
type Server struct {
	httpServer     *http.Server
	messageHandler *websocket.MessageHandler
	logger         *zap.Logger
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Initialize Redis client
	redisClient := redis.NewClient(cfg.PubSubHostName, cfg.RedisUsername, cfg.RedisPassword, logger)
	if err := redisClient.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize MessageHandler
	messageHandler, err := websocket.NewMessageHandler(redisClient, cfg.PubSubChannelName, cfg.HubName, cfg.BroadcastWorkers, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create message handler: %w", err)
	}

	// Initialize Gin Router
	router := gin.Default()

	// Define the /health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Define the WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		messageHandler.ServeHTTP(c.Writer, c.Request)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	return &Server{
		httpServer:     httpServer,
		messageHandler: messageHandler,
		logger:         logger,
	}, nil
}

// Run starts the server and listens for incoming connections.
func (s *Server) Run() error {
	// Start the MessageHandler
	go s.messageHandler.Run()
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal("HTTP server ListenAndServe", zap.Error(err))
		}
	}()
	s.logger.Info("Server started", zap.String("addr", s.httpServer.Addr))

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// Clean up resources
	if err := s.messageHandler.Close(); err != nil {
		s.logger.Error("Error closing message handler", zap.Error(err))
	}

	s.logger.Info("Server exiting")

	return nil
}
