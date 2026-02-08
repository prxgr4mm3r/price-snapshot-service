package http

import (
	"log/slog"
	"net/http"
)

// NewRouter creates the HTTP router with all routes
func NewRouter(h *Handler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", h.Health)

	// Symbols management
	mux.HandleFunc("GET /symbols", h.ListSymbols)
	mux.HandleFunc("POST /symbols", h.CreateSymbol)
	mux.HandleFunc("DELETE /symbols/{symbol}", h.DeleteSymbol)

	// Prices
	mux.HandleFunc("GET /prices", h.GetPrices)

	// History
	mux.HandleFunc("GET /history", h.GetHistory)

	// Metrics
	mux.HandleFunc("GET /metrics", h.GetMetrics)

	// Apply middleware chain (order matters: outer -> inner)
	var handler http.Handler = mux
	handler = ContentTypeMiddleware(handler)
	handler = CORSMiddleware(handler)
	handler = RecoveryMiddleware(logger)(handler)
	handler = LoggingMiddleware(logger)(handler)

	return handler
}
