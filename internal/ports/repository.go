package ports

import (
	"context"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
)

// SymbolRepository defines the contract for symbol persistence
type SymbolRepository interface {
	// Create adds a new symbol to track
	Create(ctx context.Context, symbol *domain.Symbol) error

	// GetByName retrieves a symbol by its name
	GetByName(ctx context.Context, name string) (*domain.Symbol, error)

	// GetByID retrieves a symbol by its ID
	GetByID(ctx context.Context, id int64) (*domain.Symbol, error)

	// List returns all tracked symbols
	List(ctx context.Context) ([]*domain.Symbol, error)

	// ListActive returns only active symbols
	ListActive(ctx context.Context) ([]*domain.Symbol, error)

	// Delete removes a symbol by name
	Delete(ctx context.Context, name string) error

	// Update modifies an existing symbol
	Update(ctx context.Context, symbol *domain.Symbol) error

	// Count returns total number of symbols
	Count(ctx context.Context) (int, error)

	// CountActive returns number of active symbols
	CountActive(ctx context.Context) (int, error)

	// Exists checks if a symbol exists
	Exists(ctx context.Context, name string) (bool, error)
}

// SnapshotRepository defines the contract for snapshot persistence
type SnapshotRepository interface {
	// Create stores a new price snapshot
	Create(ctx context.Context, snapshot *domain.PriceSnapshot) error

	// CreateBatch stores multiple snapshots atomically
	CreateBatch(ctx context.Context, snapshots []*domain.PriceSnapshot) error

	// GetLatestBySymbol returns the most recent snapshot for a symbol
	GetLatestBySymbol(ctx context.Context, symbolName string) (*domain.PriceSnapshot, error)

	// GetLatestBySymbols returns the most recent snapshot for multiple symbols
	GetLatestBySymbols(ctx context.Context, symbolNames []string) ([]*domain.PriceSnapshot, error)

	// GetHistory returns historical snapshots for a symbol
	GetHistory(ctx context.Context, symbolName string, limit int) ([]*domain.PriceSnapshot, error)

	// GetHistoryBetween returns snapshots within a time range
	GetHistoryBetween(ctx context.Context, symbolName string, from, to time.Time, limit int) ([]*domain.PriceSnapshot, error)

	// Count returns total number of snapshots
	Count(ctx context.Context) (int64, error)

	// CountBySymbol returns number of snapshots for a symbol
	CountBySymbol(ctx context.Context, symbolName string) (int64, error)

	// Prune removes snapshots older than the given time
	Prune(ctx context.Context, olderThan time.Time) (int64, error)
}
