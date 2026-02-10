.PHONY: help lint lint-fix test build docker-build docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make lint-fix     - Run golangci-lint with auto-fix"
	@echo "  make test         - Run tests"
	@echo "  make build        - Build Gateway binary"
	@echo "  make docker-build - Build Docker image"

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test -v -race -coverprofile=coverage.txt ./...

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/gateway ./cmd/gateway/gateway.go

docker-build:
	docker build -f deployment/gateway.dockerfile -t gateway:latest .

clean:
	rm -rf bin/
	rm -f coverage.txt
