package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// MetricsService implements the ports.MetricsService interface
type MetricsService struct {
	symbolRepo   ports.SymbolRepository
	snapshotRepo ports.SnapshotRepository
	exchange     ports.ExchangeClient
	startTime    time.Time
	logger       *slog.Logger

	mu               sync.RWMutex
	lastPollTime     *time.Time
	lastPollDuration time.Duration
	pollSuccessCount int64
	pollErrorCount   int64
	totalPollTime    time.Duration
}

// NewMetricsService creates a new metrics service
func NewMetricsService(
	symbolRepo ports.SymbolRepository,
	snapshotRepo ports.SnapshotRepository,
	exchange ports.ExchangeClient,
	logger *slog.Logger,
) *MetricsService {
	return &MetricsService{
		symbolRepo:   symbolRepo,
		snapshotRepo: snapshotRepo,
		exchange:     exchange,
		startTime:    time.Now(),
		logger:       logger.With("component", "metrics_service"),
	}
}

// GetMetrics returns current operational metrics
func (m *MetricsService) GetMetrics(ctx context.Context) (*domain.Metrics, error) {
	m.mu.RLock()
	lastPollTime := m.lastPollTime
	lastPollDuration := m.lastPollDuration
	pollSuccessCount := m.pollSuccessCount
	pollErrorCount := m.pollErrorCount
	m.mu.RUnlock()

	// Get symbol counts
	totalSymbols, err := m.symbolRepo.Count(ctx)
	if err != nil {
		m.logger.Error("failed to count symbols", "error", err)
		totalSymbols = 0
	}

	activeSymbols, err := m.symbolRepo.CountActive(ctx)
	if err != nil {
		m.logger.Error("failed to count active symbols", "error", err)
		activeSymbols = 0
	}

	// Get snapshot count
	totalSnapshots, err := m.snapshotRepo.Count(ctx)
	if err != nil {
		m.logger.Error("failed to count snapshots", "error", err)
		totalSnapshots = 0
	}

	// Check database status
	dbStatus := "healthy"
	if err := m.checkDatabaseHealth(ctx); err != nil {
		dbStatus = "unhealthy"
	}

	// Check exchange status
	exchangeStatus := "healthy"
	if err := m.exchange.Ping(ctx); err != nil {
		exchangeStatus = "unhealthy"
	}

	return &domain.Metrics{
		Uptime:           time.Since(m.startTime).Seconds(),
		TrackedSymbols:   totalSymbols,
		ActiveSymbols:    activeSymbols,
		TotalSnapshots:   totalSnapshots,
		LastPollTime:     lastPollTime,
		LastPollDuration: float64(lastPollDuration.Milliseconds()),
		PollSuccessCount: pollSuccessCount,
		PollErrorCount:   pollErrorCount,
		DatabaseStatus:   dbStatus,
		ExchangeStatus:   exchangeStatus,
	}, nil
}

// RecordPollSuccess records a successful poll
func (m *MetricsService) RecordPollSuccess(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.lastPollTime = &now
	m.lastPollDuration = duration
	m.pollSuccessCount++
	m.totalPollTime += duration
}

// RecordPollError records a failed poll
func (m *MetricsService) RecordPollError(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.lastPollTime = &now
	m.lastPollDuration = duration
	m.pollErrorCount++
	m.totalPollTime += duration
}

// GetLastPollTime returns the time of the last poll
func (m *MetricsService) GetLastPollTime() *time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPollTime
}

func (m *MetricsService) checkDatabaseHealth(ctx context.Context) error {
	// Simple health check - count symbols
	_, err := m.symbolRepo.Count(ctx)
	return err
}

// Ensure MetricsService implements ports.MetricsService
var _ ports.MetricsService = (*MetricsService)(nil)
