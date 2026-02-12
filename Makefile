.PHONY: help lint-all lint-fix-all test-all build-all clean-all docker-up docker-down docker-rebuild

help:
	@echo "=== Currency Exchange Project ==="
	@echo ""
	@echo "Available commands:"
	@echo "  make lint-all        - Run linter for all services"
	@echo "  make lint-fix-all    - Run linter with auto-fix for all services"
	@echo "  make test-all        - Run tests for all services"
	@echo "  make build-all       - Build all services"
	@echo "  make clean-all       - Clean all services"
	@echo ""
	@echo "Docker commands:"
	@echo "  make docker-up       - Start all services with docker-compose"
	@echo "  make docker-down     - Stop all services"
	@echo "  make docker-rebuild  - Rebuild and restart all services"
	@echo ""
	@echo "Service-specific:"
	@echo "  make lint-currency   - Lint currency-service only"
	@echo "  make lint-gateway    - Lint gateway only"
	@echo "  make lint-auth       - Lint auth-service only"

# ============================================
# Lint commands
# ============================================
lint-all: lint-currency lint-gateway lint-auth
	@echo "All services linted successfully!"

lint-currency:
	@echo "Linting currency-service..."
	@cd currency-service && make lint

lint-gateway:
	@echo "Linting gateway..."
	@cd gateway && make lint

lint-auth:
	@echo "Linting auth-service..."
	@cd auth-service && make lint

# ============================================
# Lint-fix commands
# ============================================
lint-fix-all: lint-fix-currency lint-fix-gateway lint-fix-auth
	@echo "All services auto-fixed!"

lint-fix-currency:
	@echo "Fixing currency-service..."
	@cd currency-service && make lint-fix

lint-fix-gateway:
	@echo "Fixing gateway..."
	@cd gateway && make lint-fix

lint-fix-auth:
	@echo "Fixing auth-service..."
	@cd auth-service && make lint-fix

# ============================================
# Test commands
# ============================================
test-all: test-currency test-gateway test-auth
	@echo "All tests passed!"

test-currency:
	@echo "Testing currency-service..."
	@cd currency-service && make test

test-gateway:
	@echo "Testing gateway..."
	@cd gateway && make test

test-auth:
	@echo "Testing auth-service..."
	@cd auth-service && make test

# ============================================
# Build commands
# ============================================
build-all: build-currency build-gateway build-auth
	@echo "All services built successfully!"

build-currency:
	@echo "Building currency-service..."
	@cd currency-service && make build-api && make build-worker

build-gateway:
	@echo " Building gateway..."
	@cd gateway && make build

build-auth:
	@echo "Building auth-service..."
	@cd auth-service && make build

# ============================================
# Clean commands
# ============================================
clean-all:
	@echo "Cleaning all services..."
	@cd currency-service && make clean || true
	@cd gateway && make clean || true
	@cd auth-service && make clean || true
	@echo "All services cleaned!"

# ============================================
# Docker commands
# ============================================
docker-up:
	@echo "Starting all services..."
	docker-compose up --build -d
	@echo "All services started!"

docker-down:
	@echo "Stopping all services..."
	docker-compose down
	@echo "All services stopped!"

docker-rebuild:
	@echo "Rebuilding all services..."
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d
	@echo "All services rebuilt and started!"

docker-logs:
	docker-compose logs -f

docker-ps:
	docker-compose ps

# ============================================
# Quality checks (lint + test)
# ============================================
check-all: lint-all test-all
	@echo "All quality checks passed!"

.PHONY: test
test:
	cd currency-service && go test -v ./internal/...

.PHONY: test-coverage
test-coverage:
	cd currency-service && go test -v -coverprofile=coverage.out ./internal/...
	cd currency-service && go tool cover -html=coverage.out -o coverage.html
	@echo Coverage report generated: currency-service/coverage.html

.PHONY: test-short
test-short:
	cd currency-service && go test -short ./internal/...
