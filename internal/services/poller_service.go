package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// PollerService implements the ports.PollerService interface
type PollerService struct {
	symbolRepo   ports.SymbolRepository
	snapshotRepo ports.SnapshotRepository
	exchange     ports.ExchangeClient
	metrics      ports.MetricsService
	logger       *slog.Logger
}

// NewPollerService creates a new poller service
func NewPollerService(
	symbolRepo ports.SymbolRepository,
	snapshotRepo ports.SnapshotRepository,
	exchange ports.ExchangeClient,
	metrics ports.MetricsService,
	logger *slog.Logger,
) *PollerService {
	return &PollerService{
		symbolRepo:   symbolRepo,
		snapshotRepo: snapshotRepo,
		exchange:     exchange,
		metrics:      metrics,
		logger:       logger.With("component", "poller_service"),
	}
}

// PollPrices fetches and stores prices for all active symbols
func (p *PollerService) PollPrices(ctx context.Context) error {
	start := time.Now()

	// Get active symbols
	symbols, err := p.symbolRepo.ListActive(ctx)
	if err != nil {
		p.logger.Error("failed to list active symbols", "error", err)
		p.metrics.RecordPollError(time.Since(start))
		return err
	}

	if len(symbols) == 0 {
		p.logger.Debug("no active symbols to poll")
		return nil
	}

	// Extract symbol names and create lookup map
	symbolNames := make([]string, len(symbols))
	symbolMap := make(map[string]*domain.Symbol)
	for i, s := range symbols {
		symbolNames[i] = s.Name
		symbolMap[s.Name] = s
	}

	p.logger.Debug("polling prices", "symbols", len(symbols))

	// Fetch prices from exchange
	prices, err := p.exchange.GetPrices(ctx, symbolNames)
	if err != nil {
		p.logger.Error("failed to fetch prices from exchange", "error", err)
		p.metrics.RecordPollError(time.Since(start))
		return err
	}

	// Create snapshots
	now := time.Now().UTC()
	snapshots := make([]*domain.PriceSnapshot, 0, len(prices))
	for _, price := range prices {
		if sym, ok := symbolMap[price.Symbol]; ok {
			snapshots = append(snapshots, &domain.PriceSnapshot{
				SymbolID:  sym.ID,
				Symbol:    price.Symbol,
				Price:     price.Price,
				Timestamp: now,
			})
		}
	}

	if len(snapshots) == 0 {
		p.logger.Warn("no prices to store")
		p.metrics.RecordPollSuccess(time.Since(start))
		return nil
	}

	// Store snapshots
	if err := p.snapshotRepo.CreateBatch(ctx, snapshots); err != nil {
		p.logger.Error("failed to store snapshots", "error", err)
		p.metrics.RecordPollError(time.Since(start))
		return err
	}

	duration := time.Since(start)
	p.metrics.RecordPollSuccess(duration)

	p.logger.Info("poll completed",
		"symbols", len(symbols),
		"snapshots", len(snapshots),
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// Ensure PollerService implements ports.PollerService
var _ ports.PollerService = (*PollerService)(nil)
