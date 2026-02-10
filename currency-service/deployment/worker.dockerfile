# syntax=docker/dockerfile:1

# ============================================
# STAGE 1: Build Stage
# ============================================
FROM golang:1.25-alpine AS builder

# Устанавливаем необходимые инструменты для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей для кэширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник воркера
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/bin/worker \
    ./cmd/worker/main.go

# ============================================
# STAGE 2: Runtime Stage
# ============================================
FROM alpine:3.19

# Устанавливаем сертификаты и timezone данные
RUN apk --no-cache add ca-certificates tzdata

# Создаём непривилегированного пользователя
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder --chown=appuser:appgroup /app/bin/worker /app/worker

# Переключаемся на непривилегированного пользователя
USER appuser

# Запуск воркера
CMD ["/app/worker"]
