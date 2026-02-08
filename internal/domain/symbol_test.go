package domain_test

import (
	"testing"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSymbolName(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		wantErr error
	}{
		{
			name:    "valid symbol BTCUSDT",
			symbol:  "BTCUSDT",
			wantErr: nil,
		},
		{
			name:    "valid symbol ETHUSDT",
			symbol:  "ETHUSDT",
			wantErr: nil,
		},
		{
			name:    "valid symbol with numbers",
			symbol:  "1INCHUSDT",
			wantErr: nil,
		},
		{
			name:    "empty symbol",
			symbol:  "",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "too short symbol",
			symbol:  "A",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "too long symbol",
			symbol:  "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "lowercase symbol",
			symbol:  "btcusdt",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "symbol with dash",
			symbol:  "BTC-USDT",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "symbol with underscore",
			symbol:  "BTC_USDT",
			wantErr: domain.ErrInvalidSymbol,
		},
		{
			name:    "symbol with space",
			symbol:  "BTC USDT",
			wantErr: domain.ErrInvalidSymbol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateSymbolName(tt.symbol)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewSymbol(t *testing.T) {
	t.Run("creates valid symbol", func(t *testing.T) {
		symbol, err := domain.NewSymbol("btcusdt")
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", symbol.Name)
		assert.True(t, symbol.Active)
		assert.NotZero(t, symbol.CreatedAt)
		assert.NotZero(t, symbol.UpdatedAt)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		symbol, err := domain.NewSymbol("  ETHUSDT  ")
		require.NoError(t, err)
		assert.Equal(t, "ETHUSDT", symbol.Name)
	})

	t.Run("rejects invalid symbol", func(t *testing.T) {
		_, err := domain.NewSymbol("invalid-symbol")
		assert.ErrorIs(t, err, domain.ErrInvalidSymbol)
	})
}

func TestSymbol_Deactivate(t *testing.T) {
	symbol, err := domain.NewSymbol("BTCUSDT")
	require.NoError(t, err)
	assert.True(t, symbol.Active)

	originalUpdatedAt := symbol.UpdatedAt
	symbol.Deactivate()

	assert.False(t, symbol.Active)
	assert.True(t, symbol.UpdatedAt.After(originalUpdatedAt) || symbol.UpdatedAt.Equal(originalUpdatedAt))
}

func TestSymbol_Activate(t *testing.T) {
	symbol, err := domain.NewSymbol("BTCUSDT")
	require.NoError(t, err)
	symbol.Deactivate()
	assert.False(t, symbol.Active)

	symbol.Activate()
	assert.True(t, symbol.Active)
}
