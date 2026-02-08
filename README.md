# Crypto Snapshot Service

A production-ready Go service that tracks cryptocurrency prices from Binance and stores historical snapshots in PostgreSQL.

## Features

- Track multiple cryptocurrency symbols (e.g., BTCUSDT, ETHUSDT)
- Automatic price polling at configurable intervals
- Historical price data with configurable retention
- RESTful HTTP API for management and queries
- Operational metrics endpoint
- Graceful shutdown with proper cleanup
- Docker and Docker Compose support
- CI/CD with GitHub Actions

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services (PostgreSQL + App)
docker-compose up -d

# Check health
curl http://localhost:8080/health

# Add a symbol to track
curl -X POST http://localhost:8080/symbols \
  -H "Content-Type: application/json" \
  -d '{"symbol": "BTCUSDT"}'

# View logs
docker-compose logs -f app
```

### Local Development

```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run migrations
make migrate-up

# Run the service
make run
```

## API Reference

### Health Check

```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "database": "healthy",
  "exchange": "healthy"
}
```

### Symbols Management

#### List Tracked Symbols
```bash
GET /symbols
```

Response:
```json
{
  "symbols": ["BTCUSDT", "ETHUSDT"]
}
```

#### Add Symbol
```bash
POST /symbols
Content-Type: application/json

{"symbol": "BTCUSDT"}
```

Response: `201 Created` (new) or `200 OK` (exists)

#### Remove Symbol
```bash
DELETE /symbols/{symbol}
```

Response: `204 No Content` or `404 Not Found`

### Price Queries

#### Get Latest Prices
```bash
GET /prices?symbols=BTCUSDT,ETHUSDT
```

Response:
```json
{
  "prices": [
    {"symbol": "BTCUSDT", "price": "43123.45", "ts": "2024-01-15T10:30:00Z"},
    {"symbol": "ETHUSDT", "price": "2345.67", "ts": "2024-01-15T10:30:00Z"}
  ],
  "missing": []
}
```

#### Get Price History
```bash
GET /history?symbol=BTCUSDT&limit=100
```

Response:
```json
{
  "symbol": "BTCUSDT",
  "items": [
    {"price": "43123.45", "ts": "2024-01-15T10:30:00Z"},
    {"price": "43100.00", "ts": "2024-01-15T10:29:00Z"}
  ]
}
```

### Operational Metrics

```bash
GET /metrics
```

Response:
```json
{
  "uptime_seconds": 3600,
  "tracked_symbols": 5,
  "active_symbols": 5,
  "total_snapshots": 1000,
  "last_poll_time": "2024-01-15T10:30:00Z",
  "last_poll_duration_ms": 150,
  "poll_success_count": 120,
  "poll_error_count": 2,
  "database_status": "healthy",
  "exchange_status": "healthy"
}
```

## Configuration

Environment variables with defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/snapshots?sslmode=disable` | PostgreSQL connection URL |
| `POLLER_INTERVAL` | `30s` | Price polling interval |
| `EXCHANGE_TIMEOUT` | `10s` | Binance API timeout |
| `EXCHANGE_MAX_RETRIES` | `3` | Max retries for API calls |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |

## Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/
│   ├── adapters/        # Infrastructure implementations
│   │   ├── binance/     # Binance API client
│   │   ├── http/        # HTTP handlers & server
│   │   └── postgres/    # Database repositories
│   ├── config/          # Configuration management
│   ├── domain/          # Core business entities
│   ├── ports/           # Interface definitions
│   ├── services/        # Business logic
│   └── worker/          # Background workers
├── migrations/          # SQL migrations
├── pkg/retry/           # Reusable retry logic
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Development

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16+ (or use Docker)

### Available Commands

```bash
make build          # Build the binary
make run            # Build and run locally
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make docker-build   # Build Docker image
make docker-up      # Start services with Docker Compose
make migrate-up     # Run database migrations
make migrate-down   # Rollback migrations
```

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test -v ./internal/adapters/binance/...
```

## Architecture

The service follows Clean Architecture / Hexagonal Architecture principles:

- **Domain Layer**: Core business entities with no external dependencies
- **Ports Layer**: Interface definitions for all I/O operations
- **Adapters Layer**: Implementations of ports (HTTP, PostgreSQL, Binance)
- **Services Layer**: Business logic orchestration
- **Worker Layer**: Background processing

Key design decisions:

1. **Interface-based design**: All dependencies are injected via interfaces
2. **Manual DI**: No DI framework, explicit wiring in main.go
3. **Graceful shutdown**: Proper cleanup of all resources on termination
4. **Retry with backoff**: Exponential backoff for external API calls
5. **Connection pooling**: PostgreSQL connection pool with configurable limits

## License

MIT
