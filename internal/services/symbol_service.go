package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// SymbolService implements the ports.SymbolService interface
type SymbolService struct {
	repo     ports.SymbolRepository
	exchange ports.ExchangeClient
	logger   *slog.Logger
}

// NewSymbolService creates a new symbol service
func NewSymbolService(
	repo ports.SymbolRepository,
	exchange ports.ExchangeClient,
	logger *slog.Logger,
) *SymbolService {
	return &SymbolService{
		repo:     repo,
		exchange: exchange,
		logger:   logger.With("component", "symbol_service"),
	}
}

// AddSymbol adds a new symbol to track
func (s *SymbolService) AddSymbol(ctx context.Context, name string) (*domain.Symbol, error) {
	name = strings.ToUpper(strings.TrimSpace(name))

	// Create and validate symbol
	symbol, err := domain.NewSymbol(name)
	if err != nil {
		return nil, err
	}

	// Check if already tracked
	exists, err := s.repo.Exists(ctx, name)
	if err != nil {
		s.logger.Error("failed to check symbol existence", "symbol", name, "error", err)
		return nil, domain.ErrInternal
	}
	if exists {
		return nil, domain.ErrSymbolExists
	}

	// Validate symbol exists on exchange
	valid, err := s.exchange.ValidateSymbol(ctx, name)
	if err != nil {
		s.logger.Error("failed to validate symbol on exchange",
			"symbol", name, "error", err)
		return nil, domain.ErrExchangeUnavailable
	}
	if !valid {
		return nil, domain.ErrInvalidSymbol
	}

	// Create in repository
	if err := s.repo.Create(ctx, symbol); err != nil {
		s.logger.Error("failed to create symbol", "symbol", name, "error", err)
		return nil, domain.ErrInternal
	}

	s.logger.Info("symbol added", "symbol", name, "id", symbol.ID)
	return symbol, nil
}

// RemoveSymbol stops tracking a symbol
func (s *SymbolService) RemoveSymbol(ctx context.Context, name string) error {
	name = strings.ToUpper(strings.TrimSpace(name))

	if err := s.repo.Delete(ctx, name); err != nil {
		if errors.Is(err, domain.ErrSymbolNotFound) {
			return err
		}
		s.logger.Error("failed to delete symbol", "symbol", name, "error", err)
		return domain.ErrInternal
	}

	s.logger.Info("symbol removed", "symbol", name)
	return nil
}

// ListSymbols returns all tracked symbols
func (s *SymbolService) ListSymbols(ctx context.Context) ([]*domain.Symbol, error) {
	symbols, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Error("failed to list symbols", "error", err)
		return nil, domain.ErrInternal
	}
	return symbols, nil
}

// GetSymbol retrieves a specific symbol
func (s *SymbolService) GetSymbol(ctx context.Context, name string) (*domain.Symbol, error) {
	name = strings.ToUpper(strings.TrimSpace(name))
	return s.repo.GetByName(ctx, name)
}

// SymbolExists checks if a symbol is being tracked
func (s *SymbolService) SymbolExists(ctx context.Context, name string) (bool, error) {
	name = strings.ToUpper(strings.TrimSpace(name))
	return s.repo.Exists(ctx, name)
}

// Ensure SymbolService implements ports.SymbolService
var _ ports.SymbolService = (*SymbolService)(nil)
