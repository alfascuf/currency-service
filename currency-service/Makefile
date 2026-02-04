.PHONY: help lint lint-fix test build-api build-worker docker-build docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make lint-fix     - Run golangci-lint with auto-fix"
	@echo "  make test         - Run tests"
	@echo "  make build-api    - Build API binary"
	@echo "  make build-worker - Build Worker binary"
	@echo "  make docker-build - Build Docker images"
	@echo "  make docker-up    - Start Docker Compose"
	@echo "  make docker-down  - Stop Docker Compose"

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test -v -race -coverprofile=coverage.txt ./...

build-api:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/api/main.go

build-worker:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/worker ./cmd/worker/main.go

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-rebuild:
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d
