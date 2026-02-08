package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// Handler contains all HTTP handlers
type Handler struct {
	symbolSvc   ports.SymbolService
	snapshotSvc ports.SnapshotService
	metricsSvc  ports.MetricsService
	exchange    ports.ExchangeClient
	logger      *slog.Logger
}

// NewHandler creates a new handler
func NewHandler(
	symbolSvc ports.SymbolService,
	snapshotSvc ports.SnapshotService,
	metricsSvc ports.MetricsService,
	exchange ports.ExchangeClient,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		symbolSvc:   symbolSvc,
		snapshotSvc: snapshotSvc,
		metricsSvc:  metricsSvc,
		exchange:    exchange,
		logger:      logger.With("component", "http_handler"),
	}
}

// Health returns service health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := "healthy"
	dbStatus := "healthy"
	exchangeStatus := "healthy"

	// Check exchange connectivity (with timeout)
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.exchange.Ping(checkCtx); err != nil {
		exchangeStatus = "unhealthy"
		status = "degraded"
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   status,
		"database": dbStatus,
		"exchange": exchangeStatus,
	})
}

// ListSymbols returns all tracked symbols
func (h *Handler) ListSymbols(w http.ResponseWriter, r *http.Request) {
	symbols, err := h.symbolSvc.ListSymbols(r.Context())
	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Extract symbol names for simpler response
	symbolNames := make([]string, len(symbols))
	for i, s := range symbols {
		symbolNames[i] = s.Name
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"symbols": symbolNames,
	})
}

// CreateSymbolRequest represents the request body for creating a symbol
type CreateSymbolRequest struct {
	Symbol string `json:"symbol"`
}

// CreateSymbol adds a new symbol to track
func (h *Handler) CreateSymbol(w http.ResponseWriter, r *http.Request) {
	var req CreateSymbolRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	symbol, err := h.symbolSvc.AddSymbol(r.Context(), req.Symbol)
	if err != nil {
		// Check if symbol already exists - return 200 instead of error
		if err == domain.ErrSymbolExists {
			existing, getErr := h.symbolSvc.GetSymbol(r.Context(), req.Symbol)
			if getErr == nil {
				respondJSON(w, http.StatusOK, existing)
				return
			}
		}
		handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, symbol)
}

// DeleteSymbol removes a tracked symbol
func (h *Handler) DeleteSymbol(w http.ResponseWriter, r *http.Request) {
	// Extract symbol from path
	symbol := r.PathValue("symbol")
	if symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	if err := h.symbolSvc.RemoveSymbol(r.Context(), symbol); err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PriceResponse represents a price in the API response
type PriceResponse struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Timestamp string `json:"ts"`
}

// GetPrices returns latest prices for specified symbols
func (h *Handler) GetPrices(w http.ResponseWriter, r *http.Request) {
	symbolsParam := r.URL.Query().Get("symbols")
	if symbolsParam == "" {
		respondError(w, http.StatusBadRequest, "symbols parameter is required")
		return
	}

	// Parse symbols
	symbols := strings.Split(symbolsParam, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	prices, missing, err := h.snapshotSvc.GetLatestPrices(r.Context(), symbols)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Format response
	priceResponses := make([]PriceResponse, len(prices))
	for i, p := range prices {
		priceResponses[i] = PriceResponse{
			Symbol:    p.Symbol,
			Price:     p.Price.String(),
			Timestamp: p.Timestamp.Format(time.RFC3339),
		}
	}

	response := map[string]interface{}{
		"prices": priceResponses,
	}

	if len(missing) > 0 {
		response["missing"] = missing
	}

	respondJSON(w, http.StatusOK, response)
}

// HistoryItem represents a history item in the API response
type HistoryItem struct {
	Price     string `json:"price"`
	Timestamp string `json:"ts"`
}

// GetHistory returns price history for a symbol
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol parameter is required")
		return
	}

	// Parse limit
	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	history, err := h.snapshotSvc.GetPriceHistory(r.Context(), symbol, limit)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Format response
	items := make([]HistoryItem, len(history))
	for i, h := range history {
		items[i] = HistoryItem{
			Price:     h.Price.String(),
			Timestamp: h.Timestamp.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"symbol": strings.ToUpper(symbol),
		"items":  items,
	})
}

// GetMetrics returns operational metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricsSvc.GetMetrics(r.Context())
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}
