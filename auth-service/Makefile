.PHONY: help lint lint-fix test build docker-build

help:
	@echo "Available commands:"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make lint-fix     - Run golangci-lint with auto-fix"
	@echo "  make test         - Run tests"
	@echo "  make build        - Build Auth-service binary"
	@echo "  make docker-build - Build Docker image"

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test -v -race -coverprofile=coverage.txt ./...

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/auth-service main.go

docker-build:
	docker build -f Dockerfile -t auth-service:latest .

clean:
	rm -rf bin/
	rm -f coverage.txt
