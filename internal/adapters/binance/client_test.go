package binance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/adapters/binance"
	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetPrice(t *testing.T) {
	t.Run("successfully fetches price", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/price", r.URL.Path)
			assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"symbol": "BTCUSDT",
				"price":  "43123.45",
			})
		}))
		defer server.Close()

		client := binance.NewClient(
			binance.WithBaseURL(server.URL),
			binance.WithTimeout(5*time.Second),
		)

		price, err := client.GetPrice(context.Background(), "BTCUSDT")
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", price.Symbol)
		assert.True(t, price.Price.Equal(decimal.NewFromFloat(43123.45)))
	})

	t.Run("returns error for invalid symbol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": -1121,
				"msg":  "Invalid symbol.",
			})
		}))
		defer server.Close()

		client := binance.NewClient(binance.WithBaseURL(server.URL))

		_, err := client.GetPrice(context.Background(), "INVALID")
		assert.ErrorIs(t, err, domain.ErrInvalidSymbol)
	})

	t.Run("handles rate limiting", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount <= 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"symbol": "BTCUSDT",
				"price":  "43123.45",
			})
		}))
		defer server.Close()

		client := binance.NewClient(
			binance.WithBaseURL(server.URL),
			binance.WithRetry(3, 10*time.Millisecond),
		)

		price, err := client.GetPrice(context.Background(), "BTCUSDT")
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", price.Symbol)
		assert.Equal(t, 3, callCount) // Retried twice
	})
}

func TestClient_GetPrices(t *testing.T) {
	t.Run("successfully fetches multiple prices", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/price", r.URL.Path)
			assert.Contains(t, r.URL.Query().Get("symbols"), "BTCUSDT")
			assert.Contains(t, r.URL.Query().Get("symbols"), "ETHUSDT")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]string{
				{"symbol": "BTCUSDT", "price": "43123.45"},
				{"symbol": "ETHUSDT", "price": "2345.67"},
			})
		}))
		defer server.Close()

		client := binance.NewClient(binance.WithBaseURL(server.URL))

		prices, err := client.GetPrices(context.Background(), []string{"BTCUSDT", "ETHUSDT"})
		require.NoError(t, err)
		require.Len(t, prices, 2)

		btcPrice := findPrice(prices, "BTCUSDT")
		require.NotNil(t, btcPrice)
		assert.True(t, btcPrice.Price.Equal(decimal.NewFromFloat(43123.45)))

		ethPrice := findPrice(prices, "ETHUSDT")
		require.NotNil(t, ethPrice)
		assert.True(t, ethPrice.Price.Equal(decimal.NewFromFloat(2345.67)))
	})

	t.Run("returns empty for empty symbols", func(t *testing.T) {
		client := binance.NewClient()
		prices, err := client.GetPrices(context.Background(), []string{})
		require.NoError(t, err)
		assert.Empty(t, prices)
	})
}

func TestClient_ValidateSymbol(t *testing.T) {
	t.Run("returns true for valid symbol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"symbol": "BTCUSDT",
				"price":  "43123.45",
			})
		}))
		defer server.Close()

		client := binance.NewClient(binance.WithBaseURL(server.URL))

		valid, err := client.ValidateSymbol(context.Background(), "BTCUSDT")
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("returns false for invalid symbol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := binance.NewClient(binance.WithBaseURL(server.URL))

		valid, err := client.ValidateSymbol(context.Background(), "INVALID")
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestClient_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ping", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := binance.NewClient(binance.WithBaseURL(server.URL))

		err := client.Ping(context.Background())
		require.NoError(t, err)
	})

	t.Run("ping failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := binance.NewClient(
			binance.WithBaseURL(server.URL),
			binance.WithRetry(1, 10*time.Millisecond),
		)

		err := client.Ping(context.Background())
		assert.Error(t, err)
	})
}

func findPrice(prices []*domain.Price, symbol string) *domain.Price {
	for _, p := range prices {
		if p.Symbol == symbol {
			return p
		}
	}
	return nil
}
