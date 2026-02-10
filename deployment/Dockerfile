# ============================================
# Universal Dockerfile for all Go services
# ============================================
# Используется для: currency-api, currency-worker, gateway, auth-service

FROM golang:1.25.1-alpine AS builder

WORKDIR /build

ARG SERVICE_DIR
COPY ${SERVICE_DIR}/go.mod ${SERVICE_DIR}/go.sum ./
RUN go mod download

COPY ${SERVICE_DIR} ./

ARG BUILD_TARGET
ARG BINARY_NAME=app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/${BINARY_NAME} \
    ${BUILD_TARGET}

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# Создаём пользователя (опционально, для production)
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

WORKDIR /app

ARG BINARY_NAME=app
COPY --from=builder --chown=appuser:appgroup /app/${BINARY_NAME} /app/service

USER appuser

EXPOSE 8080

CMD ["/app/service"]
