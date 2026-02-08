package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// SymbolRepository implements the ports.SymbolRepository interface
type SymbolRepository struct {
	db *DB
}

// NewSymbolRepository creates a new PostgreSQL symbol repository
func NewSymbolRepository(db *DB) ports.SymbolRepository {
	return &SymbolRepository{db: db}
}

// Create adds a new symbol to track
func (r *SymbolRepository) Create(ctx context.Context, symbol *domain.Symbol) error {
	query := `
		INSERT INTO symbols (name, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.Pool.QueryRow(ctx, query,
		symbol.Name,
		symbol.Active,
		symbol.CreatedAt,
		symbol.UpdatedAt,
	).Scan(&symbol.ID)

	if err != nil {
		return fmt.Errorf("failed to create symbol: %w", err)
	}

	return nil
}

// GetByName retrieves a symbol by its name
func (r *SymbolRepository) GetByName(ctx context.Context, name string) (*domain.Symbol, error) {
	query := `
		SELECT id, name, active, created_at, updated_at
		FROM symbols
		WHERE name = $1
	`

	var symbol domain.Symbol
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&symbol.ID,
		&symbol.Name,
		&symbol.Active,
		&symbol.CreatedAt,
		&symbol.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSymbolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	return &symbol, nil
}

// GetByID retrieves a symbol by its ID
func (r *SymbolRepository) GetByID(ctx context.Context, id int64) (*domain.Symbol, error) {
	query := `
		SELECT id, name, active, created_at, updated_at
		FROM symbols
		WHERE id = $1
	`

	var symbol domain.Symbol
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&symbol.ID,
		&symbol.Name,
		&symbol.Active,
		&symbol.CreatedAt,
		&symbol.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSymbolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	return &symbol, nil
}

// List returns all tracked symbols
func (r *SymbolRepository) List(ctx context.Context) ([]*domain.Symbol, error) {
	query := `
		SELECT id, name, active, created_at, updated_at
		FROM symbols
		ORDER BY name
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list symbols: %w", err)
	}
	defer rows.Close()

	var symbols []*domain.Symbol
	for rows.Next() {
		var s domain.Symbol
		if err := rows.Scan(&s.ID, &s.Name, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	return symbols, nil
}

// ListActive returns only active symbols
func (r *SymbolRepository) ListActive(ctx context.Context) ([]*domain.Symbol, error) {
	query := `
		SELECT id, name, active, created_at, updated_at
		FROM symbols
		WHERE active = TRUE
		ORDER BY name
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active symbols: %w", err)
	}
	defer rows.Close()

	var symbols []*domain.Symbol
	for rows.Next() {
		var s domain.Symbol
		if err := rows.Scan(&s.ID, &s.Name, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	return symbols, nil
}

// Delete removes a symbol by name
func (r *SymbolRepository) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM symbols WHERE name = $1`

	result, err := r.db.Pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete symbol: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSymbolNotFound
	}

	return nil
}

// Update modifies an existing symbol
func (r *SymbolRepository) Update(ctx context.Context, symbol *domain.Symbol) error {
	query := `
		UPDATE symbols
		SET name = $1, active = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.Pool.Exec(ctx, query, symbol.Name, symbol.Active, symbol.ID)
	if err != nil {
		return fmt.Errorf("failed to update symbol: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSymbolNotFound
	}

	return nil
}

// Count returns total number of symbols
func (r *SymbolRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM symbols`

	var count int
	if err := r.db.Pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count symbols: %w", err)
	}

	return count, nil
}

// CountActive returns number of active symbols
func (r *SymbolRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM symbols WHERE active = TRUE`

	var count int
	if err := r.db.Pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count active symbols: %w", err)
	}

	return count, nil
}

// Exists checks if a symbol exists
func (r *SymbolRepository) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM symbols WHERE name = $1)`

	var exists bool
	if err := r.db.Pool.QueryRow(ctx, query, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check symbol existence: %w", err)
	}

	return exists, nil
}

// Ensure SymbolRepository implements ports.SymbolRepository
var _ ports.SymbolRepository = (*SymbolRepository)(nil)
