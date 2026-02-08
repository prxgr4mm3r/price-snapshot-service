package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// PriceSnapshot represents a point-in-time price capture
type PriceSnapshot struct {
	ID        int64           `json:"id"`
	SymbolID  int64           `json:"symbol_id"`
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewPriceSnapshot creates a new price snapshot
func NewPriceSnapshot(symbolID int64, symbol string, price decimal.Decimal) *PriceSnapshot {
	return &PriceSnapshot{
		SymbolID:  symbolID,
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now().UTC(),
	}
}

// Price represents a current price from the exchange
type Price struct {
	Symbol string          `json:"symbol"`
	Price  decimal.Decimal `json:"price"`
}

// Metrics represents operational metrics
type Metrics struct {
	Uptime           float64    `json:"uptime_seconds"`
	TrackedSymbols   int        `json:"tracked_symbols"`
	ActiveSymbols    int        `json:"active_symbols"`
	TotalSnapshots   int64      `json:"total_snapshots"`
	LastPollTime     *time.Time `json:"last_poll_time,omitempty"`
	LastPollDuration float64    `json:"last_poll_duration_ms"`
	PollSuccessCount int64      `json:"poll_success_count"`
	PollErrorCount   int64      `json:"poll_error_count"`
	DatabaseStatus   string     `json:"database_status"`
	ExchangeStatus   string     `json:"exchange_status"`
}
