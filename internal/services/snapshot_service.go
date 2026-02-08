package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// SnapshotService implements the ports.SnapshotService interface
type SnapshotService struct {
	snapshotRepo ports.SnapshotRepository
	symbolRepo   ports.SymbolRepository
	logger       *slog.Logger
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(
	snapshotRepo ports.SnapshotRepository,
	symbolRepo ports.SymbolRepository,
	logger *slog.Logger,
) *SnapshotService {
	return &SnapshotService{
		snapshotRepo: snapshotRepo,
		symbolRepo:   symbolRepo,
		logger:       logger.With("component", "snapshot_service"),
	}
}

// GetLatestPrices returns current prices for specified symbols
// Returns the prices found, the list of missing symbols, and any error
func (s *SnapshotService) GetLatestPrices(ctx context.Context, symbols []string) ([]*domain.PriceSnapshot, []string, error) {
	if len(symbols) == 0 {
		return nil, nil, nil
	}

	// Normalize symbols
	normalizedSymbols := make([]string, len(symbols))
	for i, sym := range symbols {
		normalizedSymbols[i] = strings.ToUpper(strings.TrimSpace(sym))
	}

	// Get latest snapshots
	snapshots, err := s.snapshotRepo.GetLatestBySymbols(ctx, normalizedSymbols)
	if err != nil {
		s.logger.Error("failed to get latest prices", "error", err)
		return nil, nil, domain.ErrInternal
	}

	// Find missing symbols
	foundSymbols := make(map[string]bool)
	for _, snap := range snapshots {
		foundSymbols[snap.Symbol] = true
	}

	var missing []string
	for _, sym := range normalizedSymbols {
		if !foundSymbols[sym] {
			missing = append(missing, sym)
		}
	}

	return snapshots, missing, nil
}

// GetPriceHistory returns historical prices for a symbol
func (s *SnapshotService) GetPriceHistory(ctx context.Context, symbol string, limit int) ([]*domain.PriceSnapshot, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Validate limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Check if symbol is tracked
	exists, err := s.symbolRepo.Exists(ctx, symbol)
	if err != nil {
		s.logger.Error("failed to check symbol existence", "symbol", symbol, "error", err)
		return nil, domain.ErrInternal
	}
	if !exists {
		return nil, domain.ErrSymbolNotFound
	}

	// Get history
	history, err := s.snapshotRepo.GetHistory(ctx, symbol, limit)
	if err != nil {
		s.logger.Error("failed to get price history", "symbol", symbol, "error", err)
		return nil, domain.ErrInternal
	}

	return history, nil
}

// Ensure SnapshotService implements ports.SnapshotService
var _ ports.SnapshotService = (*SnapshotService)(nil)
