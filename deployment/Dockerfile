# ============================================
# Universal Dockerfile for all Go services
# ============================================
# Используется для: currency-api, currency-worker, gateway, auth-service

FROM golang:1.25-alpine AS builder

WORKDIR /build

# Установка зависимостей для сборки
RUN apk add --no-cache git

# ============================================
# ВАЖНО: Копируем go.mod/go.sum из КОРНЯ проекта
# ============================================
COPY go.mod go.sum ./
RUN go mod download

# ============================================
# Копируем ВСЁ содержимое проекта
# Это нужно для доступа к pkg/grpc/pb (общие proto-файлы)
# ============================================
COPY . .

# Сборка бинарника из конкретного сервиса
ARG SERVICE_DIR
ARG BUILD_TARGET
ARG BINARY_NAME=app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/${BINARY_NAME} \
    ./${SERVICE_DIR}/${BUILD_TARGET}

# ============================================
# Runtime stage
# ============================================
FROM alpine:3.19

# Установка зависимостей (wget для healthcheck)
RUN apk --no-cache add ca-certificates tzdata wget

# Создаём пользователя для безопасности
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

WORKDIR /app

ARG BINARY_NAME=app
COPY --from=builder --chown=appuser:appgroup /app/${BINARY_NAME} /app/service

USER appuser

EXPOSE 8080
EXPOSE 50051

CMD ["/app/service"]
