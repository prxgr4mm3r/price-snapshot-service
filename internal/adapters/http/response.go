package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/prxgr4mmer/price-snapshot-service/internal/domain"
)

// Response helpers for consistent JSON responses

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// respondJSON sends a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// respondErrorWithCode sends an error response with an error code
func respondErrorWithCode(w http.ResponseWriter, status int, message, code string) {
	respondJSON(w, status, ErrorResponse{Error: message, Code: code})
}

// handleDomainError maps domain errors to HTTP responses
func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidSymbol):
		respondErrorWithCode(w, http.StatusBadRequest, "invalid symbol format", "INVALID_SYMBOL")

	case errors.Is(err, domain.ErrSymbolNotFound):
		respondErrorWithCode(w, http.StatusNotFound, "symbol not found", "SYMBOL_NOT_FOUND")

	case errors.Is(err, domain.ErrSymbolExists):
		respondErrorWithCode(w, http.StatusConflict, "symbol already exists", "SYMBOL_EXISTS")

	case errors.Is(err, domain.ErrSnapshotNotFound):
		respondErrorWithCode(w, http.StatusNotFound, "snapshot not found", "SNAPSHOT_NOT_FOUND")

	case errors.Is(err, domain.ErrExchangeUnavailable):
		respondErrorWithCode(w, http.StatusServiceUnavailable, "exchange service unavailable", "EXCHANGE_UNAVAILABLE")

	case errors.Is(err, domain.ErrRateLimited):
		respondErrorWithCode(w, http.StatusTooManyRequests, "rate limited by exchange", "RATE_LIMITED")

	case errors.Is(err, domain.ErrInvalidResponse):
		respondErrorWithCode(w, http.StatusBadGateway, "invalid response from exchange", "INVALID_EXCHANGE_RESPONSE")

	case errors.Is(err, domain.ErrDatabaseConnection):
		respondErrorWithCode(w, http.StatusServiceUnavailable, "database connection error", "DATABASE_ERROR")

	default:
		respondErrorWithCode(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
