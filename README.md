# Currency Exchange Service

Сервис предоставляет ежедневные курсы валют с поддержкой кросс-конвертации (любая валюта → любая валюта).

[![Go](https://img.shields.io/badge/Go-1.21-blue.svg)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-green.svg)](https://postgresql.org)
[![API](https://img.shields.io/badge/API-REST-blue.svg)](http://localhost:8082/health)

## Overview
Система состоит из:
* Currency Service — загрузка курсов из API, хранение в PostgreSQL, REST API
* Worker — автоматическое обновление курсов (cron 00:00)
Возможности:
* 166 валют с помощью [ExchangeRate-API](https://app.exchangerate-api.com/dashboard)
* Кросс-конвертация через RUB (10+ тысяч пар валют из 166 записей в БД)
* История курсов с указанием выбранного периода

## Services
Currency Service:
* HTTP REST API для получения кура и истории
* PostgreSQL
* Worker обновляется ежедневно

## Deployment & Setup
Технические требования:
* Go 1.21+
* PostgreSQL 15+

Запуск:
1. Клонировать

  ```git clone https://github.com/alfascuf/currency-service.git```

  ```cd currency-service```

2. Настроить .env и отредактировать (ключ, БД)

  ```cp .env.example .env```

3. Запуск

  ```go mod tidy```

  ```go run ./cmd/currency/main.go ```

## API-endpoints

Сервис доступен по: ```http://localhost:8082```

| Endpoint              | Параметры                          | Пример                                                                                                          |
| --------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| /health               | -                                  | ```curl http://localhost:8082/health```                                                                              |
| /api/v1/rates         | base, target, date=YYYY-MM-DD      | ```curl "http://localhost:8082/api/v1/rates?base=USD&target=EUR&date=2026-01-26"```                                  |
| /api/v1/rates/history | base, target, start_date, end_date | ```curl "http://localhost:8082/api/v1/rates/history?base=EUR&target=GBP&start_date=2026-01-25&end_date=2026-01-26"``` |

Например: 

Запрос:

  ```curl "http://localhost:8082/api/v1/rates?base=USD&target=EUR&date=2026-01-26"```

Вывод:

```{
  "base": "USD",
  "target": "EUR",
  "rate": 0.908,
  "date": "2026-01-26"
}
```
