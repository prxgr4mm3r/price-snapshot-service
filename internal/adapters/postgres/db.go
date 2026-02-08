package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prxgr4mmer/price-snapshot-service/internal/config"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	Pool           *pgxpool.Pool
	config         config.DatabaseConfig
	logger         *slog.Logger
	migrationsPath string
}

// NewDB creates a new PostgreSQL connection pool
func NewDB(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		"max_conns", cfg.MaxOpenConns,
		"min_conns", cfg.MaxIdleConns,
	)

	return &DB{
		Pool:           pool,
		config:         cfg,
		logger:         logger.With("component", "postgres"),
		migrationsPath: "file://migrations",
	}, nil
}

// SetMigrationsPath sets the path to migrations directory
func (db *DB) SetMigrationsPath(path string) {
	db.migrationsPath = path
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	db.logger.Info("running database migrations", "path", db.migrationsPath)

	m, err := migrate.New(db.migrationsPath, db.config.URL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	db.logger.Info("migrations completed",
		"version", version,
		"dirty", dirty,
	)

	return nil
}

// MigrateDown rolls back all migrations
func (db *DB) MigrateDown() error {
	db.logger.Info("rolling back migrations", "path", db.migrationsPath)

	m, err := migrate.New(db.migrationsPath, db.config.URL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.logger.Info("closing database connection")
	db.Pool.Close()
}

// Ping checks if the database is reachable
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
