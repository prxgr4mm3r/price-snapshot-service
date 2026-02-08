package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
	"github.com/prxgr4mmer/price-snapshot-service/pkg/retry"
)

const (
	defaultBaseURL = "https://api.binance.com"
	tickerPath     = "/api/v3/ticker/price"
	pingPath       = "/api/v3/ping"
	exchangeInfo   = "/api/v3/exchangeInfo"
)

// Client implements the ExchangeClient interface for Binance
type Client struct {
	httpClient *http.Client
	baseURL    string
	retryConf  retry.Config
	logger     *slog.Logger
}

// ClientOption configures the client
type ClientOption func(*Client)

// WithBaseURL sets the base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetry configures retry behavior
func WithRetry(maxRetries int, backoff time.Duration) ClientOption {
	return func(c *Client) {
		c.retryConf.MaxRetries = maxRetries
		c.retryConf.InitialBackoff = backoff
	}
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger.With("component", "binance_client")
	}
}

// NewClient creates a new Binance client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   defaultBaseURL,
		retryConf: retry.DefaultConfig(),
		logger:    slog.Default().With("component", "binance_client"),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// tickerResponse represents the Binance API ticker response
type tickerResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// GetPrices fetches current prices for multiple symbols
func (c *Client) GetPrices(ctx context.Context, symbols []string) ([]*domain.Price, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	var result []*domain.Price

	err := retry.Do(ctx, c.retryConf, func(ctx context.Context) error {
		// Build URL with symbols parameter
		u, _ := url.Parse(c.baseURL + tickerPath)
		q := u.Query()

		// Format symbols as JSON array: ["BTCUSDT","ETHUSDT"]
		symbolsJSON := fmt.Sprintf(`["%s"]`, strings.Join(symbols, `","`))
		q.Set("symbols", symbolsJSON)
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.logger.Debug("request failed, will retry", "error", err)
			return retry.NewRetryableError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			c.logger.Warn("rate limited by exchange")
			return retry.NewRetryableError(domain.ErrRateLimited)
		}

		if resp.StatusCode >= 500 {
			c.logger.Warn("exchange server error", "status", resp.StatusCode)
			return retry.NewRetryableError(domain.ErrExchangeUnavailable)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			c.logger.Error("unexpected response",
				"status", resp.StatusCode,
				"body", string(body))
			return domain.ErrInvalidResponse
		}

		var tickers []tickerResponse
		if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
			c.logger.Error("failed to decode response", "error", err)
			return fmt.Errorf("failed to decode response: %w", err)
		}

		result = make([]*domain.Price, 0, len(tickers))
		for _, t := range tickers {
			price, err := decimal.NewFromString(t.Price)
			if err != nil {
				c.logger.Warn("invalid price format", "symbol", t.Symbol, "price", t.Price)
				continue
			}
			result = append(result, &domain.Price{
				Symbol: t.Symbol,
				Price:  price,
			})
		}

		return nil
	})

	return result, err
}

// GetPrice fetches the current price for a single symbol
func (c *Client) GetPrice(ctx context.Context, symbol string) (*domain.Price, error) {
	var result *domain.Price

	err := retry.Do(ctx, c.retryConf, func(ctx context.Context) error {
		u, _ := url.Parse(c.baseURL + tickerPath)
		q := u.Query()
		q.Set("symbol", symbol)
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return retry.NewRetryableError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return retry.NewRetryableError(domain.ErrRateLimited)
		}

		if resp.StatusCode == http.StatusBadRequest {
			// Symbol doesn't exist
			return domain.ErrInvalidSymbol
		}

		if resp.StatusCode >= 500 {
			return retry.NewRetryableError(domain.ErrExchangeUnavailable)
		}

		if resp.StatusCode != http.StatusOK {
			return domain.ErrInvalidResponse
		}

		var ticker tickerResponse
		if err := json.NewDecoder(resp.Body).Decode(&ticker); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		price, err := decimal.NewFromString(ticker.Price)
		if err != nil {
			return fmt.Errorf("failed to parse price: %w", err)
		}

		result = &domain.Price{
			Symbol: ticker.Symbol,
			Price:  price,
		}

		return nil
	})

	return result, err
}

// ValidateSymbol checks if a symbol exists on Binance
func (c *Client) ValidateSymbol(ctx context.Context, symbol string) (bool, error) {
	_, err := c.GetPrice(ctx, symbol)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSymbol) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Ping checks if Binance API is reachable
func (c *Client) Ping(ctx context.Context) error {
	return retry.Do(ctx, c.retryConf, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+pingPath, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return retry.NewRetryableError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return retry.NewRetryableError(domain.ErrExchangeUnavailable)
		}

		return nil
	})
}

// Ensure Client implements ExchangeClient
var _ ports.ExchangeClient = (*Client)(nil)
