# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build migrator binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /migrator ./cmd/migrator

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /migrator .

# Run migrations
CMD ["./migrator", "-cmd=up"]
