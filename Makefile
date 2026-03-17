
.PHONY: help build build-api build-frontend test test-coverage test-integration test-all test-e2e test-load lint clean tidy verify fmt \
	migrate-up migrate-down migrate-up-prod migrate-down-prod compose-up compose-down redeploy logs dev dev-api dev-frontend dev-indexer \
	docker-build docker-build-api docker-build-indexer docker-build-frontend docker-size

# Default target
.DEFAULT_GOAL := help

# Docker Compose configuration
DOCKER_COMPOSE_FILE ?= docker-compose.yml
MIGRATE_ENV_FILE ?= ./configs/dev.env

## help: Display this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build all Go binaries
build:
	@echo "Building Go binaries..."
	go build -o bin/api ./cmd/api
	go build -o bin/indexer ./cmd/indexer
	go build -o bin/backfill ./cmd/backfill

## build-api: Build only the API binary
build-api:
	@echo "Building API binary..."
	go build -o bin/api ./cmd/api

## build-frontend: Build the frontend application
build-frontend:
	@echo "Building frontend..."
	npm run build

## test: Run all tests (Go and frontend if available)
test:
	@echo "Running Go tests..."
	go test -v -race -cover ./...
	@echo "Running frontend tests (if defined in package.json)..."
	npm run test --if-present

## test-coverage: Run tests with coverage report (Go + frontend)
test-coverage:
	@echo "Running Go tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Go coverage report: coverage.html"
	@echo "Running frontend tests with coverage..."
	cd web && npm run test:coverage
	@echo "Frontend coverage report: web/coverage/index.html"

## test-integration: Run integration tests (requires Docker)
test-integration:
	@echo "Running integration tests..."
	go test -tags=integration -race -v ./...

## test-all: Run all tests (unit + integration + E2E + load)
test-all: test test-integration test-e2e test-load
	@echo "All tests complete."

## test-e2e: Run E2E tests for streaming functionality
test-e2e:
	@echo "Running E2E tests..."
	npm run test:e2e

## test-load: Run k6 load tests for streaming
test-load:
	@echo "Running load tests..."
	npm run test:load

## lint: Run linters
lint:
	@echo "Running Go linters..."
	go vet ./...
	@echo "Running frontend linters..."
	npm run lint --if-present

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/
	rm -rf coverage.out

## tidy: Tidy Go modules
tidy:
	go mod tidy

## verify: Verify Go modules
verify:
	go mod verify

## fmt: Format Go code
fmt:
	go fmt ./...

## migrate-up: Apply all pending database migrations (default env: configs/dev.env)
migrate-up:
	@test -f $(MIGRATE_ENV_FILE) || (echo "Error: $(MIGRATE_ENV_FILE) not found" && exit 1)
	@set -a && . $(MIGRATE_ENV_FILE) && set +a && ./scripts/migrate.sh up

## migrate-down: Rollback the last database migration (default env: configs/dev.env)
migrate-down:
	@test -f $(MIGRATE_ENV_FILE) || (echo "Error: $(MIGRATE_ENV_FILE) not found" && exit 1)
	@set -a && . $(MIGRATE_ENV_FILE) && set +a && ./scripts/migrate.sh down 1

## migrate-up-prod: Apply all pending migrations using deploy/.env
migrate-up-prod: MIGRATE_ENV_FILE=./deploy/.env
migrate-up-prod: export MIGRATE_USE_DOCKER=1
migrate-up-prod: export MIGRATE_DOCKER_NETWORK=subcults-internal
migrate-up-prod: migrate-up

## migrate-down-prod: Rollback last migration using deploy/.env
migrate-down-prod: MIGRATE_ENV_FILE=./deploy/.env
migrate-down-prod: export MIGRATE_USE_DOCKER=1
migrate-down-prod: export MIGRATE_DOCKER_NETWORK=subcults-internal
migrate-down-prod: migrate-down

## compose-up: Start all services with Docker Compose
compose-up:
	@test -f $(DOCKER_COMPOSE_FILE) || (echo "Error: $(DOCKER_COMPOSE_FILE) not found" && exit 1)
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d

## compose-down: Stop all services with Docker Compose
compose-down:
	@test -f $(DOCKER_COMPOSE_FILE) || (echo "Error: $(DOCKER_COMPOSE_FILE) not found" && exit 1)
	docker compose -f $(DOCKER_COMPOSE_FILE) down

## redeploy: Rebuild, migrate, and recreate production services via deploy script
redeploy:
	./scripts/deploy.sh --redeploy

## dev: Run API and frontend development servers (requires: compose-up, database migrations)
dev:
	@echo "Starting development servers..."
	@echo "Ensure you've run 'make compose-up' and 'make migrate-up' first"
	@trap 'kill %1 %2' EXIT; \
	make dev-api & \
	make dev-frontend & \
	wait

## dev-api: Run API server with hot reload (requires: compose-up, migrations)
dev-api:
	@set -a && . ./configs/dev.env && set +a && \
	echo "Starting API server on http://$$HOST:$$PORT (from configs/dev.env)" && \
	go run ./cmd/api

## dev-frontend: Run frontend development server
dev-frontend:
	@echo "Starting frontend dev server..."
	@cd web && npm run dev

## dev-indexer: Run Jetstream indexer (requires: compose-up, migrations)
dev-indexer:
	@set -a && . ./configs/dev.env && set +a && \
	echo "Starting Jetstream indexer..." && \
	go run ./cmd/indexer

## logs: Stream Docker Compose logs from all services
logs:
	@test -f $(DOCKER_COMPOSE_FILE) || (echo "Error: $(DOCKER_COMPOSE_FILE) not found" && exit 1)
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f

## logs-api: Stream logs from API service only
logs-api:
	@test -f $(DOCKER_COMPOSE_FILE) || (echo "Error: $(DOCKER_COMPOSE_FILE) not found" && exit 1)
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f api

## logs-postgres: Stream logs from PostgreSQL service only
logs-postgres:
	@test -f $(DOCKER_COMPOSE_FILE) || (echo "Error: $(DOCKER_COMPOSE_FILE) not found" && exit 1)
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f postgres

# =============================================================================
# Docker Image Builds
# =============================================================================

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
REGISTRY ?= ghcr.io/subculture-collective

## docker-build: Build all production Docker images
docker-build: docker-build-api docker-build-indexer docker-build-frontend

## docker-build-api: Build production API Docker image
docker-build-api:
	@echo "Building API image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(REGISTRY)/subcults-api:$(VERSION) \
		-t $(REGISTRY)/subcults-api:latest \
		-f Dockerfile.api .

## docker-build-indexer: Build production Indexer Docker image
docker-build-indexer:
	@echo "Building Indexer image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(REGISTRY)/subcults-indexer:$(VERSION) \
		-t $(REGISTRY)/subcults-indexer:latest \
		-f Dockerfile.indexer .

## docker-build-frontend: Build production Frontend Docker image
docker-build-frontend:
	@echo "Building Frontend image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		-t $(REGISTRY)/subcults-frontend:$(VERSION) \
		-t $(REGISTRY)/subcults-frontend:latest \
		-f Dockerfile.frontend .

## docker-size: Show Docker image sizes
docker-size:
	@echo "Image sizes:"
	@docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep subcults || echo "No subcults images found. Run 'make docker-build' first."
