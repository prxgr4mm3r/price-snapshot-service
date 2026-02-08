package ports

import (
	"context"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
)

// ExchangeClient defines the contract for fetching prices from an exchange
type ExchangeClient interface {
	// GetPrice fetches the current price for a single symbol
	GetPrice(ctx context.Context, symbol string) (*domain.Price, error)

	// GetPrices fetches current prices for multiple symbols
	GetPrices(ctx context.Context, symbols []string) ([]*domain.Price, error)

	// ValidateSymbol checks if a symbol exists on the exchange
	ValidateSymbol(ctx context.Context, symbol string) (bool, error)

	// Ping checks if the exchange is reachable
	Ping(ctx context.Context) error
}
