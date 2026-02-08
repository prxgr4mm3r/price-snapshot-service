# Portfolio: Crypto Snapshot Service

A clean-room pet project demonstrating production-grade Go development practices for cryptocurrency price tracking.

## What This Demonstrates

### Software Architecture

- **Clean Architecture / Hexagonal Architecture**: Clear separation between domain, ports, adapters, and services
- **Dependency Injection**: Manual DI without frameworks, all dependencies injected via interfaces
- **SOLID Principles**: Single responsibility, interface segregation, dependency inversion

### Go Best Practices

- **Idiomatic Go**: Error handling, context propagation, struct embedding
- **Standard Library Usage**: `net/http` for routing (Go 1.22 enhanced mux), `log/slog` for structured logging
- **Concurrency Patterns**: Goroutines, channels, context cancellation, graceful shutdown
- **Testing**: Table-driven tests, mocks, HTTP test utilities

### Production Readiness

- **Observability**: Structured JSON logging, operational metrics endpoint
- **Resilience**: Retry with exponential backoff, connection pooling, timeout handling
- **DevOps**: Docker multi-stage build, Docker Compose, GitHub Actions CI
- **Configuration**: Environment-based config with validation

## Design Decisions

### 1. PostgreSQL over SQLite

**Decision**: Use PostgreSQL instead of SQLite.

**Rationale**:
- Better demonstrates real-world database skills
- Connection pooling for concurrent access
- NUMERIC type for precise financial calculations
- Production-ready with Docker Compose

### 2. No ORM

**Decision**: Use raw SQL with pgx driver.

**Rationale**:
- Full control over queries
- Better performance understanding
- No magic - explicit is better than implicit
- Standard library compatible (database/sql patterns)

### 3. Manual Dependency Injection

**Decision**: Wire dependencies manually in main.go instead of using a DI framework.

**Rationale**:
- Compile-time safety
- Explicit dependency graph
- No reflection magic
- Standard Go idiom

### 4. Interface-First Design

**Decision**: Define interfaces in `ports/` package, implement in `adapters/`.

**Rationale**:
- Easy to mock for testing
- Swap implementations without changing business logic
- Clear boundaries between layers
- Follows Dependency Inversion Principle

### 5. Structured Logging with slog

**Decision**: Use Go 1.21+ `log/slog` package.

**Rationale**:
- Standard library (no external dependencies)
- Structured JSON output for log aggregation
- Leveled logging
- Context-aware

## Edge Cases Handled

### API Layer

- **Empty symbol parameter**: Returns 400 Bad Request with clear error message
- **Invalid symbol format**: Validates uppercase alphanumeric, 2-20 chars
- **Symbol not on exchange**: Verifies with Binance before adding
- **Duplicate symbol**: Returns 200 OK with existing data (idempotent)
- **Missing symbols in price query**: Returns partial results with "missing" array

### External API

- **Rate limiting (429)**: Retries with exponential backoff
- **Server errors (5xx)**: Retries with backoff, marks exchange as unhealthy
- **Timeout**: Configurable timeout with context cancellation
- **Network errors**: Wrapped as retryable errors

### Database

- **Connection pool exhaustion**: Configured max connections and idle timeout
- **Transaction failures**: Proper rollback on batch insert errors
- **Missing data**: Returns appropriate 404 or empty arrays

### Lifecycle

- **SIGINT/SIGTERM**: Graceful shutdown with ordered cleanup
- **Context cancellation**: Propagated to all goroutines
- **Poller interruption**: Completes current poll before stopping

## What I Would Add Next

### Short-term Improvements

1. **WebSocket Support**: Real-time price updates via Binance WebSocket API
2. **Rate Limiting**: Token bucket rate limiter for API endpoints
3. **Caching**: Redis cache for frequently accessed prices
4. **Pagination**: Cursor-based pagination for history endpoint

### Medium-term Features

1. **Multiple Exchanges**: Abstract exchange interface, add Coinbase, Kraken
2. **Price Alerts**: Threshold-based notifications
3. **Data Retention**: Automatic pruning of old snapshots
4. **Prometheus Metrics**: /metrics endpoint with Prometheus format

### Long-term Enhancements

1. **Kubernetes Deployment**: Helm charts, horizontal pod autoscaling
2. **Event Sourcing**: Store all state changes as events
3. **GraphQL API**: Alternative to REST for flexible queries
4. **Admin Dashboard**: Web UI for configuration and monitoring

## Technical Metrics

- **Lines of Code**: ~2,500 (excluding tests)
- **Test Coverage**: ~70% (critical paths covered)
- **Build Time**: ~10s (with Docker multi-stage)
- **Binary Size**: ~15MB (stripped)
- **Startup Time**: <1s (excluding migrations)

## Key Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Entry point, dependency wiring |
| `internal/ports/*.go` | Interface definitions |
| `internal/adapters/binance/client.go` | Binance API with retry |
| `internal/worker/poller.go` | Background price polling |
| `internal/adapters/http/handlers.go` | HTTP request handlers |

## Lessons Learned

1. **Go's simplicity is a feature**: Explicit error handling and manual DI lead to more maintainable code
2. **Interfaces enable testing**: Well-defined interfaces make unit testing straightforward
3. **Context is essential**: Proper context propagation enables graceful shutdown and timeout handling
4. **Standard library is powerful**: Go 1.22 routing eliminates need for external routers

---

*This project was designed as a portfolio piece to demonstrate production-grade Go development skills.*
