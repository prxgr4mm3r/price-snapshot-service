.PHONY: build run test lint fmt clean docker-build docker-up docker-down migrate-up migrate-down help

# Build variables
BINARY_NAME=snapshot-service
BUILD_DIR=./bin
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Docker variables
DOCKER_IMAGE=price-snapshot-service
DOCKER_TAG=latest

# Database variables
DATABASE_URL?=postgres://postgres:postgres@localhost:5432/snapshots?sslmode=disable

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## build: Build the application binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

## run: Run the application locally
run: build
	@echo "Starting $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run with hot reload (requires air)
dev:
	@which air > /dev/null || go install github.com/air-verse/air@latest
	air

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

## test-short: Run tests without race detector (faster)
test-short:
	go test -v -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

## fmt: Format code
fmt:
	@echo "Formatting code..."
	gofmt -w $(GO_FILES)

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## docker-up: Start all services with Docker Compose
docker-up:
	@echo "Starting services..."
	docker-compose up -d

## docker-down: Stop all services
docker-down:
	@echo "Stopping services..."
	docker-compose down

## docker-logs: View logs
docker-logs:
	docker-compose logs -f

## docker-ps: Show running containers
docker-ps:
	docker-compose ps

## migrate-up: Run database migrations
migrate-up:
	@echo "Running migrations..."
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DATABASE_URL)" up

## migrate-down: Rollback all migrations
migrate-down:
	@echo "Rolling back migrations..."
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DATABASE_URL)" down

## migrate-create: Create a new migration (usage: make migrate-create name=migration_name)
migrate-create:
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate create -ext sql -dir migrations -seq $(name)

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## check: Run all checks (lint, test)
check: lint test

## all: Build everything
all: deps lint test build
