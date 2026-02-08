package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpAdapter "github.com/prxgr4mmer/price-snapshot-service/internal/adapters/http"
	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
)

// Mock implementations for testing

type mockSymbolService struct {
	symbols     []*domain.Symbol
	addErr      error
	removeErr   error
	existsValue bool
}

func (m *mockSymbolService) AddSymbol(ctx context.Context, name string) (*domain.Symbol, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	s := &domain.Symbol{ID: 1, Name: name, Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.symbols = append(m.symbols, s)
	return s, nil
}

func (m *mockSymbolService) RemoveSymbol(ctx context.Context, name string) error {
	return m.removeErr
}

func (m *mockSymbolService) ListSymbols(ctx context.Context) ([]*domain.Symbol, error) {
	return m.symbols, nil
}

func (m *mockSymbolService) GetSymbol(ctx context.Context, name string) (*domain.Symbol, error) {
	for _, s := range m.symbols {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, domain.ErrSymbolNotFound
}

func (m *mockSymbolService) SymbolExists(ctx context.Context, name string) (bool, error) {
	return m.existsValue, nil
}

type mockSnapshotService struct {
	snapshots []*domain.PriceSnapshot
	missing   []string
	err       error
}

func (m *mockSnapshotService) GetLatestPrices(ctx context.Context, symbols []string) ([]*domain.PriceSnapshot, []string, error) {
	return m.snapshots, m.missing, m.err
}

func (m *mockSnapshotService) GetPriceHistory(ctx context.Context, symbol string, limit int) ([]*domain.PriceSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshots, nil
}

type mockMetricsService struct{}

func (m *mockMetricsService) GetMetrics(ctx context.Context) (*domain.Metrics, error) {
	return &domain.Metrics{
		Uptime:           3600,
		TrackedSymbols:   5,
		ActiveSymbols:    3,
		TotalSnapshots:   1000,
		PollSuccessCount: 100,
		PollErrorCount:   2,
		DatabaseStatus:   "healthy",
		ExchangeStatus:   "healthy",
	}, nil
}

func (m *mockMetricsService) RecordPollSuccess(duration time.Duration) {}
func (m *mockMetricsService) RecordPollError(duration time.Duration)   {}
func (m *mockMetricsService) GetLastPollTime() *time.Time              { return nil }

type mockExchangeClient struct {
	pingErr error
}

func (m *mockExchangeClient) GetPrice(ctx context.Context, symbol string) (*domain.Price, error) {
	return nil, nil
}

func (m *mockExchangeClient) GetPrices(ctx context.Context, symbols []string) ([]*domain.Price, error) {
	return nil, nil
}

func (m *mockExchangeClient) ValidateSymbol(ctx context.Context, symbol string) (bool, error) {
	return true, nil
}

func (m *mockExchangeClient) Ping(ctx context.Context) error {
	return m.pingErr
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHandler_Health(t *testing.T) {
	t.Run("returns healthy status", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.Health(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	t.Run("returns degraded when exchange is down", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{pingErr: domain.ErrExchangeUnavailable},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.Health(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "degraded", response["status"])
		assert.Equal(t, "unhealthy", response["exchange"])
	})
}

func TestHandler_CreateSymbol(t *testing.T) {
	t.Run("successfully creates symbol", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		body := bytes.NewBufferString(`{"symbol": "BTCUSDT"}`)
		req := httptest.NewRequest(http.MethodPost, "/symbols", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSymbol(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response domain.Symbol
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", response.Name)
	})

	t.Run("returns 400 for empty symbol", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		body := bytes.NewBufferString(`{"symbol": ""}`)
		req := httptest.NewRequest(http.MethodPost, "/symbols", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSymbol(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest(http.MethodPost, "/symbols", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSymbol(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 for invalid symbol format", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{addErr: domain.ErrInvalidSymbol},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		body := bytes.NewBufferString(`{"symbol": "invalid-symbol"}`)
		req := httptest.NewRequest(http.MethodPost, "/symbols", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSymbol(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_ListSymbols(t *testing.T) {
	t.Run("returns list of symbols", func(t *testing.T) {
		mockSvc := &mockSymbolService{
			symbols: []*domain.Symbol{
				{ID: 1, Name: "BTCUSDT", Active: true},
				{ID: 2, Name: "ETHUSDT", Active: true},
			},
		}

		handler := httpAdapter.NewHandler(
			mockSvc,
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/symbols", nil)
		rec := httptest.NewRecorder()

		handler.ListSymbols(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string][]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response["symbols"], 2)
		assert.Contains(t, response["symbols"], "BTCUSDT")
		assert.Contains(t, response["symbols"], "ETHUSDT")
	})
}

func TestHandler_GetHistory(t *testing.T) {
	t.Run("returns price history", func(t *testing.T) {
		now := time.Now()
		mockSvc := &mockSnapshotService{
			snapshots: []*domain.PriceSnapshot{
				{ID: 1, Symbol: "BTCUSDT", Price: decimal.NewFromFloat(43123.45), Timestamp: now},
				{ID: 2, Symbol: "BTCUSDT", Price: decimal.NewFromFloat(43100.00), Timestamp: now.Add(-time.Minute)},
			},
		}

		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			mockSvc,
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/history?symbol=BTCUSDT&limit=100", nil)
		rec := httptest.NewRecorder()

		handler.GetHistory(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", response["symbol"])
		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	t.Run("returns 400 for missing symbol", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		handler.GetHistory(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 404 for unknown symbol", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{err: domain.ErrSymbolNotFound},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/history?symbol=UNKNOWN", nil)
		rec := httptest.NewRecorder()

		handler.GetHistory(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_GetMetrics(t *testing.T) {
	t.Run("returns metrics", func(t *testing.T) {
		handler := httpAdapter.NewHandler(
			&mockSymbolService{},
			&mockSnapshotService{},
			&mockMetricsService{},
			&mockExchangeClient{},
			newTestLogger(),
		)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler.GetMetrics(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response domain.Metrics
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(3600), response.Uptime)
		assert.Equal(t, 5, response.TrackedSymbols)
		assert.Equal(t, "healthy", response.DatabaseStatus)
	})
}
