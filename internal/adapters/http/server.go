package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/config"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// Server wraps the HTTP server with graceful shutdown
type Server struct {
	server *http.Server
	config config.ServerConfig
	logger *slog.Logger
}

// NewServer creates a new HTTP server
func NewServer(
	cfg config.ServerConfig,
	symbolSvc ports.SymbolService,
	snapshotSvc ports.SnapshotService,
	metricsSvc ports.MetricsService,
	exchange ports.ExchangeClient,
	logger *slog.Logger,
) *Server {
	handler := NewHandler(symbolSvc, snapshotSvc, metricsSvc, exchange, logger)
	router := NewRouter(handler, logger)

	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      router,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		config: cfg,
		logger: logger.With("component", "http_server"),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting http server", "addr", s.server.Addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")

	// Create a deadline for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.server.Addr
}
