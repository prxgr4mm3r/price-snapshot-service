package ports

import (
	"context"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
)

// SymbolService defines the contract for symbol management
type SymbolService interface {
	// AddSymbol adds a new symbol to track
	AddSymbol(ctx context.Context, name string) (*domain.Symbol, error)

	// RemoveSymbol stops tracking a symbol
	RemoveSymbol(ctx context.Context, name string) error

	// ListSymbols returns all tracked symbols
	ListSymbols(ctx context.Context) ([]*domain.Symbol, error)

	// GetSymbol retrieves a specific symbol
	GetSymbol(ctx context.Context, name string) (*domain.Symbol, error)

	// SymbolExists checks if a symbol is being tracked
	SymbolExists(ctx context.Context, name string) (bool, error)
}

// SnapshotService defines the contract for price queries
type SnapshotService interface {
	// GetLatestPrices returns current prices for specified symbols
	GetLatestPrices(ctx context.Context, symbols []string) ([]*domain.PriceSnapshot, []string, error)

	// GetPriceHistory returns historical prices for a symbol
	GetPriceHistory(ctx context.Context, symbol string, limit int) ([]*domain.PriceSnapshot, error)
}

// MetricsService defines the contract for operational metrics
type MetricsService interface {
	// GetMetrics returns current operational metrics
	GetMetrics(ctx context.Context) (*domain.Metrics, error)

	// RecordPollSuccess records a successful poll
	RecordPollSuccess(duration time.Duration)

	// RecordPollError records a failed poll
	RecordPollError(duration time.Duration)

	// GetLastPollTime returns the time of the last poll
	GetLastPollTime() *time.Time
}

// PollerService defines the contract for price polling orchestration
type PollerService interface {
	// PollPrices fetches and stores prices for all active symbols
	PollPrices(ctx context.Context) error
}

// HealthService defines the contract for health checks
type HealthService interface {
	// CheckHealth performs health checks on all dependencies
	CheckHealth(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus represents the health of the service
type HealthStatus struct {
	Status   string            `json:"status"`
	Database string            `json:"database"`
	Exchange string            `json:"exchange"`
	Details  map[string]string `json:"details,omitempty"`
}
