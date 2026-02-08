package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// SnapshotRepository implements the ports.SnapshotRepository interface
type SnapshotRepository struct {
	db *DB
}

// NewSnapshotRepository creates a new PostgreSQL snapshot repository
func NewSnapshotRepository(db *DB) ports.SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// Create stores a new price snapshot
func (r *SnapshotRepository) Create(ctx context.Context, snapshot *domain.PriceSnapshot) error {
	query := `
		INSERT INTO snapshots (symbol_id, symbol, price, timestamp)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.Pool.QueryRow(ctx, query,
		snapshot.SymbolID,
		snapshot.Symbol,
		snapshot.Price,
		snapshot.Timestamp,
	).Scan(&snapshot.ID)

	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// CreateBatch stores multiple snapshots atomically
func (r *SnapshotRepository) CreateBatch(ctx context.Context, snapshots []*domain.PriceSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO snapshots (symbol_id, symbol, price, timestamp)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	for _, snapshot := range snapshots {
		err := tx.QueryRow(ctx, query,
			snapshot.SymbolID,
			snapshot.Symbol,
			snapshot.Price,
			snapshot.Timestamp,
		).Scan(&snapshot.ID)

		if err != nil {
			return fmt.Errorf("failed to create snapshot for %s: %w", snapshot.Symbol, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLatestBySymbol returns the most recent snapshot for a symbol
func (r *SnapshotRepository) GetLatestBySymbol(ctx context.Context, symbolName string) (*domain.PriceSnapshot, error) {
	query := `
		SELECT id, symbol_id, symbol, price, timestamp
		FROM snapshots
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var snapshot domain.PriceSnapshot
	var priceStr string

	err := r.db.Pool.QueryRow(ctx, query, symbolName).Scan(
		&snapshot.ID,
		&snapshot.SymbolID,
		&snapshot.Symbol,
		&priceStr,
		&snapshot.Timestamp,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}

	snapshot.Price, err = decimal.NewFromString(priceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	return &snapshot, nil
}

// GetLatestBySymbols returns the most recent snapshot for multiple symbols
func (r *SnapshotRepository) GetLatestBySymbols(ctx context.Context, symbolNames []string) ([]*domain.PriceSnapshot, error) {
	if len(symbolNames) == 0 {
		return nil, nil
	}

	query := `
		SELECT DISTINCT ON (symbol)
			id, symbol_id, symbol, price, timestamp
		FROM snapshots
		WHERE symbol = ANY($1)
		ORDER BY symbol, timestamp DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, symbolNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*domain.PriceSnapshot
	for rows.Next() {
		var s domain.PriceSnapshot
		var priceStr string

		if err := rows.Scan(&s.ID, &s.SymbolID, &s.Symbol, &priceStr, &s.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		s.Price, err = decimal.NewFromString(priceStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}

		snapshots = append(snapshots, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// GetHistory returns historical snapshots for a symbol
func (r *SnapshotRepository) GetHistory(ctx context.Context, symbolName string, limit int) ([]*domain.PriceSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, symbol_id, symbol, price, timestamp
		FROM snapshots
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, symbolName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var snapshots []*domain.PriceSnapshot
	for rows.Next() {
		var s domain.PriceSnapshot
		var priceStr string

		if err := rows.Scan(&s.ID, &s.SymbolID, &s.Symbol, &priceStr, &s.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		s.Price, err = decimal.NewFromString(priceStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}

		snapshots = append(snapshots, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// GetHistoryBetween returns snapshots within a time range
func (r *SnapshotRepository) GetHistoryBetween(ctx context.Context, symbolName string, from, to time.Time, limit int) ([]*domain.PriceSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, symbol_id, symbol, price, timestamp
		FROM snapshots
		WHERE symbol = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	rows, err := r.db.Pool.Query(ctx, query, symbolName, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get history between: %w", err)
	}
	defer rows.Close()

	var snapshots []*domain.PriceSnapshot
	for rows.Next() {
		var s domain.PriceSnapshot
		var priceStr string

		if err := rows.Scan(&s.ID, &s.SymbolID, &s.Symbol, &priceStr, &s.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		s.Price, err = decimal.NewFromString(priceStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}

		snapshots = append(snapshots, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// Count returns total number of snapshots
func (r *SnapshotRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM snapshots`

	var count int64
	if err := r.db.Pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count snapshots: %w", err)
	}

	return count, nil
}

// CountBySymbol returns number of snapshots for a symbol
func (r *SnapshotRepository) CountBySymbol(ctx context.Context, symbolName string) (int64, error) {
	query := `SELECT COUNT(*) FROM snapshots WHERE symbol = $1`

	var count int64
	if err := r.db.Pool.QueryRow(ctx, query, symbolName).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count snapshots by symbol: %w", err)
	}

	return count, nil
}

// Prune removes snapshots older than the given time
func (r *SnapshotRepository) Prune(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM snapshots WHERE timestamp < $1`

	result, err := r.db.Pool.Exec(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to prune snapshots: %w", err)
	}

	return result.RowsAffected(), nil
}

// Ensure SnapshotRepository implements ports.SnapshotRepository
var _ ports.SnapshotRepository = (*SnapshotRepository)(nil)
