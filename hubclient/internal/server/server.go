package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/soumya-codes/realtime-hub/hubclient/internal/config"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server represents the hub client server.
type Server struct {
	httpServer *http.Server
	cfg        *config.Config
	logger     *zap.Logger
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	router := gin.Default()
	server := &Server{
		cfg:    cfg,
		logger: logger,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: router,
		},
	}

	// Define the /health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.LoadHTMLFiles("internal/templates/index.html")
	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"hubAddr": cfg.HubAddr,
		})
	})

	return server
}

// Run starts the server and listens for incoming request
func (s *Server) Run() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server ListenAndServe error: ", zap.Error(err))
		}
	}()

	s.logger.Info("Server started", zap.String("addr", s.httpServer.Addr))

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
		return fmt.Errorf("server forced to shutdown %w", err)
	}

	s.logger.Info("Server exiting")
	return nil
}
