# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Переменная для Go прокси
ENV GOPROXY=https://proxy.golang.org,direct

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway ./cmd/gateway/main.go

# Stage 2: Runtime
FROM alpine:latest

# Устанавливаем CA сертификаты для HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /gateway .

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./gateway"]
