package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/adapters/binance"
	httpAdapter "github.com/prxgr4mmer/price-snapshot-service/internal/adapters/http"
	"github.com/prxgr4mmer/price-snapshot-service/internal/adapters/postgres"
	"github.com/prxgr4mmer/price-snapshot-service/internal/config"
	"github.com/prxgr4mmer/price-snapshot-service/internal/services"
	"github.com/prxgr4mmer/price-snapshot-service/internal/worker"
)

func main() {
	// Initialize logger
	logger := initLogger()
	slog.SetDefault(logger)

	logger.Info("starting crypto snapshot service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build and start application
	app, err := buildApplication(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to build application", "error", err)
		os.Exit(1)
	}

	// Start application components
	if err := app.Start(ctx); err != nil {
		logger.Error("failed to start application", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	waitForShutdown(ctx, cancel, app, logger)
}

func initLogger() *slog.Logger {
	logLevel := os.Getenv("LOG_LEVEL")
	logFormat := os.Getenv("LOG_FORMAT")

	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if logFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// Application holds all components
type Application struct {
	db         *postgres.DB
	httpServer *httpAdapter.Server
	poller     *worker.Poller
	logger     *slog.Logger
}

func buildApplication(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Application, error) {
	logger.Info("building application")

	// 1. Infrastructure Layer - Database
	db, err := postgres.NewDB(ctx, cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		db.Close()
		return nil, err
	}

	// 2. Infrastructure Layer - Repositories
	symbolRepo := postgres.NewSymbolRepository(db)
	snapshotRepo := postgres.NewSnapshotRepository(db)

	// 3. Infrastructure Layer - Exchange Client
	exchangeClient := binance.NewClient(
		binance.WithBaseURL(cfg.Exchange.BaseURL),
		binance.WithTimeout(cfg.Exchange.Timeout),
		binance.WithRetry(cfg.Exchange.MaxRetries, cfg.Exchange.RetryBackoff),
		binance.WithLogger(logger),
	)

	// 4. Service Layer
	metricsService := services.NewMetricsService(
		symbolRepo,
		snapshotRepo,
		exchangeClient,
		logger,
	)

	symbolService := services.NewSymbolService(
		symbolRepo,
		exchangeClient,
		logger,
	)

	snapshotService := services.NewSnapshotService(
		snapshotRepo,
		symbolRepo,
		logger,
	)

	pollerService := services.NewPollerService(
		symbolRepo,
		snapshotRepo,
		exchangeClient,
		metricsService,
		logger,
	)

	// 5. Transport Layer - HTTP Server
	httpServer := httpAdapter.NewServer(
		cfg.Server,
		symbolService,
		snapshotService,
		metricsService,
		exchangeClient,
		logger,
	)

	// 6. Background Workers
	poller := worker.NewPoller(
		pollerService,
		cfg.Poller.Interval,
		logger,
	)

	logger.Info("application built successfully")

	return &Application{
		db:         db,
		httpServer: httpServer,
		poller:     poller,
		logger:     logger,
	}, nil
}

func (a *Application) Start(ctx context.Context) error {
	a.logger.Info("starting application components")

	// Start poller in background
	go func() {
		if err := a.poller.Start(ctx); err != nil {
			a.logger.Error("poller error", "error", err)
		}
	}()

	// Start HTTP server in background (will block until shutdown)
	go func() {
		if err := a.httpServer.Start(); err != nil {
			a.logger.Error("http server error", "error", err)
		}
	}()

	a.logger.Info("application started",
		"http_addr", a.httpServer.Addr(),
	)

	return nil
}

func (a *Application) Shutdown() {
	a.logger.Info("shutting down application")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop poller first
	if err := a.poller.Stop(); err != nil {
		a.logger.Error("failed to stop poller", "error", err)
	}

	// Stop HTTP server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error("failed to shutdown http server", "error", err)
	}

	// Close database connection
	a.db.Close()

	a.logger.Info("application shutdown complete")
}

func waitForShutdown(ctx context.Context, cancel context.CancelFunc, app *Application, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
		app.Shutdown()
	case <-ctx.Done():
		app.Shutdown()
	}
}
